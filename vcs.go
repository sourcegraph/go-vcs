package vcs

type VCS interface {
	// Clones the repository at the given URL into dir. If dir already exists, the error os.ErrExist
	// is returned.
	Clone(url, dir string) (Repository, error)
}

type Repository interface {
	Dir() string // The repository's root directory.
	VCS() VCS

	// Downloads updates to the repository from the default remote.
	Download() error

	// CheckOut returns the path of a directory containing a working tree at revision rev. CheckOut
	// assumes that rev is local or has already been fetched; it does not update the repository.
	CheckOut(rev string) (dir string, err error)
}

func Clone(vcs VCS, url, dir string) (Repository, error) {
	return vcs.Clone(url, dir)
}
