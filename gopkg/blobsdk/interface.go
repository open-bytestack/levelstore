package blobsdk

import (
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"io"
)

type Interface interface {
	NewBlobReader(in *goproto.BlobKey) io.ReadCloser
	NewBlobWriter(in *goproto.BlobKey) io.WriteCloser
	goproto.BlobServiceClient
}
