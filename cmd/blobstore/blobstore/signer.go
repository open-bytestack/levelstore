package blobstore

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/groupcache/lru"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	ErrSignIncomplete = errors.New("signature data incomplete")
	ErrBlobCrc        = errors.New("blob crc32 incorrect")
	ErrBlobIncomplete = errors.New("blob incomplete")
)

type DataSigner struct {
	cs        []uint32
	blockData []byte
	nx        int
	bs        int
	len       uint64
}

func SignMarshall(signs []uint32) []byte {
	var a [4]byte
	signData := make([]byte, 0, len(signs)*4)
	for _, s := range signs {
		binary.BigEndian.PutUint32(a[:], s)
		signData = append(signData, a[:]...)
	}
	return signData
}

func SignUnmarshall(signData []byte) ([]uint32, error) {
	if len(signData)%4 != 0 {
		return nil, ErrSignIncomplete
	}
	signs := make([]uint32, 0, len(signData)/4)
	for i := 0; i < len(signData); i += 4 {
		sign := binary.BigEndian.Uint32(signData[i : i+4])
		signs = append(signs, sign)
	}
	return signs, nil
}

func SignGenerate(p []byte) uint32 {
	return crc32.ChecksumIEEE(p)
}

func NewDataSigner(blobSize int, dataSize uint64) *DataSigner {
	return &DataSigner{
		bs:        blobSize,
		cs:        make([]uint32, 0, (dataSize+uint64(blobSize)-1)/uint64(blobSize)),
		blockData: make([]byte, blobSize),
	}
}

func (s *DataSigner) blockGeneric(p []byte) {
	for i := 0; i <= len(p)-s.bs; i += s.bs {
		q := p[i:]
		q = q[:s.bs:s.bs]
		sign := SignGenerate(q)
		s.cs = append(s.cs, sign)
	}
}

func (s *DataSigner) Write(p []byte) (nn int, err error) {
	// Note that we currently call block or blockGeneric
	// directly (guarded using haveAsm) because this allows
	// escape analysis to see that p and d don't escape.
	nn = len(p)
	s.len += uint64(nn)
	if s.nx > 0 {
		n := copy(s.blockData[s.nx:], p)
		s.nx += n
		if s.nx == s.bs {
			s.blockGeneric(s.blockData[:])
			s.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= s.bs {
		n := len(p) &^ (s.bs - 1)
		s.blockGeneric(p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		s.nx = copy(s.blockData[:], p)
	}
	return
}

func (s *DataSigner) Sum() []uint32 {
	if s.nx > 0 {
		padding := s.bs - s.nx
		_, _ = s.Write(make([]byte, padding))
	}
	return s.cs
}

type BlobSign struct {
	signs []uint32
}

func NewBlobSign(signB []byte) (*BlobSign, error) {
	s, err := SignUnmarshall(signB)
	if err != nil {
		return nil, err
	}
	return &BlobSign{
		signs: s,
	}, nil
}

func (bs *BlobSign) Bytes() []byte {
	return SignMarshall(bs.signs)
}

func (bs *BlobSign) Get(seq int64) uint32 {
	return bs.signs[seq]
}

type blobSignManager struct {
	c  *lru.Cache
	mu sync.Mutex
}

func NewBlobSignManager() *blobSignManager {
	return &blobSignManager{
		c: lru.New(1000),
	}
}

func (bsm *blobSignManager) Get(in *goproto.BlobKey) (bs *BlobSign, ok bool) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()
	v, ok := bsm.c.Get(fmt.Sprintf("%d/%d", in.VolumeId, in.Seq))
	if ok {
		bs = v.(*BlobSign)
	}
	return bs, ok
}

func (bsm *blobSignManager) Add(in *goproto.BlobKey, bs *BlobSign) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()
	bsm.c.Add(fmt.Sprintf("%d/%d", in.VolumeId, in.Seq), bs)
}

type signedBlobFile struct {
	*os.File
	size int64
	s    *BlobSign
	buf  [SignBlobSize]byte
}

func (f *signedBlobFile) Seek(offset int64, whence int) (int64, error) {
	return f.File.Seek(offset, whence)
}

func (f *signedBlobFile) Read(p []byte) (n int, err error) {
	offset, err := f.File.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	droppedSize := offset % int64(SignBlobSize)
	if _, err := f.File.Seek(offset-droppedSize, io.SeekStart); err != nil {
		return n, err
	}

	bufSize := len(p)
	for n < bufSize {
		fn, err := f.File.Read(f.buf[:])
		if err != nil && err != io.EOF {
			return n, err
		} else if err == io.EOF {
			break
		} else if fn != SignBlobSize {
			return n, ErrBlobIncomplete
		}

		desSign := f.s.Get(offset / SignBlobSize)
		if desSign != SignGenerate(f.buf[:]) {
			return n, ErrBlobCrc
		}

		cn := copy(p[n:], f.buf[droppedSize:])
		offset += int64(cn)
		n += cn
		if _, err := f.File.Seek(offset, io.SeekStart); err != nil {
			return n, err
		}

		if offset >= f.size {
			n -= int(offset - f.size)
			return n, io.EOF
		}
		droppedSize = 0
	}
	return n, err
}

func (f *signedBlobFile) Close() error {
	return f.File.Close()
}

func (s *Server) openBlobReader(in *goproto.BlobKey) (io.ReadSeekCloser, error) {
	path := filepath.Join(s.blobDir, getBlobPath(in))
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open blob file error: %w", err)
	}
	if !s.blobChecksum {
		return f, nil
	}

	bs, ok := s.signManager.Get(in)
	if !ok {
		signFile, err := os.Open(path + signSuffix())
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("open sign file error: %w", err)
			}
		} else {
			signBytes, err := io.ReadAll(signFile)
			if err != nil {
				return nil, fmt.Errorf("read sign file error: %w", err)
			}
			sign, err := NewBlobSign(signBytes)
			if err != nil {

				return nil, fmt.Errorf("unmarshal sign file error: %w", err)
			}
			s.signManager.Add(in, sign)
			bs = sign
		}
	}
	if bs != nil {
		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}
		return &signedBlobFile{
			File: f,
			size: fi.Size(),
			s:    bs,
			buf:  [65536]byte{},
		}, nil
	}
	return f, nil
}
