go-vcs - manipulate and inspect VCS repositories
================================================

go-vcs is a library for manipulating and inspecting VCS repositories in Go. It currently supports
Git and Mercurial (hg).

Note: the public API is experimental and subject to change until further notice.


Installing
==========

Requires libgit2, which you can install by running:

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

Run `go test ./vcs`.


Contributors
============

* Quinn Slack <sqs@sourcegraph.com>
