package blobsdk_test

import (
	"bytes"
	"context"
	"github.com/open-bytestack/levelstore/cmd/blobstore/blobstore"
	"github.com/open-bytestack/levelstore/gopkg/blobsdk"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

func TestBlobSdk(t *testing.T) {
	tempDir := t.TempDir()
	cmd := blobstore.MainCommand()
	cmd.SetArgs([]string{"--basedir", tempDir})
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		cmd.ExecuteContext(ctx)
		finished <- struct{}{}
	}()
	time.Sleep(5 * time.Second)
	cli, err := blobsdk.NewClient("localhost:8080")
	assert.Nil(t, err)
	t.Log("create blob")
	_, err = cli.CreateBlob(ctx, &goproto.CreateBlobReq{
		VolumeId: 10000,
		Seq:      0,
		BlobSize: 8192,
	})
	assert.Nil(t, err)

	t.Log("create blob writer")

	bkey := &goproto.BlobKey{
		VolumeId: 10000,
		Seq:      0,
	}
	wc := cli.NewBlobWriter(bkey)

	t.Log("write data")
	data := bytes.Repeat([]byte("a"), 8192)
	n, err := wc.Write(data)
	t.Log("data  write finished")
	assert.Equal(t, 8192, n)
	assert.Nil(t, wc.Close())
	t.Log("shutdown the servers")

	bi, err := cli.StatBlob(ctx, bkey)
	assert.Nil(t, err)
	assert.Equal(t, goproto.BlobState_NORMAL, bi.State)

	rc := cli.NewBlobReader(bkey)
	content, err := io.ReadAll(rc)
	assert.Nil(t, err)
	assert.Nil(t, rc.Close())
	assert.Equal(t, content, data)

	cancel()
	<-finished
}
