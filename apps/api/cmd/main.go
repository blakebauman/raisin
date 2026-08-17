package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/raisin-run/raisin/internal/audience"
	"github.com/raisin-run/raisin/internal/billing"
	"github.com/raisin-run/raisin/internal/broadcast"
	"github.com/raisin-run/raisin/internal/config"
	"github.com/raisin-run/raisin/internal/db"
	"github.com/raisin-run/raisin/internal/domain"
	"github.com/raisin-run/raisin/internal/email"
	"github.com/raisin-run/raisin/internal/httpapi"
	"github.com/raisin-run/raisin/internal/inbound"
	"github.com/raisin-run/raisin/internal/jobs"
	"github.com/raisin-run/raisin/internal/logging"
	"github.com/raisin-run/raisin/internal/metrics"
	"github.com/raisin-run/raisin/internal/sender"
	"github.com/raisin-run/raisin/internal/storage"
	"github.com/raisin-run/raisin/internal/suppression"
	"github.com/raisin-run/raisin/internal/template"
	"github.com/raisin-run/raisin/internal/webhook"
	"github.com/redis/go-redis/v9"
)

func main() {
	_ = godotenv.Load()
	logging.Setup()
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	asynqClient := jobs.NewClient(cfg)
	defer asynqClient.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr(cfg.RedisURL)})
	defer rdb.Close()

	store, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	srv := &httpapi.Server{
		Cfg:          cfg,
		Pool:         pool,
		Asynq:        asynqClient,
		Redis:        rdb,
		Emails:       &email.Service{Pool: pool, Client: asynqClient, Storage: store},
		Domains:      &domain.Service{Pool: pool, Identity: sender.NewIdentity(cfg)},
		Webhooks:     &webhook.Service{Pool: pool, Client: asynqClient},
		Suppressions: &suppression.Service{Pool: pool},
		Audience:     &audience.Service{Pool: pool},
		Templates:    &template.Service{Pool: pool},
		Broadcasts:   &broadcast.Service{Pool: pool, Client: asynqClient},
		Billing:      &billing.Service{Pool: pool, SecretKey: cfg.StripeSecretKey},
		Metrics:      &metrics.Service{Pool: pool},
		Inbound:      &inbound.Service{Pool: pool, Storage: store},
	}

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func redisAddr(u string) string {
	if len(u) > 8 && u[:8] == "redis://" {
		u = u[8:]
	}
	for i, c := range u {
		if c == '/' {
			return u[:i]
		}
	}
	if u == "" {
		return "localhost:6379"
	}
	return u
}
