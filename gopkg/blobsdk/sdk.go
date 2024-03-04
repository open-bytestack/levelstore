package blobsdk

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"net/http"
)

type Client struct {
	goproto.BlobServiceClient
	addr       string
	httpClient *http.Client
}

func NewClient(addr string) (Interface, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		BlobServiceClient: goproto.NewBlobServiceClient(conn),
		addr:              addr,
		httpClient:        &http.Client{},
	}, nil
}

type limitReadCloser struct {
	origin        io.ReadCloser
	limiterReader io.Reader
	err           error
}

func (l *limitReadCloser) Read(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	return l.limiterReader.Read(p)
}

func (l *limitReadCloser) Close() error {
	if l.origin != nil {
		l.err = errors.Join(l.err, l.origin.Close())
	}
	return l.err
}

func (c *Client) NewBlobReader(in *goproto.BlobKey) io.ReadCloser {
	blobInfo, err := c.StatBlob(context.TODO(), in)
	if err != nil {
		return &limitReadCloser{err: err}
	}
	if blobInfo.State != goproto.BlobState_NORMAL {
		return &limitReadCloser{err: fmt.Errorf("blob is in state[%s] not readable", blobInfo.State.String())}
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/%d/%d", c.addr, blobInfo.VolumeId, blobInfo.Seq), nil)
	if err != nil {
		return &limitReadCloser{err: err}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &limitReadCloser{err: err}
	}
	return &limitReadCloser{
		origin:        resp.Body,
		limiterReader: io.LimitReader(resp.Body, int64(blobInfo.BlobSize)),
		err:           err,
	}
}

type limitWriteCloser struct {
	origin       io.WriteCloser
	bytesWritten int64
	blobSize     int64
	err          error
	finished     chan struct{}
	closed       bool
}

func (l *limitWriteCloser) Write(p []byte) (int, error) {
	if l.bytesWritten+int64(len(p)) > l.blobSize {
		l.err = errors.Join(l.err, errors.New("blob is full"))
		return 0, l.err
	}

	n, err := l.origin.Write(p)
	l.bytesWritten += int64(n)
	l.err = errors.Join(l.err, err)
	return n, err
}

func (l *limitWriteCloser) Close() error {
	if l.closed {
		return l.err
	}
	if l.bytesWritten != l.blobSize {
		l.err = errors.Join(l.err, fmt.Errorf("data size written mismatched the blob size, should be %d but be %d", l.blobSize, l.bytesWritten))
		return l.err
	}

	err := l.origin.Close()
	if err != nil {
		l.err = errors.Join(l.err, err)
		return l.err
	}
	<-l.finished
	l.closed = true
	return l.err
}

type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, e.err
}

func (e *errorWriter) Close() error {
	return e.err
}

func (c *Client) NewBlobWriter(in *goproto.BlobKey) io.WriteCloser {
	blobInfo, err := c.StatBlob(context.TODO(), in)
	if err != nil {
		return &errorWriter{err: err}
	}
	if blobInfo.State != goproto.BlobState_INIT {
		return &errorWriter{err: fmt.Errorf("blob is in state[%s] not writable", blobInfo.State.String())}
	}

	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://%s/%d/%d", c.addr, blobInfo.VolumeId, blobInfo.Seq), bufio.NewReader(pr))
	if err != nil {
		return &errorWriter{err: err}
	}
	req.ContentLength = int64(blobInfo.BlobSize)

	lw := &limitWriteCloser{
		origin:       pw,
		bytesWritten: 0,
		blobSize:     int64(blobInfo.BlobSize),
		err:          nil,
		finished:     make(chan struct{}),
	}

	go func() {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lw.err = err
			return
		}
		defer resp.Body.Close()
		err = pr.Close()
		if err != nil {
			lw.err = errors.Join(lw.err, err)
		}

		if resp.StatusCode != 200 {
			content, err := io.ReadAll(resp.Body)
			if err != nil {
				lw.err = errors.Join(lw.err, err)
			}
			lw.err = errors.Join(lw.err, fmt.Errorf("write request error, status_code: %d, err_msg: %s", resp.StatusCode, string(content)))
		}
		zap.L().With(zap.Error(err)).Info("blob writer finished")
		lw.finished <- struct{}{}
	}()
	return lw
}
