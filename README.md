go-vcs - manipulate and inspect VCS repositories
================================================

go-vcs is a library for manipulating and inspecting VCS repositories in Go. It currently supports
Git and Mercurial (hg).

Note: the public API is experimental and subject to change until further notice.

* [Documentation on Sourcegraph](https://sourcegraph.com/sourcegraph.com/sourcegraph/go-vcs)

[![Build Status](https://travis-ci.org/sourcegraph/go-vcs.png?branch=master)](https://travis-ci.org/sourcegraph/go-vcs)
[![status](https://sourcegraph.com/api/repos/sourcegraph.com/sourcegraph/go-vcs/.badges/status.png)](https://sourcegraph.com/sourcegraph.com/sourcegraph/go-vcs)

Resolving dependencies
======================

The faster libgit2 implementation of git depends on `git2go` on its `next` branch. To install it, you will [need](https://github.com/libgit2/git2go/tree/next#installing):

- `cmake`
- `pkg-config`
- `libssh2`
- `libgcrypt`

Once you have those prerequisites, follow [these steps](https://github.com/libgit2/git2go/tree/next#from-next) to install `git2go` on `next` branch.

For hg blame, you need to install hglib: `pip install python-hglib`.

Installing
==========

```
go get -u sourcegraph.com/sourcegraph/go-vcs/vcs
```

Implementation differences
==========================

The goal is to have all supported backends at feature parity, but until then, consult this table for implementation differences.

| Feature                               | git                  | gitcmd             | hg                   | hgcmd                |
|---------------------------------------|----------------------|--------------------|----------------------|----------------------|
| vcs.CommitsOptions.Path               | :white_large_square: | :white_check_mark: | :white_large_square: | :white_large_square: |
| vcs.BranchesOptions.MergedInto        | :white_large_square: | :white_check_mark: | :white_large_square: | :white_large_square: |
| vcs.BranchesOptions.IncludeCommit     | :white_large_square: | :white_check_mark: | :white_large_square: | :white_large_square: |
| vcs.BranchesOptions.BehindAheadBranch | :white_large_square: | :white_check_mark: | :white_large_square: | :white_large_square: |
| vcs.Repository.Committers             | :white_large_square: | :white_check_mark: | :white_large_square: | :white_large_square: |
| vcs.FileLister                        | :white_large_square: | :white_check_mark: | :white_large_square: | :white_large_square: |

Contributions that fill in the gaps are welcome!

Development
===========

### First-time installation of protobuf and other codegen tools

You need to install and run the protobuf compiler before you can regenerate Go code after you change the `vcs.proto` file.

1.	**Install protoc**, the protobuf compiler. Find more details in the [protobuf README](https://github.com/google/protobuf).

	On OS X, you can install it with Homebrew by running:

	```
	brew install --devel protobuf
	```

	Then make sure the `protoc` binary is in your `$PATH`.

2.	**Install [gogo/protobuf](https://github.com/gogo/protobuf)**.

	```
	go get -u github.com/gogo/protobuf/...
	```

3.	**Install `gopathexec`**:

	```
	go get -u sourcegraph.com/sourcegraph/gopathexec
	```

### Regenerating Go code after changing `vcs.proto`

```
go generate sourcegraph.com/sourcegraph/go-vcs/vcs/...
```

### Running tests

Run `go test ./vcs/...`. You may need to supply linker flags to link with libgit2. If you get a linker error, try running `make test` instead. If that doesn't work, check the command that `make test` runs to see if it is using the correct paths on your system.

Note that the tests test the libgit2 implementation and SSH support (see above instructions).

Contributors
============

* Quinn Slack <sqs@sourcegraph.com>

See all contributors [here](https://github.com/sourcegraph/go-vcs/graphs/contributors).
