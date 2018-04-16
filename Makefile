.PHONY: docker-test

docker-test:
	docker build -t go-vcs . && docker run go-vcs
