package vcs_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/go-vcs/vcs"
)

func TestRepository_BlameFile(t *testing.T) {
	t.Parallel()

	cmds := []string{
		"echo line1 > f",
		"git add f",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"echo line2 >> f",
		"git add f",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
	}
	// TODO(sqs): implement blame for hg
	tests := map[string]struct {
		repo interface {
			vcs.Blamer
			ResolveRevision(spec string) (vcs.CommitID, error)
		}
		path string
		opt  *vcs.BlameOptions

		wantHunks []*vcs.Hunk
	}{
		"git libgit2": {
			repo: makeGitRepositoryLibGit2(t, cmds...),
			path: "f",
			opt: &vcs.BlameOptions{
				NewestCommit: "master",
			},
			wantHunks: []*vcs.Hunk{
				{
					StartLine: 1, EndLine: 2, CommitID: "e6093374dcf5725d8517db0dccbbf69df65dbde0",
					Author: vcs.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				},
				{
					StartLine: 2, EndLine: 3, CommitID: "fad406f4fe02c358a09df0d03ec7a36c2c8a20f1",
					Author: vcs.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				},
			},
		},
		"git cmd": {
			repo: makeGitRepositoryCmd(t, cmds...),
			path: "f",
			opt: &vcs.BlameOptions{
				NewestCommit: "master",
			},
			wantHunks: []*vcs.Hunk{
				{
					StartLine: 1, EndLine: 2, CommitID: "e6093374dcf5725d8517db0dccbbf69df65dbde0",
					Author: vcs.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				},
				{
					StartLine: 2, EndLine: 3, CommitID: "fad406f4fe02c358a09df0d03ec7a36c2c8a20f1",
					Author: vcs.Signature{Name: "a", Email: "a@a.com", Date: mustParseTime(time.RFC3339, "2006-01-02T15:04:05Z")},
				},
			},
		},
	}

	for label, test := range tests {
		newestCommitID, err := test.repo.ResolveRevision(string(test.opt.NewestCommit))
		if err != nil {
			t.Errorf("%s: ResolveRevision(%q) on base: %s", label, test.opt.NewestCommit, err)
			continue
		}

		test.opt.NewestCommit = newestCommitID
		hunks, err := test.repo.BlameFile(test.path, test.opt)
		if err != nil {
			t.Errorf("%s: BlameFile(%s, %+v): %s", label, test.path, test.opt, err)
			continue
		}

		if !reflect.DeepEqual(hunks, test.wantHunks) {
			t.Errorf("%s: hunks != wantHunks\n\nhunks ==========\n%s\n\nwantHunks ==========\n%s", label, asJSON(hunks), asJSON(test.wantHunks))
		}
	}
}
