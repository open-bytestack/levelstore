.PHONY: proto


proto:
	protoc --go-grpc_out="${PWD}"/gopkg --go_out="${PWD}"/gopkg proto/blobstore.proto