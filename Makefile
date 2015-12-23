.PHONY: test docker-test

test:
	go test ./...

docker-test:
	docker build -t go-vcs . && docker run go-vcs
