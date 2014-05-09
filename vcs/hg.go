package vcs

type HgRepository interface {
	Repository

	// No hg-specific methods yet, but let's keep this type to be consistent
	// with GitRepository.
}

type hgRepository struct {
	dir string
	*HgRepositoryNative
	cmd *HgRepositoryCmd
}

func OpenHgRepository(dir string) (HgRepository, error) {
	native, err := OpenHgRepositoryNative(dir)
	if err != nil {
		return nil, err
	}

	return &hgRepository{dir, native, &HgRepositoryCmd{dir}}, nil
}
