package vcs

import "testing"

func TestRepository_ResolveBranch(t *testing.T) {
	defer removeTmpDirs()

	cmds := []string{
		"GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit --allow-empty -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
	}
	tests := map[string]struct {
		repo         GitRepository
		branch       string
		wantCommitID CommitID
	}{
		"git": {
			repo:         makeLocalGitRepository(t, false, cmds...),
			branch:       "master",
			wantCommitID: "c556aa409427eed1322744a02ad23066f51040fb",
		},
		"git cmd": {
			repo:         makeLocalGitRepository(t, true, cmds...),
			branch:       "master",
			wantCommitID: "c556aa409427eed1322744a02ad23066f51040fb",
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
