package blobstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ncw/directio"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (s *Server) Read(request *bytestream.ReadRequest, server bytestream.ByteStream_ReadServer) error {
	blockKeyProto, err := parseVolumeIDAndSeq(request.ResourceName)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "parse volume id and seq error: %s", err)
	}
	blobInfo, err := s.StatBlob(server.Context(), blockKeyProto)
	if err != nil {
		return err
	}
	if blobInfo.State != goproto.BlobState_NORMAL {
		return status.Errorf(codes.InvalidArgument, "read unreadable blob in state: %s", blobInfo.State.String())
	}

	path := filepath.Join(s.blockDir, getBlobPath(blockKeyProto))
	f, err := os.Open(path)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "parse volume id and seq error: %s", err)
	}
	defer f.Close()
	if request.ReadOffset != 0 {
		_, err = f.Seek(request.ReadOffset, io.SeekStart)
		if err != nil {
			return status.Errorf(codes.Internal, "seek file error: %s", err)
		}
	}
	buf := s._4MBytesPool.Get().([]byte)
	defer s._4MBytesPool.Put(buf)
	var lr io.Reader = f
	if request.ReadLimit != 0 {
		lr = io.LimitReader(f, request.ReadLimit)
	}
	for {
		n, err := lr.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return status.Errorf(codes.Internal, "read file error: %s", err)
			}
			err = server.Send(&bytestream.ReadResponse{Data: buf[:n]})
			if err != nil {
				return status.Errorf(codes.Internal, "send data error: %s", err)
			}
			return nil
		}
		err = server.Send(&bytestream.ReadResponse{Data: buf[:n]})
		if err != nil {
			return err
		}
		continue
	}
}

func (s *Server) Write(server bytestream.ByteStream_WriteServer) (err error) {
	wr, err := server.Recv()
	if err != nil {
		return err
	}
	blockKeyProto, err := parseVolumeIDAndSeq(wr.ResourceName)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "parse volume id and seq error: %s", err)
	}
	blobInfo, err := s.StatBlob(server.Context(), blockKeyProto)
	if err != nil {
		return err
	}
	formattedBlobKey := formatBlobKey(blockKeyProto)

	if blobInfo.State != goproto.BlobState_INIT {
		return status.Error(codes.InvalidArgument, "write unwritable blob in state: %s"+blobInfo.State.String())
	}
	err = s.atomicUpdateBlobState(blockKeyProto, goproto.BlobState_PENDING)
	if err != nil {
		return status.Errorf(codes.Internal, "write blob atomicUpdateBlobState error: %s", err)
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
			uerr := s.db.Update(func(tx *bbolt.Tx) error {
				return tx.Bucket(bucketNameBlob).Put(formattedBlobKey, bin)
			})
			if uerr != nil {
				zap.L().With(
					zap.Error(err),
					zap.Uint64("volume_id", blockKeyProto.VolumeId),
					zap.Uint32("seq", blockKeyProto.Seq),
					zap.String("blob_info", blobInfo.String()),
				).Error("update blob info error")
				err = status.Errorf(codes.Internal, "update meta db during write blob error: %s", err)
				return
			}
			_, isGRPCError := status.FromError(writeErr)
			if isGRPCError {
				err = writeErr
				return
			}
			err = status.Errorf(codes.Internal, "write blob failed: %s", writeErr)
			return
		} else {
			blobInfo.State = goproto.BlobState_NORMAL
			bin, _ := proto.Marshal(blobInfo)
			uerr := s.db.Update(func(tx *bbolt.Tx) error {
				return tx.Bucket(bucketNameBlob).Put(formattedBlobKey, bin)
			})
			if uerr != nil {
				zap.L().With(
					zap.Error(err),
					zap.Uint64("volume_id", blockKeyProto.VolumeId),
					zap.Uint32("seq", blockKeyProto.Seq),
					zap.String("blob_info", blobInfo.String()),
				).Error("update blob info error")
				err = status.Errorf(codes.Internal, "update meta db during write blob error: %s", err)
				return
			}
			err = nil
			return
		}
	}()

	blobPath := filepath.Join(s.blockDir, getBlobPath(blockKeyProto))
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

	var finishWrite bool
	buf := s._4KBytesPoolAlignedBlock.Get().([]byte)
	defer s._4KBytesPoolAlignedBlock.Put(buf)

	var committedSize int64
	writeOnceFn := func(InputData []byte) error {
		written, err := io.CopyBuffer(df, bytes.NewReader(InputData), buf)
		if err != nil {
			return fmt.Errorf("copy body to file error: %s", err)
		}
		if written != int64(len(wr.Data)) {
			return fmt.Errorf("copy body size mismatched with content-length, should be %d but %d written", len(wr.Data), written)
		}
		committedSize += written
		return nil
	}

	writeErr = writeOnceFn(wr.Data)
	if writeErr != nil {
		return
	}
	finishWrite = wr.FinishWrite
	for !finishWrite {
		wr, writeErr = server.Recv()
		if writeErr != nil {
			return
		}
		writeErr = writeOnceFn(wr.Data)
		if writeErr != nil {
			return
		}
		finishWrite = wr.FinishWrite
	}

	return server.SendAndClose(&bytestream.WriteResponse{
		CommittedSize: committedSize,
	})
}

func (s *Server) QueryWriteStatus(ctx context.Context, request *bytestream.QueryWriteStatusRequest) (*bytestream.QueryWriteStatusResponse, error) {
	panic("implement me")
}

var _ bytestream.ByteStreamServer = (*Server)(nil)
