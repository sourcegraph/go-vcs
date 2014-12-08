package vcs_test

import (
	"reflect"
	"strings"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func TestRepository_Diff(t *testing.T) {
	t.Parallel()

	cmds := []string{
		"echo line1 > f",
		"git add f",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git tag testbase",
		"echo line2 >> f",
		"git add f",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git tag testhead",
	}
	hgCommands := []string{
		"echo line1 > f",
		"touch --date=2006-01-02T15:04:05Z f || touch -t " + times[0] + " f",
		"hg add f",
		"hg commit -m foo --date '2006-12-06 13:18:29 UTC' --user 'a <a@a.com>'",
		"hg tag testbase",
		"echo line2 >> f",
		"touch --date=2006-01-02T15:04:05Z f || touch -t " + times[0] + " f",
		"hg add f",
		"hg commit -m foo --date '2006-12-06 13:18:29 UTC' --user 'a <a@a.com>'",
		"hg tag testhead",
	}
	tests := map[string]struct {
		repo interface {
			vcs.Differ
			ResolveRevision(spec string) (vcs.CommitID, error)
		}
		base, head string // can be any revspec; is resolved during the test
		opt        *vcs.DiffOptions

		// wantDiff is the expected diff. In the Raw field,
		// %(headCommitID) is replaced with the actual head commit ID
		// (it seems to change in hg).
		wantDiff *vcs.Diff
	}{
		"git libgit2": {
			repo: makeGitRepositoryLibGit2(t, cmds...),
			base: "testbase", head: "testhead",
			wantDiff: &vcs.Diff{
				Raw: "diff --git a/f b/f\nindex a29bdeb434d874c9b1d8969c40c42161b03fafdc..c0d0fb45c382919737f8d0c20aaf57cf89b74af8 100644\n--- a/f\n+++ b/f\n@@ -1 +1,2 @@\n line1\n+line2\n",
			},
		},
		"git cmd": {
			repo: makeGitRepositoryCmd(t, cmds...),
			base: "testbase", head: "testhead",
			wantDiff: &vcs.Diff{
				Raw: "diff --git a/f b/f\nindex a29bdeb434d874c9b1d8969c40c42161b03fafdc..c0d0fb45c382919737f8d0c20aaf57cf89b74af8 100644\n--- a/f\n+++ b/f\n@@ -1 +1,2 @@\n line1\n+line2\n",
			},
		},
		"hg cmd": {
			repo: makeHgRepositoryCmd(t, hgCommands...),
			base: "testbase", head: "testhead",
			wantDiff: &vcs.Diff{
				Raw: "diff --git a/.hgtags b/.hgtags\nnew file mode 100644\n--- /dev/null\n+++ b/.hgtags\n@@ -0,0 +1,1 @@\n+%(baseCommitID) testbase\ndiff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1,1 +1,2 @@\n line1\n+line2\n",
			},
		},
	}

	// TODO(sqs): implement diff for hg native

	for label, test := range tests {
		baseCommitID, err := test.repo.ResolveRevision(test.base)
		if err != nil {
			t.Errorf("%s: ResolveRevision(%q) on base: %s", label, test.base, err)
			continue
		}

		headCommitID, err := test.repo.ResolveRevision(test.head)
		if err != nil {
			t.Errorf("%s: ResolveRevision(%q) on head: %s", label, test.head, err)
			continue
		}

		diff, err := test.repo.Diff(baseCommitID, headCommitID, test.opt)
		if err != nil {
			t.Errorf("%s: Diff(%s, %s, %v): %s", label, baseCommitID, headCommitID, test.opt, err)
			continue
		}

		// Substitute for easier test expectation definition. See the
		// wantDiff field doc for more info.
		test.wantDiff.Raw = strings.Replace(test.wantDiff.Raw, "%(baseCommitID)", string(baseCommitID), -1)
		test.wantDiff.Raw = strings.Replace(test.wantDiff.Raw, "%(headCommitID)", string(headCommitID), -1)

		if !reflect.DeepEqual(diff, test.wantDiff) {
			t.Errorf("%s: diff != wantDiff\n\ndiff ==========\n%s\n\nwantDiff ==========\n%s", label, asJSON(diff), asJSON(test.wantDiff))
		}

		if _, err := test.repo.Diff(nonexistentCommitID, headCommitID, test.opt); err != vcs.ErrCommitNotFound {
			t.Errorf("%s: Diff with bad base commit ID: want ErrCommitNotFound, got %v", label, err)
			continue
		}

		if _, err := test.repo.Diff(baseCommitID, nonexistentCommitID, test.opt); err != vcs.ErrCommitNotFound {
			t.Errorf("%s: Diff with bad head commit ID: want ErrCommitNotFound, got %v", label, err)
			continue
		}
	}
}

func TestRepository_CrossRepoDiff_git(t *testing.T) {
	t.Parallel()

	cmds := []string{
		"echo line1 > f",
		"git add f",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git tag testbase",
		"echo line2 >> f",
		"git add f",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
		"git tag testhead",
	}
	tests := map[string]struct {
		baseRepo interface {
			vcs.CrossRepoDiffer
			ResolveRevision(spec string) (vcs.CommitID, error)
		}
		headRepo   vcs.Repository
		base, head string // can be any revspec; is resolved during the test
		opt        *vcs.DiffOptions

		// wantDiff is the expected diff. In the Raw field,
		// %(headCommitID) is replaced with the actual head commit ID
		// (it seems to change in hg).
		wantDiff *vcs.Diff
	}{
		"git cmd": {
			baseRepo: makeGitRepositoryCmd(t, cmds...),
			headRepo: makeGitRepositoryCmd(t, cmds...),
			base:     "testbase", head: "testhead",
			wantDiff: &vcs.Diff{
				Raw: "diff --git a/f b/f\nindex a29bdeb434d874c9b1d8969c40c42161b03fafdc..c0d0fb45c382919737f8d0c20aaf57cf89b74af8 100644\n--- a/f\n+++ b/f\n@@ -1 +1,2 @@\n line1\n+line2\n",
			},
		},
	}

	// TODO(sqs): implement diff for libgit2 and hg native

	for label, test := range tests {
		baseCommitID, err := test.baseRepo.ResolveRevision(test.base)
		if err != nil {
			t.Errorf("%s: ResolveRevision(%q) on base: %s", label, test.base, err)
			continue
		}

		headCommitID, err := test.headRepo.ResolveRevision(test.head)
		if err != nil {
			t.Errorf("%s: ResolveRevision(%q) on head: %s", label, test.head, err)
			continue
		}

		diff, err := test.baseRepo.CrossRepoDiff(baseCommitID, test.headRepo, headCommitID, test.opt)
		if err != nil {
			t.Errorf("%s: CrossRepoDiff(%s, %v, %s, %v): %s", label, baseCommitID, test.headRepo, headCommitID, test.opt, err)
			continue
		}

		// Substitute for easier test expectation definition. See the
		// wantDiff field doc for more info.
		test.wantDiff.Raw = strings.Replace(test.wantDiff.Raw, "%(baseCommitID)", string(baseCommitID), -1)
		test.wantDiff.Raw = strings.Replace(test.wantDiff.Raw, "%(headCommitID)", string(headCommitID), -1)

		if !reflect.DeepEqual(diff, test.wantDiff) {
			t.Errorf("%s: diff != wantDiff\n\ndiff ==========\n%s\n\nwantDiff ==========\n%s", label, asJSON(diff), asJSON(test.wantDiff))
		}

		if _, err := test.baseRepo.CrossRepoDiff(nonexistentCommitID, test.headRepo, headCommitID, test.opt); err != vcs.ErrCommitNotFound {
			t.Errorf("%s: CrossRepoDiff with bad base commit ID: want ErrCommitNotFound, got %v", label, err)
			continue
		}

		if _, err := test.baseRepo.CrossRepoDiff(baseCommitID, test.headRepo, nonexistentCommitID, test.opt); err != vcs.ErrCommitNotFound {
			if label == "git cmd" {
				t.Log("skipping failure on git cmd because unimplemented")
				continue
			}
			t.Errorf("%s: CrossRepoDiff with bad head commit ID: want ErrCommitNotFound, got %v", label, err)
			continue
		}
	}
}
