package blobstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (s *Server) Read(request *bytestream.ReadRequest, server bytestream.ByteStream_ReadServer) error {
	blobKeyProto, err := parseVolumeIDAndSeq(request.ResourceName)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "parse volume id and seq error: %s", err)
	}
	blobInfo, err := s.StatBlob(server.Context(), blobKeyProto)
	if err != nil {
		return err
	}
	if blobInfo.State != goproto.BlobState_NORMAL {
		return status.Errorf(codes.InvalidArgument, "read unreadable blob in state: %s", blobInfo.State.String())
	}
	br, err := s.openBlobReader(blobKeyProto)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "parse volume id and seq error: %s", err)
	}
	defer br.Close()
	if request.ReadOffset != 0 {
		_, err = br.Seek(request.ReadOffset, io.SeekStart)
		if err != nil {
			return status.Errorf(codes.Internal, "seek file error: %s", err)
		}
	}
	buf := s._2MBytesPool.Get().(*[]byte)
	defer s._2MBytesPool.Put(buf)
	var lr io.Reader = br
	if request.ReadLimit != 0 {
		lr = io.LimitReader(br, request.ReadLimit)
	}
	for {
		n, err := lr.Read(*buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return status.Errorf(codes.Internal, "read file error: %s", err)
			}
			err = server.Send(&bytestream.ReadResponse{Data: (*buf)[:n]})
			if err != nil {
				return status.Errorf(codes.Internal, "send data error: %s", err)
			}
			return nil
		}
		err = server.Send(&bytestream.ReadResponse{Data: (*buf)[:n]})
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
	blobKeyProto, err := parseVolumeIDAndSeq(wr.ResourceName)
	if err != nil {
		_ = server.SendAndClose(&bytestream.WriteResponse{CommittedSize: 0})
		return status.Errorf(codes.InvalidArgument, "parse volume id and seq error: %s", err)
	}
	blobInfo, err := s.StatBlob(server.Context(), blobKeyProto)
	if err != nil {
		return err
	}
	formattedBlobKey := formatBlobKey(blobKeyProto)
	if blobInfo.State != goproto.BlobState_INIT {
		_ = server.SendAndClose(&bytestream.WriteResponse{CommittedSize: 0})
		return status.Errorf(codes.InvalidArgument, "write unwritable blob in state: %s", blobInfo.State.String())
	}
	err = s.atomicUpdateBlobState(blobKeyProto, goproto.BlobState_PENDING)
	if err != nil {
		_ = server.SendAndClose(&bytestream.WriteResponse{CommittedSize: 0})
		return status.Errorf(codes.Internal, "write blob atomicUpdateBlobState error: %s", err)
	}

	var writeErr error = nil
	var committedSize int64

	defer func() {
		_ = server.SendAndClose(&bytestream.WriteResponse{
			CommittedSize: committedSize,
		})
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
					zap.Uint64("volume_id", blobKeyProto.VolumeId),
					zap.Uint32("seq", blobKeyProto.Seq),
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
					zap.Uint64("volume_id", blobKeyProto.VolumeId),
					zap.Uint32("seq", blobKeyProto.Seq),
					zap.String("blob_info", blobInfo.String()),
				).Error("update blob info error")
				err = status.Errorf(codes.Internal, "update meta db during write blob error: %s", err)
				return
			}
			err = nil
			return
		}
	}()

	blobPath := filepath.Join(s.blobDir, getBlobPath(blobKeyProto))
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

	var finishWrite bool
	buf := s._2MBytesPool.Get().(*[]byte)
	defer s._2MBytesPool.Put(buf)

	var writer io.Writer
	var signer *DataSigner
	if s.blobChecksum {
		signer = NewDataSigner(SignBlobSize, blobInfo.BlobSize)
		writer = io.MultiWriter(df, signer)
	} else {
		writer = df
	}

	writeOnceFn := func(InputData []byte) error {
		var written int64
		var err error
		if s.writeDio {
			written, err = io.CopyBuffer(writer, bytes.NewReader(InputData), *buf)
			if err != nil {
				return fmt.Errorf("copy body to file error: %s", err)
			}
		} else {
			w, err := writer.Write(InputData)
			written = int64(w)
			if err != nil {
				return fmt.Errorf("copy body to file error: %s", err)
			}
		}
		if written != int64(len(wr.Data)) {
			return fmt.Errorf("body size copied mismatched with content-length, should be %d but %d written", len(wr.Data), written)
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
		if errors.Is(writeErr, io.EOF) {
			if committedSize != int64(blobInfo.BlobSize) {
				writeErr = fmt.Errorf("unexpectEOF: size copied mismatched with blobsize, should be %d but be %d", blobInfo.BlobSize, committedSize)
			} else {
				writeErr = nil
			}
			break
		}
		if writeErr != nil {
			return
		}
		if wr.WriteOffset != committedSize {
			return status.Errorf(codes.InvalidArgument, "write offset should be %d but be %d", committedSize, wr.WriteOffset)
		}
		writeErr = writeOnceFn(wr.Data)
		if writeErr != nil {
			return
		}
		finishWrite = wr.FinishWrite
	}

	if s.blobChecksum {
		err = os.WriteFile(blobPath+signSuffix(), SignMarshall(signer.Sum()), 0o600)
		if err != nil {
			writeErr = fmt.Errorf("write file sign error: %w", err)
			return
		}
	}
	return nil
}

func (s *Server) QueryWriteStatus(ctx context.Context, request *bytestream.QueryWriteStatusRequest) (*bytestream.QueryWriteStatusResponse, error) {
	panic("implement me")
}

var _ bytestream.ByteStreamServer = (*Server)(nil)
