package vcs

import (
	"fmt"
	"os/exec"
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
