package main

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"github.com/nx-a/ring/cmd/migrate"
	"github.com/nx-a/ring/internal/adapter/storage"
	bucketstor "github.com/nx-a/ring/internal/adapter/storage/bucket"
	controlstor "github.com/nx-a/ring/internal/adapter/storage/control"
	datastor "github.com/nx-a/ring/internal/adapter/storage/data"
	pointstor "github.com/nx-a/ring/internal/adapter/storage/point"
	tokenstor "github.com/nx-a/ring/internal/adapter/storage/token"
	"github.com/nx-a/ring/internal/adapter/tcp"
	"github.com/nx-a/ring/internal/adapter/web/server"
	"github.com/nx-a/ring/internal/adapter/web/server/route"
	"github.com/nx-a/ring/internal/core/service/bucket"
	"github.com/nx-a/ring/internal/core/service/control"
	"github.com/nx-a/ring/internal/core/service/data"
	"github.com/nx-a/ring/internal/core/service/point"
	"github.com/nx-a/ring/internal/core/service/token"
	"github.com/nx-a/ring/internal/engine"
	"github.com/nx-a/ring/internal/engine/env"
	"github.com/nx-a/ring/internal/engine/event"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

//go:embed config.yml
var config embed.FS

//go:embed tls/*
var tlsDir embed.FS

func main() {
	ws, q, closes := dependency()
	ctx, cancel := context.WithCancel(context.Background())
	go ws.Listen(ctx, cancel)
	go q.Run(ctx)
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-done
		log.Info("shutting down...")
		cancel()
	}()
	<-ctx.Done()
	for _, cl := range closes {
		if err := cl.Close(); err != nil {
			log.Error(err)
		}
	}
}

var ignoreDir = "/home/nx/project/go/ring/"

func dependency() (*server.Server, *tcp.Server, []engine.Closable) {
	_event := event.New()
	cfg := env.New(config)
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := f.File[len(ignoreDir):]

			return "", fmt.Sprintf(" %s:%d", filename, f.Line)
		},
	})
	fmt.Println(os.Getwd())
	log.Debug(cfg.Get("service.prod"))
	if cfg.Get("service.prod") == "false" {
		err := migrate.Run(cfg)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}
	db := storage.New(cfg)
	backetStor := bucketstor.New(db.Get())
	dataStor := datastor.New(db.Get())
	pointStor := pointstor.New(db.Get())
	tokenStor := tokenstor.New(db.Get())

	//core
	bucketService := bucket.New(backetStor, _event)
	controlStor := controlstor.New(db.Get(), bucketService)
	pointService := point.New(pointStor, bucketService)
	dataService := data.New(dataStor, bucketService, pointService, _event)
	tokenService := token.New(tokenStor, bucketService)
	controlService := control.New(controlStor)

	ws := server.New(cfg, &server.Services{
		TokenService:  tokenService,
		BucketService: bucketService,
		DataService:   dataService,
		PointService:  pointService,
	})
	route.Bucket(ws, bucketService)
	route.Control(ws, controlService)
	route.Point(ws, pointService)
	route.Token(ws, tokenService)
	route.Data(ws, dataService)
	tlsConfig, err := loadTLSConfigFromFiles("tls/server.crt", "tls/server.key")
	if err != nil {
		log.Fatalf("Failed to load TLS files: %v", err)
	}
	srv, err := tcp.NewServer(":7888", tlsConfig)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	srv.AddHandler(dataService, tokenService)
	return ws, srv, []engine.Closable{db, ws, srv}
}
func loadTLSConfigFromFiles(certFile, keyFile string) (*tls.Config, error) {
	certData, err := tlsDir.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	// Read key file
	keyData, err := tlsDir.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"ring-quic"},
	}, nil
}
