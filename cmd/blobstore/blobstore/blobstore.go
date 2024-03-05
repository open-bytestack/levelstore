package blobstore

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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

type Config struct {
	BaseDir      string
	WriteDio     bool
	BlobChecksum bool
}

type Server struct {
	db      *bbolt.DB
	blobDir string

	gcController *gcController

	_2MBytesPool    sync.Pool
	writeDio        bool
	writeFileOpener func(name string, flag int, perm os.FileMode) (file *os.File, err error)

	blobChecksum bool
	signManager  *blobSignManager

	goproto.UnimplementedBlobServiceServer
}

func NewServer(cfg *Config) *Server {
	fi, err := os.Stat(cfg.BaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(cfg.BaseDir, 0o755)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} else {
		if !fi.IsDir() {
			panic(cfg.BaseDir + " is not dir")
		}
	}
	metaDBPath := filepath.Join(cfg.BaseDir, "meta.db")
	db, err := bbolt.Open(metaDBPath, 0o600, nil)
	if err != nil {
		panic(err)
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketNameBlob)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(bucketNameDeleteTombstone)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	srv := &Server{
		db:           db,
		blobDir:      filepath.Join(cfg.BaseDir, "blobs"),
		writeDio:     cfg.WriteDio,
		blobChecksum: cfg.BlobChecksum,
	}

	if srv.writeDio {
		srv._2MBytesPool = sync.Pool{New: func() interface{} {
			ptr := directio.AlignedBlock(2 * 1024 * 1024)
			return &ptr
		}}
		srv.writeFileOpener = directio.OpenFile
	} else {
		srv._2MBytesPool = sync.Pool{New: func() interface{} {
			ptr := make([]byte, 2*1024*1024)
			return &ptr
		}}
		srv.writeFileOpener = os.OpenFile
	}

	if srv.blobChecksum {
		srv.signManager = NewBlobSignManager()
	}

	srv.gcController = &gcController{
		gcInterval: 30 * time.Second,
		db:         db,
		blobDir:    srv.blobDir,
	}
	go srv.gcController.runGCLoop()
	return srv
}

const (
	keyLen          = 8 + 4
	hashToDirPrefix = 2
	SignBlobSize    = 65536 // 64K
)

func getBlobPath(in *goproto.BlobKey) string {
	blobKey := formatBlobKey(in)
	hashedBlobKeyRaw := sha256.Sum256(blobKey)
	encodedHashString := hex.EncodeToString(hashedBlobKeyRaw[:])
	return filepath.Join(encodedHashString[:hashToDirPrefix], encodedHashString+"_"+hex.EncodeToString(blobKey))
}

func signSuffix() string {
	return ".sign"
}

var (
	bucketNameBlob            = []byte("blobs")
	bucketNameDeleteTombstone = []byte("delete")

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
	if req.BlobSize%SignBlobSize != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid blobsize %d, must align to 4096", req.BlobSize)
	}
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
		if err != nil {
			return err
		}
		return errors.Join(tx.Bucket(bucketNameDeleteTombstone).Put(blobKey, nil),
			tx.Bucket(bucketNameBlob).Put(blobKey, bin))
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
		if len(key) == 0 {
			return nil
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
			key, _ = cur.Next()
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

func (s *Server) CheckBlob(ctx context.Context, req *goproto.BlobKey) (*goproto.CheckBlobResp, error) {
	if !s.blobChecksum {
		return nil, status.Errorf(codes.FailedPrecondition, "blob checksum not opened")
	}
	_, err := s.StatBlob(ctx, req)
	if err != nil {
		return nil, err
	}
	f, err := s.openBlobReader(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open blob error: %s", err)
	}
	defer f.Close()
	resp := &goproto.CheckBlobResp{
		ChecksumOk: true,
	}
	_, err = io.Copy(io.Discard, f)

	if err != nil {
		resp.ChecksumOk = false
		resp.Reason = err.Error()
	}
	return resp, nil
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

func parseVolumeIDAndSeq(in string) (*goproto.BlobKey, error) {
	volumeIDAndSeq := strings.Split(in, "/")
	err := fmt.Errorf("invalid resource name %s", in)
	if len(volumeIDAndSeq) != 2 {
		return nil, err
	}
	volumeID, err := strconv.ParseInt(volumeIDAndSeq[0], 10, 64)
	if err != nil {
		return nil, err
	}
	seq, err := strconv.ParseInt(volumeIDAndSeq[1], 10, 64)
	if err != nil {
		return nil, err
	}
	return &goproto.BlobKey{
		VolumeId: uint64(volumeID),
		Seq:      uint32(seq),
	}, nil
}
func (s *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	blobKey := strings.TrimPrefix(r.RequestURI, "/")
	blobKeyProto, err := parseVolumeIDAndSeq(blobKey)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		_, _ = rw.Write([]byte(err.Error()))
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.readBlob(rw, r, blobKeyProto)
	case http.MethodPut:
		s.writeBlob(rw, r, blobKeyProto)
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
				_, _ = rw.Write([]byte("blob not found: " + r.RequestURI))
				return
			}
		}

		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("read blob error: " + err.Error()))
		return
	}
	if blobInfo.State != goproto.BlobState_NORMAL {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("read unreadable blob in state: " + blobInfo.State.String()))
		return
	}

	rw.Header().Set("Content-Length", strconv.FormatInt(int64(blobInfo.BlobSize), 10))
	rw.Header().Set("Content-Type", "application/octet-stream")

	f, err := s.openBlobReader(in)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("open blob file error: %w", err)
		return
	}

	defer f.Close()
	buf := s._2MBytesPool.Get().(*[]byte)
	defer s._2MBytesPool.Put(buf)
	n, err := io.CopyBuffer(rw, f, *buf)
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
}

func (s *Server) writeBlob(rw http.ResponseWriter, r *http.Request, in *goproto.BlobKey) {
	blobInfo, err := s.StatBlob(r.Context(), in)
	formattedBlobKey := formatBlobKey(in)
	if err != nil {
		code, ok := status.FromError(err)
		if ok {
			if code.Code() == codes.NotFound {
				rw.WriteHeader(http.StatusNotFound)
				_, _ = rw.Write([]byte("write not found: " + r.RequestURI))
				return
			}
		}

		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("write blob error: " + err.Error()))
		return
	}

	if blobInfo.State != goproto.BlobState_INIT {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("write unwritable blob in state: " + blobInfo.State.String()))
		return
	}

	err = s.atomicUpdateBlobState(in, goproto.BlobState_PENDING)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("write blob atomicUpdateBlobState error: " + err.Error()))
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
			_, _ = rw.Write([]byte(writeErr.Error()))
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
				_, _ = rw.Write([]byte(err.Error()))
				return
			}
		}
	}()

	defer r.Body.Close()

	blobPath := filepath.Join(s.blobDir, getBlobPath(in))
	blobDir := filepath.Dir(blobPath)
	err = os.MkdirAll(blobDir, 0o755)
	if err != nil {
		writeErr = fmt.Errorf("mkdir %s error: %s", blobDir, err)
		return
	}

	// opened file can not be deferred close may cause leak
	df, err := s.writeFileOpener(blobPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		writeErr = fmt.Errorf("open file %s error: %s", blobPath, err)
		return
	}
	var writer io.Writer
	var signer *DataSigner
	if s.blobChecksum {
		signer = NewDataSigner(SignBlobSize, blobInfo.BlobSize)
		writer = io.MultiWriter(df, signer)
	} else {
		writer = df
	}

	buf := s._2MBytesPool.Get().(*[]byte)
	defer s._2MBytesPool.Put(buf)

	written, err := io.CopyBuffer(writer, r.Body, *buf)
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

	if s.blobChecksum {
		err = os.WriteFile(blobPath+signSuffix(), SignMarshall(signer.Sum()), 0o600)
		if err != nil {
			writeErr = fmt.Errorf("write file sign error: %w", err)
			return
		}
	}
}
