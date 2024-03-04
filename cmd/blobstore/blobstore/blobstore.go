package blobstore

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ncw/directio"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	db *bbolt.DB
	tc map[string]*taskController

	blockDir string

	goproto.UnimplementedBlobServiceServer

	bufferPool sync.Pool
}

func NewServer(baseDir string) *Server {
	fi, err := os.Stat(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(baseDir, 0o755)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} else {
		if !fi.IsDir() {
			panic(baseDir + " is not dir")
		}
	}
	metaDBPath := filepath.Join(baseDir, "meta.db")
	db, err := bbolt.Open(metaDBPath, 0o600, nil)
	if err != nil {
		panic(err)
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketNameBlob)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	srv := &Server{
		db:       db,
		tc:       make(map[string]*taskController),
		blockDir: filepath.Join(baseDir, "blocks"),
		bufferPool: sync.Pool{New: func() interface{} {
			return directio.AlignedBlock(4096)
		}},
	}

	srv.registerBackgroundHandler()

	go srv.backgroundLoop()
	return srv
}

const (
	keyLen = 8 + 4
)

var (
	bucketNameBlob       = []byte("blobs")
	bucketNameDeleteTask = []byte("delete")

	errorBlobExists         = errors.New("blob exists")
	errorBlobNotExists      = errors.New("blob not exists")
	errorBlobDeleting       = errors.New("blob deleting")
	errorBlobMetaCorruption = errors.New("blob meta corruption")
	errorCASUpdateBlobState = errors.New("cas blob state error")
)

const (
	blobMetaWriteErrorReason = "write-error-reason"
)

// formatBlobKey
func formatBlobKey(in *goproto.BlobKey) []byte {
	var buf = make([]byte, keyLen)
	binary.BigEndian.PutUint64(buf[:8], in.VolumeId)
	binary.BigEndian.PutUint32(buf[8:], in.Seq)
	return buf
}

func parseBlobKey(in []byte) *goproto.BlobKey {
	if len(in) != keyLen {
		panic("invalid key blob key length")
	}
	volumeID := binary.BigEndian.Uint64(in[:8])
	seq := binary.BigEndian.Uint32(in[8:])
	return &goproto.BlobKey{
		VolumeId: volumeID,
		Seq:      seq,
	}
}

func (s *Server) CreateBlob(_ context.Context, req *goproto.CreateBlobReq) (*emptypb.Empty, error) {
	zap.L().With(zap.String("req", req.String())).Info("create blob success")
	var err error
	defer func() {
		if err != nil {
			zap.L().With(zap.Error(err), zap.String("req", req.String())).Error("create blob failed")
		}
	}()
	blobKey := formatBlobKey(&goproto.BlobKey{
		VolumeId: req.VolumeId,
		Seq:      req.Seq,
	})
	err = s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketNameBlob)
		shouldBeEmpty := b.Get(blobKey)
		if len(shouldBeEmpty) != 0 {
			return errorBlobExists
		}
		blob := &goproto.BlobInfo{
			VolumeId:           req.VolumeId,
			Seq:                req.Seq,
			BlobSize:           req.BlobSize,
			State:              goproto.BlobState_INIT,
			CreateTimestamp:    time.Now().UTC().Unix(),
			LastCheckTimestamp: 0,
			Meta:               nil,
		}
		bin, err := proto.Marshal(blob)
		if err != nil {
			return err
		}
		return b.Put(blobKey, bin)
	})

	if err != nil {
		if errors.Is(err, errorBlobExists) {
			return &emptypb.Empty{}, status.Errorf(codes.AlreadyExists, "can not create blob: %s", err.Error())
		}
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "update meta error: %s", err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) StatBlob(_ context.Context, req *goproto.BlobKey) (*goproto.BlobInfo, error) {
	blobKey := formatBlobKey(req)
	var out *goproto.BlobInfo
	err := s.db.View(func(tx *bbolt.Tx) error {
		bin := tx.Bucket(bucketNameBlob).Get(blobKey)
		if len(bin) == 0 {
			return errorBlobNotExists
		}
		out = new(goproto.BlobInfo)
		err := proto.Unmarshal(bin, out)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errorBlobNotExists) {
			return nil, status.Errorf(codes.NotFound, "blob [%s] not exists", req.String())
		}
		return nil, status.Errorf(codes.Internal, "stat blob error: %s", err.Error())
	}
	return out, nil
}

func (s *Server) DeleteBlob(_ context.Context, req *goproto.BlobKey) (*emptypb.Empty, error) {
	blobKey := formatBlobKey(req)
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bin := tx.Bucket(bucketNameBlob).Get(blobKey)
		if len(bin) == 0 {
			return errorBlobNotExists
		}
		out := new(goproto.BlobInfo)
		err := proto.Unmarshal(bin, out)
		if err != nil {
			return err
		}
		if out.State == goproto.BlobState_DELETING {
			return errorBlobDeleting
		}
		zap.L().With(
			zap.Uint64("volume_id", req.VolumeId),
			zap.Uint32("seq", req.Seq),
			zap.Time("deletion_time", time.Now()),
		).Info("delete blob")
		out.State = goproto.BlobState_DELETING
		bin, _ = proto.Marshal(out)
		err = s.tc[getDeleteTaskController()].emitTask(tx, req)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketNameBlob).Put(blobKey, bin)
	})
	if err != nil {
		if errors.Is(err, errorBlobNotExists) {
			return nil, status.Errorf(codes.NotFound, "blob [%s] not exists", req.String())
		}
		return nil, status.Errorf(codes.Internal, "update meta error: %s", err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) ListBlob(_ context.Context, req *goproto.ListBlobReq) (*goproto.ListBlobResp, error) {
	if req.Limit == 0 {
		req.Limit = 128
	}
	limit := req.Limit
	var out = new(goproto.ListBlobResp)

	filterFn := func(info *goproto.BlobInfo) bool {
		if req.StateFilter != nil && *req.StateFilter != info.State {
			return false
		}
		return true
	}

	handleOne := func(key, value []byte) error {
		blobKey := parseBlobKey(key)
		var info = new(goproto.BlobInfo)
		err := proto.Unmarshal(value, info)
		if err != nil {
			return err
		}
		if info.VolumeId != blobKey.VolumeId || info.Seq != blobKey.Seq {
			return errorBlobMetaCorruption
		}
		if filterFn(info) {
			out.BlobInfos = append(out.BlobInfos, info)
			limit--
		}
		return nil
	}
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketNameBlob)
		cur := b.Cursor()

		var key, value []byte
		if req.Blob != nil {
			key, value = cur.Seek(formatBlobKey(req.Blob))
		} else {
			key, value = cur.First()
		}
		err := handleOne(key, value)
		if err != nil {
			return err
		}
		for limit > 0 {
			key, value = cur.Next()
			if len(key) == 0 {
				break
			}
			err = handleOne(key, value)
			if err != nil {
				return err
			}
		}

		if limit == 0 {
			key, value = cur.Next()
			if len(key) == 0 {
				return nil
			}
			out.Truncated = true
			blobKey := parseBlobKey(key)
			out.NextContinuationToken = &goproto.BlobKey{
				VolumeId: blobKey.VolumeId,
				Seq:      blobKey.Seq,
			}
		}
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list meta error: %s", err.Error())
	}
	return out, nil
}

func (s *Server) atomicUpdateBlobState(in *goproto.BlobKey, state goproto.BlobState) error {
	blobKey := formatBlobKey(in)
	return s.db.Update(func(tx *bbolt.Tx) error {
		bin := tx.Bucket(bucketNameBlob).Get(blobKey)
		if len(bin) == 0 {
			return errorBlobNotExists
		}
		bi := new(goproto.BlobInfo)
		err := proto.Unmarshal(bin, bi)
		if err != nil {
			return errors.Join(err, errorBlobMetaCorruption)
		}
		if bi.State >= state {
			return errorCASUpdateBlobState
		}

		bi.State = state
		bin, err = proto.Marshal(bi)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketNameBlob).Put(blobKey, bin)
	})
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	blockKey := strings.TrimPrefix(r.RequestURI, "/")
	volumeIDAndSeq := strings.Split(blockKey, "/")
	if len(volumeIDAndSeq) != 2 {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("invalid request url: " + r.RequestURI))
		return
	}
	volumeID, err := strconv.ParseInt(volumeIDAndSeq[0], 10, 64)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("invalid request url: " + r.RequestURI))
		return
	}
	seq, err := strconv.ParseInt(volumeIDAndSeq[1], 10, 64)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("invalid request url: " + r.RequestURI))
		return
	}
	in := &goproto.BlobKey{
		VolumeId: uint64(volumeID),
		Seq:      uint32(seq),
	}
	switch r.Method {
	case http.MethodGet:
		s.readBlob(rw, r, in)
	case http.MethodPut:
		s.writeBlob(rw, r, in)
	}
}

func (s *Server) readBlob(rw http.ResponseWriter, r *http.Request, in *goproto.BlobKey) {
	zap.L().With(zap.String("blob_key", in.String())).Info("read blob")
	var err error
	defer func() {
		if err != nil {
			zap.L().With(
				zap.Error(err),
				zap.Uint64("volume_id", in.VolumeId),
				zap.Uint32("volume_seq", in.Seq),
			).Error("read blob failed")
		}
	}()
	blobInfo, err := s.StatBlob(r.Context(), in)
	if err != nil {
		code, ok := status.FromError(err)
		if ok {
			if code.Code() == codes.NotFound {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte("blob not found: " + r.RequestURI))
				return
			}
		}

		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("read blob error: " + err.Error()))
		return
	}
	if blobInfo.State != goproto.BlobState_NORMAL {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("read unreadable blob in state: " + blobInfo.State.String()))
		return
	}

	rw.Header().Set("Content-Length", strconv.FormatInt(int64(blobInfo.BlobSize), 10))
	rw.Header().Set("Content-Type", "application/octet-stream")

	path := filepath.Join(s.blockDir, getBlobPath(in))
	f, err := os.Open(path)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("open blob file error: %w", err)
		return
	}
	defer f.Close()
	buf := s.bufferPool.Get().([]byte)
	defer s.bufferPool.Put(buf)
	n, err := io.CopyBuffer(rw, f, buf)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("copy blob file error: %w", err)
		return
	}
	if n != int64(blobInfo.BlobSize) {
		rw.WriteHeader(http.StatusInternalServerError)
		err = errors.New("copied size mismatch the blob size")
		return
	}
	return
}

func (s *Server) writeBlob(rw http.ResponseWriter, r *http.Request, in *goproto.BlobKey) {
	blobInfo, err := s.StatBlob(r.Context(), in)
	formattedBlobKey := formatBlobKey(in)
	if err != nil {
		code, ok := status.FromError(err)
		if ok {
			if code.Code() == codes.NotFound {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte("write not found: " + r.RequestURI))
				return
			}
		}

		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("write blob error: " + err.Error()))
		return
	}

	if blobInfo.State != goproto.BlobState_INIT {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("write unwritable blob in state: " + blobInfo.State.String()))
		return
	}

	err = s.atomicUpdateBlobState(in, goproto.BlobState_PENDING)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("write blob atomicUpdateBlobState error: " + err.Error()))
		return
	}

	var writeErr error = nil

	defer func() {
		if writeErr != nil {
			blobInfo.State = goproto.BlobState_BROKEN
			if blobInfo.Meta == nil {
				blobInfo.Meta = make(map[string]string, 1)
			}
			blobInfo.Meta[blobMetaWriteErrorReason] = writeErr.Error()
			bin, _ := proto.Marshal(blobInfo)
			err := s.db.Update(func(tx *bbolt.Tx) error {
				return tx.Bucket(bucketNameBlob).Put(formattedBlobKey, bin)
			})
			if err != nil {
				zap.L().With(
					zap.Error(err),
					zap.Uint64("volume_id", in.VolumeId),
					zap.Uint32("seq", in.Seq),
					zap.String("blob_info", blobInfo.String()),
				).Error("update blob info error")
			}
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(writeErr.Error()))
			return
		} else {
			blobInfo.State = goproto.BlobState_NORMAL
			bin, _ := proto.Marshal(blobInfo)
			err := s.db.Update(func(tx *bbolt.Tx) error {
				return tx.Bucket(bucketNameBlob).Put(formattedBlobKey, bin)
			})
			if err != nil {
				zap.L().With(
					zap.Error(err),
					zap.Uint64("volume_id", in.VolumeId),
					zap.Uint32("seq", in.Seq),
					zap.String("blob_info", blobInfo.String()),
				).Error("update blob info error")
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(err.Error()))
				return
			}
		}
	}()

	defer r.Body.Close()

	blobPath := filepath.Join(s.blockDir, getBlobPath(in))
	blobDir := filepath.Dir(blobPath)
	err = os.MkdirAll(blobDir, 0o755)
	if err != nil {
		writeErr = fmt.Errorf("mkdir %s error: %s", blobDir, err)
		return
	}

	// opened file can not be deferred close may cause leak
	df, err := directio.OpenFile(blobPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		writeErr = fmt.Errorf("open file %s error: %s", blobPath, err)
		return
	}

	buf := s.bufferPool.Get().([]byte)
	defer s.bufferPool.Put(buf)
	written, err := io.CopyBuffer(df, r.Body, buf)
	if err != nil {
		writeErr = fmt.Errorf("copy body to file error: %s", err)
		return
	}
	if r.ContentLength != written {
		writeErr = fmt.Errorf("copy body size mismatched with content-length, should be %d but %d written", r.ContentLength, written)
		return
	}

	if uint64(r.ContentLength) != blobInfo.BlobSize {
		writeErr = fmt.Errorf("copy body size mismatched with blobinfo.size, should be %d but %d written", r.ContentLength, blobInfo.BlobSize)
		return
	}

	err = df.Sync()
	if err != nil {
		writeErr = fmt.Errorf("sync file error: %w", err)
		return
	}

	err = df.Close()
	if err != nil {
		writeErr = fmt.Errorf("close file error: %w", err)
		return
	}
}
