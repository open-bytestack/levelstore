.PHONY: proto

GOPATH := $(shell go env GOPATH)

golangci-lint:
	if ! test -f ${GOPATH}/bin/golangci-lint; then curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" | sh -s -- -b ${GOPATH}/bin latest; fi
	${GOPATH}/bin/golangci-lint run

proto:
	protoc --go-grpc_out="${PWD}"/gopkg --go_out="${PWD}"/gopkg proto/blobstore.proto