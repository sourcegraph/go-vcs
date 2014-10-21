package git

/*
extern int _govcs_gcrypt_init();
#cgo LDFLAGS: -lgcrypt
*/
import "C"
import (
	"log"
	"os"

	"code.google.com/p/go.crypto/ssh"

	git2go "github.com/libgit2/git2go"
	"github.com/sourcegraph/go-vcs/vcs"
	"github.com/sourcegraph/go-vcs/vcs/gitcmd"
	"github.com/sourcegraph/go-vcs/vcs/util"
)

func init() {
	// Overwrite the git cloner to use the faster libgit2
	// implementation.
	vcs.RegisterCloner("git", func(url, dir string, opt vcs.CloneOpt) (vcs.Repository, error) {
		return Clone(url, dir, opt)
	})
}

func init() {
	// Initialize gcrypt for multithreaded operation. See
	// gcrypt_init.c for more information.
	rv := C._govcs_gcrypt_init()
	if rv != 0 {
		log.Fatal("gcrypt multithreaded init failed (see gcrypt_init.c)")
	}
}

func Clone(url, dir string, opt vcs.CloneOpt) (vcs.Repository, error) {
	clopt := git2go.CloneOptions{Bare: opt.Bare}

	rc, cfs, err := makeRemoteCallbacks(url, opt.RemoteOpts)
	if err != nil {
		return nil, err
	}
	if cfs != nil {
		defer cfs.run()
	}
	clopt.RemoteCallbacks = rc

	u, err := git2go.Clone(url, dir, &clopt)
	if err != nil {
		return nil, err
	}
	cr, err := gitcmd.Open(dir)
	if err != nil {
		return nil, err
	}
	return &Repository{cr, u}, nil
}

func (r *Repository) UpdateEverything(opt vcs.RemoteOpts) error {
	// TODO(sqs): allow use of a remote other than "origin"
	rm, err := r.u.LoadRemote("origin")
	if err != nil {
		return err
	}

	rc, cfs, err := makeRemoteCallbacks(rm.Url(), opt)
	if err != nil {
		return err
	}
	if cfs != nil {
		defer cfs.run()
	}
	rm.SetCallbacks(rc)

	if err := rm.Fetch(nil, nil, ""); err != nil {
		return err
	}

	return nil
}

type cleanupFuncs []func() error

func (f cleanupFuncs) run() error {
	for _, cf := range f {
		if err := cf(); err != nil {
			return err
		}
	}
	return nil
}

// makeRemoteCallbacks constructs the remote callbacks for libgit2
// remote operations. Currently the remote callbacks are trivial
// (empty) except when using an SSH remote.
//
// cleanupFuncs's run method should be called when the RemoteCallbacks
// struct is done being used. It is OK to ignore the error return.
func makeRemoteCallbacks(url string, opt vcs.RemoteOpts) (rc *git2go.RemoteCallbacks, cfs cleanupFuncs, err error) {
	defer func() {
		// Clean up if error; don't expect the caller to clean up if
		// we have a non-nil error.
		if err != nil {
			cfs.run()
		}
	}()

	if opt.SSH != nil {
		privkeyFilename, privkeyFile, err := util.WriteKeyTempFile(url, opt.SSH.PrivateKey)
		if err != nil {
			return nil, nil, err
		}
		cfs = append(cfs, privkeyFile.Close)
		cfs = append(cfs, func() error { return os.Remove(privkeyFile.Name()) })

		// Derive public key from private key if empty.
		if opt.SSH.PublicKey == nil {
			privKey, err := ssh.ParsePrivateKey(opt.SSH.PrivateKey)
			if err != nil {
				return nil, cfs, err
			}
			opt.SSH.PublicKey = ssh.MarshalAuthorizedKey(privKey.PublicKey())
		}

		pubkeyFilename, pubkeyFile, err := util.WriteKeyTempFile(url, opt.SSH.PublicKey)
		if err != nil {
			return nil, cfs, err
		}
		cfs = append(cfs, pubkeyFile.Close)
		cfs = append(cfs, func() error { return os.Remove(pubkeyFile.Name()) })

		rc = &git2go.RemoteCallbacks{
			CredentialsCallback: func(url string, usernameFromURL string, allowedTypes git2go.CredType) (int, *git2go.Cred) {
				if allowedTypes&git2go.CredTypeSshKey != 0 {
					rv, cred := git2go.NewCredSshKey(usernameFromURL, pubkeyFilename, privkeyFilename, "")
					return rv, &cred
				}
				log.Printf("No authentication available for git URL %q.", url)
				return 1, nil
			},
			CertificateCheckCallback: func(cert *git2go.Certificate, valid bool, hostname string) int {
				if InsecureSkipCheckVerifySSH {
					return 0
				}
				if !valid {
					log.Printf("Invalid certificate for SSH host %s: %v.", hostname, cert)
					return -1
				}
				return 0
			},
		}
	}

	return rc, cfs, nil
}

// InsecureSkipCheckVerifySSH controls whether the client verifies the
// SSH server's certificate or host key. If InsecureSkipCheckVerifySSH
// is true, the program is susceptible to a man-in-the-middle
// attack. This should only be used for testing.
var InsecureSkipCheckVerifySSH bool
