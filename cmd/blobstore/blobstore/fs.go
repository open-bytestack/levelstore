package blobstore

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"path/filepath"
)

const hashToDirPrefix = 2

func getBlobPath(in *goproto.BlobKey) string {
	blobKey := formatBlobKey(in)
	hashedBlobKeyRaw := sha256.Sum256(blobKey)
	encodedHashString := hex.EncodeToString(hashedBlobKeyRaw[:])
	return filepath.Join(encodedHashString[:hashToDirPrefix], encodedHashString+"_"+hex.EncodeToString(blobKey))
}
