# A Dockerfile that tries to simulate running in CI, to make debugging test
# failures easier.

FROM golang:1.26

# git-shell (shipped with git) and ssh are needed by the git-over-SSH tests;
# they are skipped when either is missing.
RUN apt-get update -q \
	&& apt-get install -qq --no-install-recommends git openssh-client \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /go-vcs
COPY go.mod go.sum ./
RUN go mod download
COPY . .

CMD ["test", "-race", "./..."]
ENTRYPOINT ["go"]
