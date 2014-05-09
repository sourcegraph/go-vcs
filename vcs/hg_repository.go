package vcs

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.tools/godoc/vfs"
	"github.com/knieriem/hgo"
	hg_changelog "github.com/knieriem/hgo/changelog"
	hg_revlog "github.com/knieriem/hgo/revlog"
	hg_store "github.com/knieriem/hgo/store"
)

type LocalHgRepository struct {
	dir     string
	u       *hgo.Repository
	st      *hg_store.Store
	cl      *hg_revlog.Index
	allTags *hgo.Tags
}

func OpenLocalHgRepository(dir string) (*LocalHgRepository, error) {
	r, err := hgo.OpenRepository(dir)
	if err != nil {
		return nil, err
	}

	st := r.NewStore()
	cl, err := st.OpenChangeLog()
	if err != nil {
		return nil, err
	}

	globalTags, allTags := r.Tags()
	globalTags.Sort()
	allTags.Sort()

	return &LocalHgRepository{dir, r, st, cl, allTags}, nil
}

func (r *LocalHgRepository) ResolveRevision(spec string) (CommitID, error) {
	rs := r.parseRevisionSpec(spec)
	rec, err := rs.Lookup(r.cl)
	if err != nil {
		return "", err
	}
	return CommitID(hex.EncodeToString(rec.Id())), nil
}

func (r *LocalHgRepository) ResolveTag(name string) (CommitID, error) {
	return r.ResolveRevision(name)
}

func (r *LocalHgRepository) FileSystem(at CommitID) (vfs.FileSystem, error) {
	rec, err := hg_revlog.NodeIdRevSpec(at).Lookup(r.cl)
	if err != nil {
		return nil, err
	}

	return &localHgFS{
		dir:  r.dir,
		at:   hg_revlog.FileRevSpec(rec.FileRev()),
		repo: r.u,
		st:   r.st,
		cl:   r.cl,
		fb:   hg_revlog.NewFileBuilder(),
	}, nil
}

func (r *LocalHgRepository) parseRevisionSpec(s string) hg_revlog.RevisionSpec {
	if s == "" {
		s = "tip"
		// TODO(sqs): determine per-repository default branch name (not always "default"?)
	}
	if s == "tip" {
		return hg_revlog.TipRevSpec{}
	}
	if s == "null" {
		return hg_revlog.NullRevSpec{}
	}
	if id, ok := r.allTags.IdByName[s]; ok {
		s = id
	} else if i, err := strconv.Atoi(s); err == nil {
		return hg_revlog.FileRevSpec(i)
	}

	return hg_revlog.NodeIdRevSpec(s)
}

type localHgFS struct {
	dir  string
	at   hg_revlog.FileRevSpec
	repo *hgo.Repository
	st   *hg_store.Store
	cl   *hg_revlog.Index
	fb   *hg_revlog.FileBuilder
}

func (fs *localHgFS) manifestEntry(chgId hg_revlog.FileRevSpec, fileName string) (me *hg_store.ManifestEnt, err error) {
	m, err := fs.getManifest(chgId)
	if err != nil {
		return
	}
	me = m.Map()[fileName]
	if me == nil {
		err = errors.New("file does not exist in given revision")
	}
	return
}

func (fs *localHgFS) getManifest(chgId hg_revlog.FileRevSpec) (m hg_store.Manifest, err error) {
	rec, err := chgId.Lookup(fs.cl)
	if err != nil {
		return
	}
	c, err := hg_changelog.BuildEntry(rec, fs.fb)
	if err != nil {
		return
	}

	// st := fs.repo.NewStore()
	mlog, err := fs.st.OpenManifests()
	if err != nil {
		return nil, err
	}

	rec2, err := mlog.LookupRevision(int(c.Linkrev), c.ManifestNode)
	if err != nil {
		return nil, err
	}

	return hg_store.BuildManifest(rec2, fs.fb)
}

func (fs *localHgFS) getEntry(path string) (*hg_revlog.Rec, *hg_store.ManifestEnt, error) {
	fileLog, err := fs.st.OpenRevlog(path)
	if err != nil {
		return nil, nil, err
	}

	rec, err := hg_revlog.LinkRevSpec{Rev: int(fs.at)}.Lookup(fileLog)
	if err != nil {
		return nil, nil, err
	}
	if rec.FileRev() == -1 {
		return nil, nil, hg_revlog.ErrRevisionNotFound
	}

	// Check for the file's existence using the manifest.
	ent, err := fs.manifestEntry(fs.at, path)
	if err != nil {
		return nil, nil, err
	}

	// compare hashes
	wantId, err := ent.Id()
	if err != nil {
		return nil, nil, err
	}
	if !wantId.Eq(rec.Id()) {
		return nil, nil, errors.New("manifest node id does not match file id")
	}

	if int(rec.Linkrev) == int(fs.at) {
		// The requested revision matches this record, which can be
		// used as a sign that the file exists. (TODO(sqs): original comments
		// say maybe this means the file is NOT existent yet? the word "not" is
		// not there but that seems to be a mistake.)
		return rec, ent, nil
	}

	if !rec.IsLeaf() {
		// There are other records that have the current record as a parent.
		// This means, the file was existent, no need to check the manifest.
		return rec, ent, nil
	}

	return rec, ent, nil
}

func (fs *localHgFS) Open(name string) (vfs.ReadSeekCloser, error) {
	rec, _, err := fs.getEntry(name)
	if err != nil {
		return nil, standardizeHgError(err)
	}

	data, err := fs.readFile(rec)
	if err != nil {
		return nil, err
	}
	return nopCloser{bytes.NewReader(data)}, nil
}

func (fs *localHgFS) readFile(rec *hg_revlog.Rec) ([]byte, error) {
	fb := hg_revlog.NewFileBuilder()
	return fb.Build(rec)
}

func (fs *localHgFS) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(path)
}

func (fs *localHgFS) Stat(path string) (os.FileInfo, error) {
	// TODO(sqs): follow symlinks (as Stat is required to do)
	rec, ent, err := fs.getEntry(path)
	if os.IsNotExist(err) {
		// check if path is a dir (dirs are not in hg's manifest, so we need to
		// hack around to get them).
		return fs.dirStat(path)
	}
	if err != nil {
		return nil, standardizeHgError(err)
	}

	fi := fs.fileInfo(ent)

	// read data to determine file size
	data, err := fs.readFile(rec)
	if err != nil {
		return nil, err
	}
	fi.size = int64(len(data))

	return fi, nil
}

// dirStat determines whether a directory exists at path by listing files
// underneath it. If it has files, then it's a directory. We must do it this way
// because hg doesn't track directories in the manifest.
func (fs *localHgFS) dirStat(path string) (os.FileInfo, error) {
	m, err := fs.getManifest(fs.at)
	if err != nil {
		return nil, err
	}

	dirPrefix := filepath.Clean(path) + "/"
	for _, e := range m {
		if strings.HasPrefix(e.FileName, dirPrefix) {
			return &fileInfo{
				name: filepath.Base(path),
				mode: os.ModeDir,
			}, nil
		}
	}

	return nil, os.ErrNotExist
}

func (fs *localHgFS) fileInfo(ent *hg_store.ManifestEnt) *fileInfo {
	var mode os.FileMode
	if ent.IsExecutable() {
		mode |= 0111 // +x
	}
	if ent.IsLink() {
		mode |= os.ModeSymlink
	}

	return &fileInfo{
		name: filepath.Base(ent.FileName),
		mode: mode,
	}
}

func (fs *localHgFS) ReadDir(path string) ([]os.FileInfo, error) {
	m, err := fs.getManifest(fs.at)
	if err != nil {
		return nil, err
	}

	var fis []os.FileInfo
	dirPrefix := filepath.Clean(path) + "/"
	for _, e := range m {
		if strings.HasPrefix(e.FileName, dirPrefix) {
			fis = append(fis, fs.fileInfo(&e))
		}
	}
	return fis, nil
}

func (fs *localHgFS) String() string {
	return fmt.Sprintf("local hg repository %s commit %s", fs.dir, fs.at)
}

func standardizeHgError(err error) error {
	if err == hg_revlog.ErrRevisionNotFound {
		return os.ErrNotExist
	}
	return err
}

type fileInfo struct {
	name  string
	mode  os.FileMode
	size  int64
	mtime time.Time
}

func (fi *fileInfo) Name() string       { return fi.name }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *fileInfo) ModTime() time.Time { return fi.mtime }
func (fi *fileInfo) IsDir() bool        { return fi.Mode().IsDir() }
func (fi *fileInfo) Sys() interface{}   { return nil }
