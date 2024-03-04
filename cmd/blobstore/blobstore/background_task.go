package blobstore

import (
	"os"
	"path/filepath"
	"time"

	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"go.etcd.io/bbolt"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type taskController struct {
	backgroundTaskInterval time.Duration
	backgroundTaskHandler  func(in *goproto.BlobKey) error

	emitTask func(tx *bbolt.Tx, in *goproto.BlobKey) error
}

func (s *Server) registerBackgroundHandler() {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketNameDeleteTask)
		return err
	})
	if err != nil {
		panic(err)
	}

	// delete blob task
	s.tc[string(bucketNameDeleteTask)] = &taskController{
		backgroundTaskHandler: func(in *goproto.BlobKey) error {
			blobPath := filepath.Join(s.blockDir, getBlobPath(in))
			err := os.Remove(blobPath)
			if err != nil {
				zap.L().With(zap.Error(err), zap.String("blob_path", blobPath)).Error("remove file error")
				return err
			}
			blobKey := formatBlobKey(in)
			zap.L().With(zap.String("blob_path", blobPath)).Info("try delete blob")
			return s.db.Update(func(tx *bbolt.Tx) error {
				return multierr.Append(
					tx.Bucket(bucketNameBlob).Delete(blobKey),
					tx.Bucket(bucketNameDeleteTask).Delete(blobKey),
				)
			})
		},
		backgroundTaskInterval: 10 * time.Second,

		emitTask: func(tx *bbolt.Tx, in *goproto.BlobKey) error {
			return tx.Bucket(bucketNameDeleteTask).Put(formatBlobKey(in), nil)
		},
	}
}

func (s *Server) backgroundLoop() {
	for t, bh := range s.tc {
		go func(typ string, tc *taskController) {
			for {
				time.Sleep(tc.backgroundTaskInterval)
				zap.L().With(zap.String("task_type", typ)).Info("start run background loop")
				var allKeys []*goproto.BlobKey
				err := s.db.View(func(tx *bbolt.Tx) error {
					return tx.Bucket([]byte(typ)).ForEach(func(k, v []byte) error {
						blobKey := parseBlobKey(k)
						allKeys = append(allKeys, blobKey)
						return nil
					})
				})
				if len(allKeys) != 0 {
					zap.L().With(zap.String("task_type", typ), zap.Int("entry_count", len(allKeys))).Info("start to execute all task")
				}
				if err != nil {
					zap.L().With(
						zap.String("task_type", typ),
						zap.Error(err),
					).Error("run background loop error")
					continue
				}

				for i := range allKeys {
					err = tc.backgroundTaskHandler(allKeys[i])
					if err != nil {
						zap.L().With(
							zap.String("task_type", typ),
							zap.Error(err),
						).Error("handle background task error")
						continue
					}
				}
			}
		}(t, bh)
	}
}

func getDeleteTaskController() string {
	return string(bucketNameDeleteTask)
}
