.PHONY: install test docker-test

install:
	go get -d github.com/libgit2/git2go
	cd `go list -f '{{.Dir}}' github.com/libgit2/git2go` && git submodule update --init && make install
	go install ./vcs

test:
	go test -ldflags "-extldflags=-L"`go list -f '{{.Dir}}' github.com/libgit2/git2go`/vendor/libgit2/build ./...

docker-test:
	docker build -t go-vcs . && docker run go-vcs
