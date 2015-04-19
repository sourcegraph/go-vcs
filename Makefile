.PHONY: test docker-test

test:
	go test -ldflags "-extldflags=-L"`go list -f '{{.Dir}}' github.com/libgit2/git2go`/vendor/libgit2/build ./...

docker-test:
	docker build -t go-vcs . && docker run go-vcs
