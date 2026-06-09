package git

import (
	"errors"

	"sourcegraph.com/sourcegraph/go-git"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func standardizeError(err error) error {
	var objectNotFound git.ObjectNotFound
	if errors.As(err, &objectNotFound) {
		return vcs.ErrCommitNotFound
	}
	return err
}
