package vcs

import (
	"fmt"
	"os/exec"

	"code.google.com/p/go.tools/godoc/vfs"
)

type HgRepository interface {
	Repository

	// No hg-specific methods yet, but let's keep this type to be consistent
	// with GitRepository.
}

type hgRepository struct {
	Dir string
	*HgRepositoryNative
	cmd *HgRepositoryCmd
}

func (r *hgRepository) MirrorUpdate() error {
	cmd := exec.Command("hg", "pull")
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec `hg pull` failed: %s. Output was:\n\n%s", err, out)
	}
	return nil
}

func (r *hgRepository) FileSystem(at CommitID) (vfs.FileSystem, error) {
	// TODO(sqs): this is a temporary hack to fix issues with file
	// handling in hg repos (specifically some bitbucket atlassian
	// repos).
	return r.cmd.FileSystem(at)
}

func OpenHgRepository(dir string) (HgRepository, error) {
	native, err := OpenHgRepositoryNative(dir)
	if err != nil {
		return nil, err
	}

	return &hgRepository{dir, native, &HgRepositoryCmd{dir}}, nil
}

func CloneHgRepository(urlStr, dir string) (HgRepository, error) {
	cmd := exec.Command("hg", "clone", "--", urlStr, dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg clone` failed: %s. Output was:\n\n%s", err, out)
	}

	return OpenHgRepository(dir)
}
