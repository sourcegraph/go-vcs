package tracer

import (
	"fmt"
	"time"

	"sourcegraph.com/sourcegraph/appdash"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func init() { appdash.RegisterEvent(GoVCS{}) }

// GoVCS records a go-vcs method invocation.
type GoVCS struct {
	Name, Args string

	StartTime time.Time
	EndTime   time.Time
}

// Schema returns the constant "GoVCS".
func (GoVCS) Schema() string { return "GoVCS" }

func (e GoVCS) Start() time.Time { return e.StartTime }
func (e GoVCS) End() time.Time   { return e.EndTime }

// Wrap wraps the given VCS repository, returning a repository which emits
// tracing events.
func Wrap(r vcs.Repository, rec *appdash.Recorder) vcs.Repository {
	// Wrap the repository.
	t := repository{r: r, rec: rec}

	// Also wrap optional interfaces.
	realBlamer, isBlamer := r.(vcs.Blamer)
	realDiffer, isDiffer := r.(vcs.Differ)
	realCrossRepoDiffer, isCrossRepoDiffer := r.(vcs.CrossRepoDiffer)
	realFileLister, isFileLister := r.(vcs.FileLister)

	blamer := blamer{repository: t, b: realBlamer, rec: rec}
	differ := differ{repository: t, d: realDiffer, rec: rec}
	crossRepoDiffer := crossRepoDiffer{repository: t, c: realCrossRepoDiffer, rec: rec}
	fileLister := fileLister{repository: t, f: realFileLister, rec: rec}

	switch {
	case isBlamer && isDiffer && isCrossRepoDiffer && isFileLister:
		return struct {
			vcs.Repository
			vcs.Blamer
			vcs.Differ
			vcs.CrossRepoDiffer
			vcs.FileLister
		}{t, blamer, differ, crossRepoDiffer, fileLister}

	case isBlamer && isDiffer && isCrossRepoDiffer:
		return struct {
			vcs.Repository
			vcs.Blamer
			vcs.Differ
			vcs.CrossRepoDiffer
		}{t, blamer, differ, crossRepoDiffer}

	case isBlamer && isDiffer:
		return struct {
			vcs.Repository
			vcs.Blamer
			vcs.Differ
		}{t, blamer, differ}

	case isBlamer:
		return struct {
			vcs.Repository
			vcs.Blamer
		}{t, blamer}

	default:
		return t
	}
}

// blamer implements the vcs.Repository interface, adding a wrapped vcs.Blamer
// implementation.
type blamer struct {
	repository
	b   vcs.Blamer
	rec *appdash.Recorder
}

// BlameFile implements the vcs.Blamer interface.
func (b blamer) BlameFile(path string, opt *vcs.BlameOptions) ([]*vcs.Hunk, error) {
	start := time.Now()
	hunks, err := b.b.BlameFile(path, opt)
	b.rec.Child().Event(GoVCS{
		Name:      "vcs.Blamer.BlameFile",
		Args:      fmt.Sprintf("%#v, %#v", path, opt),
		StartTime: start,
		EndTime:   time.Now(),
	})
	return hunks, err
}

// differ implements the vcs.Repository interface, adding a wrapped vcs.Differ
// implementation.
type differ struct {
	repository
	d   vcs.Differ
	rec *appdash.Recorder
}

// Diff implements the vcs.Differ interface.
func (d differ) Diff(base, head vcs.CommitID, opt *vcs.DiffOptions) (*vcs.Diff, error) {
	start := time.Now()
	diff, err := d.d.Diff(base, head, opt)
	d.rec.Child().Event(GoVCS{
		Name:      "vcs.Differ.Diff",
		Args:      fmt.Sprintf("%#v, %#v, %#v", base, head, opt),
		StartTime: start,
		EndTime:   time.Now(),
	})
	return diff, err
}

// crossRepoDiffer implements the vcs.Repository interface, adding a wrapped
// vcs.CrossRepoDiffer implementation.
type crossRepoDiffer struct {
	repository
	c   vcs.CrossRepoDiffer
	rec *appdash.Recorder
}

// CrossRepoDiff implements the vcs.CrossRepoDiffer interface.
func (c crossRepoDiffer) CrossRepoDiff(base vcs.CommitID, headRepo vcs.Repository, head vcs.CommitID, opt *vcs.DiffOptions) (*vcs.Diff, error) {
	start := time.Now()
	diff, err := c.c.CrossRepoDiff(base, headRepo, head, opt)
	c.rec.Child().Event(GoVCS{
		Name:      "vcs.CrossRepoDiffer.CrossRepoDiff",
		Args:      fmt.Sprintf("%#v, %#v, %#v, %#v", base, headRepo, head, opt),
		StartTime: start,
		EndTime:   time.Now(),
	})
	return diff, err
}

// fileLister implements the vcs.Repository interface, adding a wrapped
// vcs.FileListener implementation.
type fileLister struct {
	repository
	f   vcs.FileLister
	rec *appdash.Recorder
}

// ListFiles implements the vcs.FileLister interface.
func (f fileLister) ListFiles(commit vcs.CommitID) ([]string, error) {
	start := time.Now()
	files, err := f.f.ListFiles(commit)
	f.rec.Child().Event(GoVCS{
		Name:      "vcs.FileLister.ListFiles",
		Args:      fmt.Sprintf("%#v", commit),
		StartTime: start,
		EndTime:   time.Now(),
	})
	return files, err
}
