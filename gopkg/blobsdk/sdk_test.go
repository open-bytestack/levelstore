package blobsdk_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/open-bytestack/levelstore/cmd/blobstore/blobstore"
	"github.com/open-bytestack/levelstore/gopkg/blobsdk"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/bytestream"
)

func TestBlobSDK(t *testing.T) {
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

func TestBlobSDKGRPC(t *testing.T) {
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
	t.Log("write data")
	data := bytes.Repeat([]byte("a"), 8192)
	wc, err := cli.Write(ctx)
	assert.Nil(t, err)
	err = wc.Send(&bytestream.WriteRequest{
		ResourceName: fmt.Sprintf("%d/%d", 10000, 0),
		FinishWrite:  true,
		Data:         data,
	})
	assert.Nil(t, err)
	resp, err := wc.CloseAndRecv()
	assert.Nil(t, err)
	assert.Equal(t, int64(8192), resp.CommittedSize)

	bi, err := cli.StatBlob(ctx, bkey)
	assert.Nil(t, err)
	assert.Equal(t, goproto.BlobState_NORMAL, bi.State)
	rc, err := cli.Read(ctx, &bytestream.ReadRequest{
		ResourceName: fmt.Sprintf("%d/%d", 10000, 0),
		ReadOffset:   0,
		ReadLimit:    0,
	})
	defer rc.CloseSend()
	rr, err := rc.Recv()
	assert.Nilf(t, err, "%s", err)
	assert.Equal(t, len(rr.Data), len(data))
	cancel()
	<-finished
}
