package git

import (
	"github.com/sourcegraph/go-git"
	"github.com/sourcegraph/go-vcs/vcs"
)

func standardizeError(err error) error {
	switch err := err.(type) {
	case git.ObjectNotFound:
		return vcs.ErrCommitNotFound
	default:
		return err
	}
}
