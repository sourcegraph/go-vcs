package vcs

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/knieriem/hgo"
	"github.com/knieriem/hgo/changelog"
	"github.com/knieriem/hgo/revlog"
	"github.com/knieriem/hgo/store"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type hg struct {
	cmd string
}

func (_ hg) ShortName() string { return "hg" }

var Hg VCS = hg{"hg"}

type hgRepo struct {
	dir string
	hg  *hg
}

func (hg hg) Clone(url, dir string) (Repository, error) {
	r := &hgRepo{dir, &hg}

	cmd := exec.Command("hg", "clone", "--", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), fmt.Sprintf("abort: destination '%s' is not empty", dir)) {
			return nil, os.ErrExist
		}
		return nil, fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}

	return r, nil
}

func (hg hg) Open(dir string) (Repository, error) {
	// TODO(sqs): check for .hg or bare repo
	if _, err := os.Stat(dir); err == nil {
		return &hgRepo{dir, &hg}, nil
	} else {
		return nil, err
	}
}

func (hg hg) CloneMirror(url, dir string) error {
	cmd := exec.Command("hg", "clone", "-U", "--", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), fmt.Sprintf("abort: destination '%s' is not empty", dir)) {
			return os.ErrExist
		}
		return fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (hg hg) UpdateMirror(dir string) error {
	cmd := exec.Command("hg", "pull")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (r *hgRepo) Dir() (dir string) {
	return r.dir
}

func (r *hgRepo) VCS() VCS {
	return r.hg
}

func (r *hgRepo) Download() error {
	cmd := exec.Command("hg", "pull")
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
	return nil
}

func (r *hgRepo) CommitLog() ([]*Commit, error) {
	cmd := exec.Command("hg", "log", `--template={node}\n{author|person}\n{author|email}\n{date|rfc3339date}\n\n{desc}\n\x00`)
	cmd.Dir = r.dir
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	commitEntries := bytes.Split(out, []byte{'\x00'})
	commitEntries = commitEntries[:len(commitEntries)-1] // hg log puts delimiter at end
	commits := make([]*Commit, len(commitEntries))
	for i, e := range commitEntries {
		if len(e) == 0 {
			continue
		}
		commit := new(Commit)
		parts := bytes.SplitN(e, []byte("\n\n"), 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unhandled hg commit entry: %q", string(e))
		}
		header, commitMsg := parts[0], parts[1]

		headers := bytes.Split(header, []byte{'\n'})
		commit.ID = string(headers[0])
		commit.AuthorName = string(headers[1])
		commit.AuthorEmail = string(headers[2])

		var err error
		commit.AuthorDate, err = time.Parse(time.RFC3339, string(headers[3]))
		if err != nil {
			return nil, err
		}

		commit.Message = strings.TrimSpace(string(commitMsg))
		commits[i] = commit
	}

	return commits, nil
}

func (r *hgRepo) CheckOut(rev string) (dir string, err error) {
	if rev == "" {
		rev = "default"
	}
	cmd := exec.Command("hg", "update", "-r", rev)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return r.dir, nil
	} else {
		return "", fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
}

// hgo is faster but doesn't always work. try it first.
func (r *hgRepo) ReadFileAtRevision(path string, rev string) ([]byte, FileType, error) {
	data, err := r.readFileAtRevisionHgo(path, rev)
	if err == nil {
		return data, File, err
	}
	return r.readFileAtRevisionHg(path, rev)
}

func (r *hgRepo) readFileAtRevisionHg(path string, rev string) ([]byte, FileType, error) {
	// if a dir, list its contents
	isDir := false
	if path == "" {
		path = "."
		isDir = true
	}
	cmd := exec.Command("hg", "locate", "-r", rev, "-I", path)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		if isDir || strings.HasPrefix(string(out), path+"/") {
			files := strings.Split(string(out), "\n")
			var filelist []string
			cwd := ""
			for _, f := range files {
				name := ""
				if strings.HasPrefix(f, path+"/") {
					name = f[len(path+"/"):] // remove header
				} else if isDir {
					name = f
				}
				if strings.Contains(name, "/") {
					if !(cwd != "" && strings.HasPrefix(name, cwd)) { // only add new dir
						name = name[:strings.Index(name, "/")+1]
						cwd = name
						filelist = append(filelist, name)
					}
				} else if name != "" {
					filelist = append(filelist, name)
				}
			}
			sort.Sort(fileSlice(filelist))
			filestr := strings.Join(filelist, "\n")
			return []byte(filestr), Dir, nil
		}
	}

	// if a file, display the file
	cmd = exec.Command("hg", "cat", "-r", rev, "--", path)
	cmd.Dir = r.dir
	if out, err := cmd.CombinedOutput(); err == nil {
		return out, File, nil
	} else {
		if strings.Contains(string(out), fmt.Sprintf("%s: no such file in rev", path)) {
			return nil, File, os.ErrNotExist
		}
		if strings.Contains(string(out), fmt.Sprintf("abort: unknown revision '%s'!", rev)) {
			return nil, File, os.ErrNotExist
		}
		return nil, File, fmt.Errorf("hg %v failed: %s\n%s", cmd.Args, err, out)
	}
}

func (r *hgRepo) readFileAtRevisionHgo(path string, rev string) ([]byte, error) {
	rp, err := hgo.OpenRepository(r.dir)
	if err != nil {
		return nil, err
	}
	st := rp.NewStore()
	rs := parseRevisionSpec(rp, rev, "tip")

	fileLog, err := st.OpenRevlog(path)
	if err != nil {
		return nil, err
	}

	ra := repoAccess{
		rp: rp,
		fb: revlog.NewFileBuilder(),
		st: st,
	}

	localId, ok := rs.(revlog.FileRevSpec)
	if !ok {
		localId, err = ra.localChangesetId(rs)
		if err != nil {
			return nil, err
		}
	}

	rec, err := lookupFile(fileLog, int(localId), func() (*store.ManifestEnt, error) {
		return ra.manifestEntry(int(localId), path)
	})
	if err != nil {
		return nil, err
	}

	fb := revlog.NewFileBuilder()
	return fb.Build(rec)
}

type repoAccess struct {
	rp        *hgo.Repository
	fb        *revlog.FileBuilder
	st        *store.Store
	changelog *revlog.Index
}

func (ra *repoAccess) localChangesetId(rs revlog.RevisionSpec) (chgId revlog.FileRevSpec, err error) {
	r, err := ra.clRec(rs)
	if err == nil {
		chgId = revlog.FileRevSpec(r.FileRev())
	}
	return
}

func (ra *repoAccess) clRec(rs revlog.RevisionSpec) (r *revlog.Rec, err error) {
	if ra.changelog == nil {
		log, err1 := ra.st.OpenChangeLog()
		if err1 != nil {
			err = err1
			return
		}
		ra.changelog = log
	}
	r, err = rs.Lookup(ra.changelog)
	return
}

func (ra *repoAccess) manifestEntry(chgId int, fileName string) (me *store.ManifestEnt, err error) {
	r, err := ra.clRec(revlog.FileRevSpec(chgId))
	if err != nil {
		return
	}
	c, err := changelog.BuildEntry(r, ra.fb)
	if err != nil {
		return
	}
	m, err := getManifest(ra.rp, int(c.Linkrev), c.ManifestNode, ra.fb)
	if err != nil {
		return
	}
	me = m.Map()[fileName]
	if me == nil {
		err = errors.New("file does not exist in given revision")
	}
	return
}

func getManifest(rp *hgo.Repository, linkrev int, id revlog.NodeId, b *revlog.FileBuilder) (m store.Manifest, err error) {
	st := rp.NewStore()
	mlog, err := st.OpenManifests()
	if err != nil {
		return
	}

	r, err := mlog.LookupRevision(linkrev, id)
	if err != nil {
		return
	}

	m, err = store.BuildManifest(r, b)
	return
}

// Lookup the given revision of a file. The Manifest is consulted only
// if necessary, i.e. if it can''t be told from the filelog whether a file exists yet or not
func lookupFile(fileLog *revlog.Index, chgId int, manifestEntry func() (*store.ManifestEnt, error)) (r *revlog.Rec, err error) {
	r, err = revlog.LinkRevSpec(chgId).Lookup(fileLog)
	if err != nil {
		return
	}
	if r.FileRev() == -1 {
		err = revlog.ErrRevisionNotFound
		return
	}

	if int(r.Linkrev) == chgId {
		// The requested revision matches this record, which can be
		// used as a sign that the file is existent yet.
		return
	}

	if !r.IsLeaf() {
		// There are other records that have the current record as a parent.
		// This means, the file was existent, no need to check the manifest.
		return
	}

	// Check for the file's existence using the manifest.
	ent, err := manifestEntry()
	if err != nil {
		return
	}

	// compare hashes
	wantId, err := ent.Id()
	if err != nil {
		return
	}
	if !wantId.Eq(r.Id()) {
		err = errors.New("manifest node id does not match file id")
	}
	return
}

func parseRevisionSpec(rp *hgo.Repository, s, dflt string) revlog.RevisionSpec {
	if s == "" {
		s = dflt
	}
	if s == "tip" {
		return revlog.TipRevSpec{}
	}
	if s == "null" {
		return revlog.NullRevSpec{}
	}
	_, allTags := rp.Tags()
	if id, ok := allTags.IdByName[s]; ok {
		s = id
	} else if i, err := strconv.ParseInt(s, 16, 0); err == nil {
		return revlog.FileRevSpec(i)
	}

	return revlog.NodeIdRevSpec(s)
}

func (r *hgRepo) CurrentCommitID() (string, error) {
	cmd := exec.Command("hg", "identify", "-i")
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
