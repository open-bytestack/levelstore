package blobstore

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
)

// grpcHandlerFunc copy from
// https://github.com/philips/grpc-gateway-example/blob/master/cmd/serve.go#L51C1-L62C1
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

func MainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "blobstore",
	}
	flagPort := cmd.Flags().String("port", ":8080", "port for service")
	flagBaseDir := cmd.Flags().String("basedir", "", "basedir")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg := zap.NewProductionConfig()
		cfg.Level.SetLevel(zap.InfoLevel)
		cfg.DisableStacktrace = true
		lgr, err := cfg.Build()
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(lgr)
		defer lgr.Sync()

		if *flagBaseDir == "" {
			zap.L().Fatal("basedir can not be empty")
		}

		srv := NewServer(*flagBaseDir)
		grpcServer := grpc.NewServer()
		goproto.RegisterBlobServiceServer(grpcServer, srv)
		bytestream.RegisterByteStreamServer(grpcServer, srv)

		httpSrv := http.Server{Handler: h2c.NewHandler(grpcHandlerFunc(grpcServer, srv), &http2.Server{}), Addr: *flagPort}

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			err := httpSrv.ListenAndServe()
			zap.L().With(zap.Error(err)).Info("listen and serve failed")
		}()

		select {
		case <-cmd.Context().Done():
			zap.L().With(zap.Error(cmd.Context().Err())).Info("context canceled, shutting down servers!")

		case sig := <-sigs:
			zap.L().With(zap.String("sig", sig.String())).Info("signal received, shutting down servers!")
		}

		grpcServer.GracefulStop()
		ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
		defer cancel()

		zap.L().With(zap.String("port", *flagPort)).Info("server started at")
		return httpSrv.Shutdown(ctx)
	}
	return cmd
}
