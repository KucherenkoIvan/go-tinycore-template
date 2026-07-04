package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/KucherenkoIvan/go-kernel/app"
	"github.com/KucherenkoIvan/go-kernel/config"
	"github.com/KucherenkoIvan/go-kernel/events"
	"github.com/KucherenkoIvan/go-kernel/health"
	"github.com/KucherenkoIvan/go-kernel/httpapi"
	"github.com/KucherenkoIvan/go-kernel/observability"
	"github.com/gin-gonic/gin"

	"github.com/KucherenkoIvan/go-tinycore-template/internal/features/changeme"
	"github.com/KucherenkoIvan/go-tinycore-template/internal/shared/infra/storage"
)

type AppConfig struct {
	HTTPAddr string `env:"HTTP_ADDR" default:":8080"`
	DBPath   string `env:"DB_PATH" default:"app.db"`
}

func main() {
	// 1. observe first
	logger := observability.NewLogger("tinycore") // rename with the project
	slog.SetDefault(logger)

	// 2. config: loaded once, passed down as values
	cfg := config.MustLoad[AppConfig]()
	ctx := context.Background()

	// 3. infra
	db, err := storage.Open(ctx, cfg.DBPath)
	if err != nil {
		logger.Error("opening storage", "error", err)
		os.Exit(1)
	}
	pub := events.NewChannelPublisher(events.WithLogger(logger))

	// 4. features
	_ = changeme.New(db, pub) // wire route registration here when transport adapters appear

	// 5. transport — no API endpoints yet; the router serves health only.
	// When a feature grows handlers: feature.Handlers.RegisterRoutes(r.Group("/api"))
	r := httpapi.NewRouter(httpapi.WithLogger(logger))

	// 6. health
	checker := health.NewChecker(health.Check{Name: "sqlite", Check: db.Ping})
	r.GET("/livez", gin.WrapH(health.Liveness()))
	r.GET("/healthz", gin.WrapH(checker.Handler()))

	// 7. run — registration order is dependency order
	if err := app.Run(ctx,
		app.Adapter{Name: "sqlite", Close: func(context.Context) error { return db.Close() }},
		app.Adapter{Name: "events", Run: pub.Run, Close: pub.Close},
		app.Adapter{Name: "http", Run: func(ctx context.Context) error { return httpapi.Run(ctx, r, cfg.HTTPAddr) }},
	); err != nil {
		logger.Error("service failed", "error", err)
		os.Exit(1)
	}
}
