go-vcs - manipulate and inspect VCS repositories
================================================

go-vcs is a library for manipulating and inspecting VCS repositories in Go. It currently supports
Git and Mercurial (hg).

Note: the public API is experimental and subject to change until further notice.

* [View on Sourcegraph](https://sourcegraph.com/github.com/sourcegraph/go-vcs/-/def/GoPackage/github.com/sourcegraph/go-vcs/vcs/-/Repository)


[![GoDoc](https://pkg.go.dev/badge/github.com/sourcegraph/go-vcs/vcs.svg)](https://pkg.go.dev/github.com/sourcegraph/go-vcs/vcs#Repository)

Resolving dependencies
======================

For hg blame, you need to install hglib: `pip install python-hglib`.

Installing
==========

```
go get github.com/sourcegraph/go-vcs/vcs
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
| vcs.UpdateResult                      | :white_large_square: | :white_check_mark: | :white_large_square: | :white_large_square: |

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

2.	**Install [gogo/protobuf](https://github.com/gogo/protobuf)**'s code generator.

	```
	go install github.com/gogo/protobuf/protoc-gen-gogo@latest
	```

	Then make sure the `protoc-gen-gogo` binary is in your `$PATH`.

### Regenerating Go code after changing `vcs.proto`

```
go generate ./vcs/...
```

### Running tests

Run `go test ./vcs/...`.

Contributors
============

* Quinn Slack <sqs@sourcegraph.com>

See all contributors [here](https://github.com/sourcegraph/go-vcs/graphs/contributors).
