package blobsdk

import (
	"io"

	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"google.golang.org/genproto/googleapis/bytestream"
)

type Interface interface {
	NewBlobReader(in *goproto.BlobKey) io.ReadCloser
	NewBlobWriter(in *goproto.BlobKey) io.WriteCloser

	bytestream.ByteStreamClient
	goproto.BlobServiceClient
}
