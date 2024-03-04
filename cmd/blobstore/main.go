package main

import (
	"github.com/open-bytestack/levelstore/cmd/blobstore/blobstore"
)

func main() {
	err := blobstore.MainCommand().Execute()
	if err != nil {
		panic(err)
	}
}
