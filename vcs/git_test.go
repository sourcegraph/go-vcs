package vcs

import "testing"

func TestRepository_ResolveBranch(t *testing.T) {
	defer removeTmpDirs()

	cmds := []string{
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
	}
	tests := map[string]struct {
		repo interface {
			ResolveBranch(string) (CommitID, error)
		}
		branch       string
		wantCommitID CommitID
	}{
		"git native": {
			repo:         makeGitRepositoryNative(t, cmds...),
			branch:       "master",
			wantCommitID: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8",
		},
		"git libgit2": {
			repo:         makeGitRepositoryLibGit2(t, cmds...),
			branch:       "master",
			wantCommitID: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8",
		},
		"git cmd": {
			repo:         &GitRepositoryCmd{initGitRepository(t, cmds...)},
			branch:       "master",
			wantCommitID: "ea167fe3d76b1e5fd3ed8ca44cbd2fe3897684f8",
		},
	}

	for label, test := range tests {
		commitID, err := test.repo.ResolveBranch(test.branch)
		if err != nil {
			t.Errorf("%s: ResolveBranch: %s", label, err)
			continue
		}

		if commitID != test.wantCommitID {
			t.Errorf("%s: got commitID == %v, want %v", label, commitID, test.wantCommitID)
		}
	}
}
