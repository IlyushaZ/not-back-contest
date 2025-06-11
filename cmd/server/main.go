package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/cache"
	"github.com/IlyushaZ/not-back-contest/pkg/config"
	"github.com/IlyushaZ/not-back-contest/pkg/database"
	"github.com/IlyushaZ/not-back-contest/pkg/limiter"
	"github.com/IlyushaZ/not-back-contest/pkg/server"
	"github.com/IlyushaZ/not-back-contest/pkg/service"
	"github.com/redis/go-redis/v9"
)

var (
	shutdown = make(chan struct{})
)

const (
	gracefulTimeout = time.Second * 15
)

func main() {
	cfg := config.New()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	db, closeDB, err := database.New(cfg.PostgresAddr, cfg.PostgresDB, cfg.PostgresUser, cfg.PostgresPassword)
	if err != nil {
		log.Fatalf("### Can't init database: %v", err)
	}
	defer closeDB()

	redis, closeRedis, err := cache.NewRedis(cfg.RedisAddr, cfg.RedisUser, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("### Can't init redis: %v", err)
	}
	defer closeRedis()

	itemSvc, saleSvc := composeServices(db, redis, cfg)

	srv, err := server.New(cfg.ListenAddr, itemSvc, saleSvc)
	if err != nil {
		log.Fatalf("### Can't create server: %v", err)
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("### Can't listen and serve: %v", err)
		}
	}()
	slog.Info(fmt.Sprintf("HTTP server listening at %s", srv.Addr))

	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer cancel()

	srv.Shutdown(ctx)
}

func composeServices(db *sql.DB, redis *redis.Client, cfg *config.Config) (item service.Item, sale service.Sale) {
	item = &service.ItemGeneric{
		&database.ItemDatabase{db},
		database.NewCheckoutBatchingDatabase(db, cfg.CheckoutsBatchSize, cfg.CheckoutsFlushInterval),
		cfg.CheckoutTimeout,
	}

	if cfg.CacheCheckouts {
		item = &service.ItemCaching{item, redis, cfg.CheckoutTimeout}
	}

	item = &service.ItemLimiting{item, &limiter.Limiter{redis, cfg.PurchasesLimit}, cfg.LimiterFailOpen}
	item = &service.ItemLogging{item}

	sale = &service.SaleGeneric{
		&database.SaleDatabase{db},
	}

	return
}

func parseLogLevel(lvl string) slog.Level {
	switch lvl {
	case slog.LevelDebug.String():
		return slog.LevelDebug
	case slog.LevelInfo.String():
		return slog.LevelInfo
	case slog.LevelWarn.String():
		return slog.LevelWarn
	case slog.LevelError.String():
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
