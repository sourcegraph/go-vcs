go-vcs - manipulate and inspect VCS repositories
================================================

go-vcs is a library for manipulating and inspecting VCS repositories in Go. It currently supports
Git and Mercurial (hg).

Note: the public API is experimental and subject to change until further notice.

* [Documentation on Sourcegraph](https://sourcegraph.com/github.com/sourcegraph/go-vcs)

[![Build Status](https://travis-ci.org/sourcegraph/go-vcs.png?branch=master)](https://travis-ci.org/sourcegraph/go-vcs)
[![status](https://sourcegraph.com/api/repos/github.com/sourcegraph/go-vcs/badges/status.png)](https://sourcegraph.com/github.com/sourcegraph/go-vcs)
[![xrefs](https://sourcegraph.com/api/repos/github.com/sourcegraph/go-vcs/badges/xrefs.png)](https://sourcegraph.com/github.com/sourcegraph/go-vcs)
[![top func](https://sourcegraph.com/api/repos/github.com/sourcegraph/go-vcs/badges/top-func.png)](https://sourcegraph.com/github.com/sourcegraph/go-vcs)
[![library users](https://sourcegraph.com/api/repos/github.com/sourcegraph/go-vcs/badges/library-users.png)](https://sourcegraph.com/github.com/sourcegraph/go-vcs)


Installing
==========

Use of the `git_libgit2` package (which provides faster git operations than the
`git`-command-based implementation) requires libgit2, which you can install by
running:

```
git clone git://github.com/libgit2/libgit2.git /tmp/libgit2
cd /tmp/libgit2
git checkout e18d5e52e385c0cc2ad8d9d4fdd545517f170a11 # known good version; newer versions probably work too
mkdir build
cd build
cmake -DCMAKE_INSTALL_PREFIX=/usr -DBUILD_CLAR=OFF -DTHREADSAFE=ON ..
cmake --build . --target install
```

You probably need to be `root` to run the last command.


Running tests
=============

Run `go test ./vcs`. Note that the tests test the `git_libgit2` implementation,
which requires libgit2 (see above usage instructions).


TODOs
============

* Use build tags in package vcs tests to eliminate hard requirement of libgit2.


Contributors
============

* Quinn Slack <sqs@sourcegraph.com>
