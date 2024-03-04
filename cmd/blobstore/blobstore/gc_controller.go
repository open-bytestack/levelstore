package blobstore

import (
	"context"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"go.etcd.io/bbolt"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"time"
)

type gcController struct {
	gcInterval time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	db         *bbolt.DB
	blobDir    string
}

func (gc *gcController) gcOneBlob(in *goproto.BlobKey) error {
	blobPath := filepath.Join(gc.blobDir, getBlobPath(in))
	err := os.Remove(blobPath)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("blob_path", blobPath)).Error("remove file error")
		return err
	}
	blobKey := formatBlobKey(in)
	zap.L().With(zap.String("blob_path", blobPath)).Info("try delete blob")
	return gc.db.Update(func(tx *bbolt.Tx) error {
		return multierr.Append(
			tx.Bucket(bucketNameBlob).Delete(blobKey),
			tx.Bucket(bucketNameDeleteTombstone).Delete(blobKey),
		)
	})
}

func (gc *gcController) runGCLoop() {
	if gc.ctx != nil {
		zap.L().Error("gc loop already started")
		return
	}
	gc.ctx, gc.cancel = context.WithCancel(context.Background())
	defer func() {
		gc.ctx, gc.cancel = nil, nil
	}()
	for {
		time.Sleep(gc.gcInterval)
		if gc.ctx == nil {
			return
		}
		select {
		case <-gc.ctx.Done():
			return
		default:
		}
		logger := zap.L().With(zap.String("module", "gc"))
		logger.Info("start run background loop")
		var allKeys []*goproto.BlobKey
		err := gc.db.View(func(tx *bbolt.Tx) error {
			return tx.Bucket(bucketNameDeleteTombstone).ForEach(func(k, v []byte) error {
				blobKey := parseBlobKey(k)
				allKeys = append(allKeys, blobKey)
				return nil
			})
		})
		if len(allKeys) != 0 {
			logger.With(zap.Int("entry_count", len(allKeys))).Info("start to execute all task")
		}
		if err != nil {
			logger.With(
				zap.Error(err),
			).Error("run background loop error")
			continue
		}

		for i := range allKeys {
			err = gc.gcOneBlob(allKeys[i])
			if err != nil {
				logger.With(
					zap.Error(err),
				).Error("handle background task error")
				continue
			}
		}
	}
}

func (gc *gcController) stopGCLoop() {
	gc.cancel()
}
