package vcs

import (
	"fmt"
	"os/exec"
)

func init() {
	RegisterOpener("hg", func(dir string) (Repository, error) {
		return OpenHgRepositoryCmd(dir)
	})
	RegisterCloner("hg", func(url, dir string, opt CloneOpt) (Repository, error) {
		return CloneHgRepository(url, dir, opt)
	})
}

func CloneHgRepository(url, dir string, opt CloneOpt) (*HgRepositoryCmd, error) {
	args := []string{"clone"}
	if opt.Bare {
		args = append(args, "--noupdate")
	}
	args = append(args, "--", url, dir)
	cmd := exec.Command("hg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec `hg clone` failed: %s. Output was:\n\n%s", err, out)
	}
	return OpenHgRepositoryCmd(dir)
}
