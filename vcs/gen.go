package vcs

// Regenerating vcs.pb.go requires protoc and protoc-gen-gogo on $PATH. The
// .proto files that vcs.proto imports live in the module cache, so their
// include directories are resolved with "go list -m" instead of $GOPATH.

//go:generate sh -c "protoc -I. -I$(go list -m -f '{{.Dir}}' github.com/gogo/protobuf) -I$(go list -m -f '{{.Dir}}' github.com/gogo/protobuf)/protobuf -I$(go list -m -f '{{.Dir}}' github.com/sqs/pbtypes) --gogo_out=Mtimestamp.proto=github.com/sqs/pbtypes:. vcs.proto"
