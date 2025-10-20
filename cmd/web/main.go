package main

import (
	"context"
	"embed"
	"github.com/nx-a/ring/cmd/migrate"
	"github.com/nx-a/ring/internal/adapter/storage"
	bucketstor "github.com/nx-a/ring/internal/adapter/storage/bucket"
	controlstor "github.com/nx-a/ring/internal/adapter/storage/control"
	datastor "github.com/nx-a/ring/internal/adapter/storage/data"
	pointstor "github.com/nx-a/ring/internal/adapter/storage/point"
	tokenstor "github.com/nx-a/ring/internal/adapter/storage/token"
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
	"syscall"
)

//go:embed config.yml
var config embed.FS

func main() {
	ws, closes := dependency()
	ctx, cancel := context.WithCancel(context.Background())
	ws.Listen(ctx, cancel)
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

func dependency() (*server.Server, []engine.Closable) {
	_event := event.New()
	cfg := env.New(config)
	log.SetLevel(log.DebugLevel)
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
	return ws, []engine.Closable{db, ws}
}
