package vcs_test

import (
	"reflect"
	"testing"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
)

func TestRepository_Search(t *testing.T) {
	t.Parallel()

	searchOpt := vcs.SearchOptions{
		Query:        "xy",
		QueryType:    vcs.FixedQuery,
		ContextLines: 1,
	}
	wantRes := []*vcs.SearchResult{
		{
			File:      "f1",
			StartLine: 2,
			EndLine:   3,
			Match:     []byte("def\nxyz"),
		},
		{
			File:      "f2",
			StartLine: 1,
			EndLine:   1,
			Match:     []byte("xyz"),
		},
	}

	gitCommands := []string{
		"echo abc > f1",
		"echo def >> f1",
		"echo xyz >> f1",
		"echo xyz > f2",
		"git add f1 f2",
		"GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2006-01-02T15:04:05Z git commit f1 f2 -m foo --author='a <a@a.com>' --date 2006-01-02T15:04:05Z",
	}
	// TODO(sqs): implement hg Searcher
	tests := map[string]struct {
		repo        vcs.Searcher
		spec        vcs.CommitID
		opt         vcs.SearchOptions
		wantResults []*vcs.SearchResult
	}{
		"git libgit2": {
			repo:        makeGitRepositoryLibGit2(t, gitCommands...),
			spec:        "master",
			opt:         searchOpt,
			wantResults: wantRes,
		},
		"git cmd": {
			repo:        makeGitRepositoryCmd(t, gitCommands...),
			spec:        "master",
			opt:         searchOpt,
			wantResults: wantRes,
		},
	}

	for label, test := range tests {
		res, err := test.repo.Search(test.spec, test.opt)
		if err != nil {
			t.Errorf("%s: Search: %s", label, err)
			continue
		}

		if !reflect.DeepEqual(res, test.wantResults) {
			t.Errorf("%s: got results == %v, want %v", label, asJSON(res), asJSON(test.wantResults))
		}
	}
}
