package git_libgit2

import (
	"log"

	git2go "github.com/libgit2/git2go"
	"github.com/sourcegraph/go-vcs/vcs"
)

func CloneMirrorGitRepository(url, dir string) (vcs.MirrorRepository, error) {

	opt := git2go.CloneOptions{
		Bare: true,
		RemoteCallbacks: &git2go.RemoteCallbacks{
			SidebandProgressCallback: func(s string) int {
				log.Println("P: ", s)
				return 0
			},
			CompletionCallback: func(rc git2go.RemoteCompletion) int {
				log.Println("C: ", rc)
				return 0
			},
			CredentialsCallback: func(url string, usernameFromURL string, allowedTypes git2go.CredType) (int, *git2go.Cred) {
				if allowedTypes&git2go.CredTypeSshKey != 0 {
					rv, cred := git2go.NewCredSshKey("git", "/home/sqs/.ssh/id_dsa.pub", "/home/sqs/.ssh/id_dsa", "")
					log.Printf("NewCredSshKey: rv=%d, cred=%+v", rv, cred)
					return rv, &cred
				}
				log.Printf("No authentication available for git URL %q.", url)
				return 1, nil
			},
		},
	}
	if _, err := git2go.Clone(url, dir, &opt); err != nil {
		return nil, err
	}
	return vcs.OpenMirror("git", dir)
}

func GitMirrorUpdate(dir string) error {
	return nil
}
