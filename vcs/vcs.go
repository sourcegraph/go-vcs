package vcs

import (
	"github.com/libgit2/git2go"
)

type VCS interface {
	OpenRepository(path string) (Repository, error)
}

type Git struct{}

var _ VCS = Git{}

type Repository interface {
	LookupReference(name string) (Reference, error)
	LookupBlob(oid *Oid) (Blob, error)
	LookupCommit(oid *Oid) (Commit, error)
}

type gitRepository struct{ *git.Repository }

var _ Repository = &gitRepository{}

func (_ Git) OpenRepository(path string) (Repository, error) {
	r, err := git.OpenRepository(path)
	return &gitRepository{r}, err
}

func (r *gitRepository) LookupReference(name string) (Reference, error) {
	v, err := r.Repository.LookupReference(name)
	return &gitReference{v}, err
}

func (r *gitRepository) LookupCommit(oid *Oid) (Commit, error) {
	v, err := r.Repository.LookupCommit((*git.Oid)(oid))
	return &gitCommit{v}, err
}

func (r *gitRepository) LookupBlob(oid *Oid) (Blob, error) {
	v, err := r.Repository.LookupBlob((*git.Oid)(oid))
	return &gitBlob{v}, err
}

type Oid git.Oid

type Reference interface {
	Resolve() (Reference, error)
	Target() *Oid
}

type gitReference struct {
	*git.Reference
}

var _ Reference = &gitReference{}

func (ref *gitReference) Resolve() (Reference, error) {
	ref2, err := ref.Reference.Resolve()
	return &gitReference{ref2}, err
}

func (ref *gitReference) Target() *Oid {
	return (*Oid)(ref.Reference.Target())
}

type Signature git.Signature

type Commit interface {
	Message() string
	Tree() (Tree, error)
	Author() *Signature
}

type gitCommit struct {
	*git.Commit
}

var _ Commit = &gitCommit{}

func (c *gitCommit) Message() string { return c.Commit.Message() }

func (c *gitCommit) Tree() (Tree, error) {
	t, err := c.Commit.Tree()
	return &gitTree{t}, err
}

func (c *gitCommit) Author() *Signature {
	return (*Signature)(c.Commit.Author())
}

type TreeEntry struct {
	Name     string
	Id       *Oid
	Type     git.ObjectType
	Filemode int
}

type Tree interface {
	Walk(callback TreeWalkCallback) error
}

type TreeWalkCallback func(string, *TreeEntry) int

type gitTree struct {
	*git.Tree
}

var _ Tree = &gitTree{}

func (t *gitTree) Walk(callback TreeWalkCallback) error {
	return t.Walk(func(a string, b *TreeEntry) int {
		return callback(a, b)
	})
}

type Blob interface {
	Size() int64
	Contents() []byte
}

type gitBlob struct {
	*git.Blob
}

var _ Blob = &gitBlob{}

func (b *gitBlob) Size() int64      { return b.Blob.Size() }
func (b *gitBlob) Contents() []byte { return b.Blob.Contents() }
