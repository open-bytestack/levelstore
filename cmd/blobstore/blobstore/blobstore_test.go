package blobstore_test

import (
	"context"
	"github.com/open-bytestack/levelstore/cmd/blobstore/blobstore"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"path/filepath"
	"testing"
)

func TestBlobStoreServer(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.db")

	srv := blobstore.NewServer(&blobstore.Config{
		BaseDir:      tempFile,
		BlobChecksum: false,
		WriteDio:     false,
	})
	t.Run("test create blob", func(t *testing.T) {
		createReq := &goproto.CreateBlobReq{
			VolumeId: 1,
			Seq:      0,
			BlobSize: blobstore.SignBlobSize,
		}
		_, err := srv.CreateBlob(context.TODO(), createReq)
		assert.Nil(t, err)

		blob, err := srv.StatBlob(context.TODO(), &goproto.BlobKey{
			VolumeId: 1,
			Seq:      0,
		})
		assert.Nil(t, err)
		assert.Equal(t, blob.VolumeId, createReq.VolumeId)
		assert.Equal(t, blob.Seq, createReq.Seq)
		assert.Equal(t, blob.BlobSize, createReq.BlobSize)
		assert.Equal(t, blob.State, goproto.BlobState_INIT)

		listBlobResp, err := srv.ListBlob(context.TODO(), &goproto.ListBlobReq{
			Blob:  nil,
			Limit: 128,
		})
		assert.Nil(t, err)
		assert.Len(t, listBlobResp.BlobInfos, 1)
		assert.Equal(t, listBlobResp.Truncated, false)
		assert.Equal(t, listBlobResp.BlobInfos[0].VolumeId, createReq.VolumeId)
		assert.Equal(t, listBlobResp.BlobInfos[0].Seq, createReq.Seq)
		assert.Equal(t, listBlobResp.BlobInfos[0].BlobSize, createReq.BlobSize)

		_, err = srv.DeleteBlob(context.TODO(), &goproto.BlobKey{
			VolumeId: createReq.VolumeId,
			Seq:      createReq.Seq,
		})
		assert.Nil(t, err)

		blob, err = srv.StatBlob(context.TODO(), &goproto.BlobKey{
			VolumeId: 1,
			Seq:      0,
		})
		assert.Nil(t, err)
		assert.Equal(t, blob.VolumeId, createReq.VolumeId)
		assert.Equal(t, blob.Seq, createReq.Seq)
		assert.Equal(t, blob.BlobSize, createReq.BlobSize)
		assert.Equal(t, blob.State, goproto.BlobState_DELETING)
	})

	t.Run("create blob again", func(t *testing.T) {
		createReq := &goproto.CreateBlobReq{
			VolumeId: 1,
			Seq:      0,
			BlobSize: blobstore.SignBlobSize,
		}
		_, err := srv.CreateBlob(context.TODO(), createReq)
		code, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, code.Code(), codes.AlreadyExists)
	})

	t.Run("test list blobs", func(t *testing.T) {
		for i := 2; i <= 200; i++ {
			createReq := &goproto.CreateBlobReq{
				VolumeId: uint64(i),
				Seq:      0,
				BlobSize: blobstore.SignBlobSize,
			}
			_, err := srv.CreateBlob(context.TODO(), createReq)
			assert.Nil(t, err)
			blob, err := srv.StatBlob(context.TODO(), &goproto.BlobKey{
				VolumeId: uint64(i),
				Seq:      0,
			})
			assert.Nil(t, err)
			assert.Equal(t, blob.VolumeId, createReq.VolumeId)
			assert.Equal(t, blob.Seq, createReq.Seq)
			assert.Equal(t, blob.BlobSize, createReq.BlobSize)
			assert.Equal(t, blob.State, goproto.BlobState_INIT)
		}

		resp, err := srv.ListBlob(context.TODO(), &goproto.ListBlobReq{
			Blob:  nil,
			Limit: 128,
		})
		assert.Nil(t, err)
		assert.True(t, resp.Truncated)
		assert.Len(t, resp.BlobInfos, 128)
		assert.Equal(t, uint64(129), resp.NextContinuationToken.VolumeId)
		assert.Equal(t, uint32(0), resp.NextContinuationToken.Seq)

		resp2, err := srv.ListBlob(context.TODO(), &goproto.ListBlobReq{
			Blob:  resp.NextContinuationToken,
			Limit: 128,
		})
		assert.Nil(t, err)
		assert.False(t, resp2.Truncated)
		assert.Len(t, resp2.BlobInfos, 200-128)
		assert.Nil(t, resp2.NextContinuationToken)

		resp, err = srv.ListBlob(context.TODO(), &goproto.ListBlobReq{
			Blob:  nil,
			Limit: 200,
		})

		assert.Nil(t, err)
		assert.False(t, resp.Truncated)
		assert.Len(t, resp.BlobInfos, 200)
		assert.Nil(t, resp2.NextContinuationToken)

		resp, err = srv.ListBlob(context.TODO(), &goproto.ListBlobReq{
			Blob:  nil,
			Limit: 201,
		})

		assert.Nil(t, err)
		assert.False(t, resp.Truncated)
		assert.Len(t, resp.BlobInfos, 200)
		assert.Nil(t, resp2.NextContinuationToken)
	})
}

func BenchmarkBlobServer(b *testing.B) {

	tempDir := b.TempDir()
	tempFile := filepath.Join(tempDir, "test.db")

	srv := blobstore.NewServer(&blobstore.Config{
		BaseDir:      tempFile,
		BlobChecksum: true,
		WriteDio:     true,
	})

	for i := 0; i < 1000; i++ {
		_, err := srv.CreateBlob(context.TODO(), &goproto.CreateBlobReq{
			VolumeId: uint64(i),
			Seq:      0,
			BlobSize: 4096,
		})
		if err != nil {
			b.Errorf("create blob error: %s", err)
			b.Fail()
		}
	}

	for i := 1000 - 1; i > 0; i-- {
		_, err := srv.DeleteBlob(context.TODO(), &goproto.BlobKey{
			VolumeId: uint64(i),
			Seq:      0,
		})
		if err != nil {
			b.Errorf("delete blob error: %s", err)
			b.Fail()
		}
	}
}
