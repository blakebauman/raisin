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
	"github.com/blakebauman/raisin/internal/automation"
	"github.com/blakebauman/raisin/internal/billing"
	"github.com/blakebauman/raisin/internal/config"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/domain"
	"github.com/blakebauman/raisin/internal/email"
	"github.com/blakebauman/raisin/internal/events"
	"github.com/blakebauman/raisin/internal/ippool"
	"github.com/blakebauman/raisin/internal/jobs"
	jobhandlers "github.com/blakebauman/raisin/internal/jobs/handlers"
	"github.com/blakebauman/raisin/internal/logging"
	"github.com/blakebauman/raisin/internal/sender"
	"github.com/blakebauman/raisin/internal/storage"
	"github.com/blakebauman/raisin/internal/suppression"
	"github.com/blakebauman/raisin/internal/webhook"
	"github.com/redis/go-redis/v9"
)

func main() {
	_ = godotenv.Load()
	logging.Setup()
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	snd, err := sender.New(cfg)
	if err != nil {
		log.Fatalf("sender: %v", err)
	}

	store, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	asynqClient := jobs.NewClient(cfg)
	defer asynqClient.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr(cfg.RedisURL)})
	defer rdb.Close()

	wh := &webhook.Service{Pool: pool, Client: asynqClient}
	bill := &billing.Service{
		Pool:            pool,
		SecretKey:       cfg.StripeSecretKey,
		PricePro:        cfg.StripePricePro,
		PriceProAnnual:  cfg.StripePriceProAnnual,
		ProMonthlyQuota: cfg.StripeProQuota,
	}
	supp := &suppression.Service{Pool: pool}
	auto := &automation.Service{Pool: pool, Client: asynqClient}
	proc := &events.Processor{Pool: pool, Redis: rdb, Webhooks: wh, Suppressions: supp, Billing: bill, Automations: auto}
	dom := &domain.Service{Pool: pool, Identity: sender.NewIdentity(cfg)}
	ips := &ippool.Service{Pool: pool, ConfigSets: sender.NewConfigurationSets(cfg), LocalIPs: cfg.SenderDriver != "ses"}
	emails := &email.Service{Pool: pool, Client: asynqClient, Storage: store}

	h := &jobhandlers.Handlers{
		Cfg: cfg, Pool: pool, Sender: snd, Webhooks: wh, Billing: bill, Events: proc,
		Asynq: asynqClient, Storage: store, Domains: dom, Emails: emails, IPPools: ips,
	}

	server := jobs.NewServer(cfg)
	go func() {
		log.Printf("asynq worker starting")
		if err := server.Run(h.Mux()); err != nil {
			log.Fatalf("asynq: %v", err)
		}
	}()

	scheduler := jobs.NewScheduler(cfg)
	if task, err := jobs.NewIPWarmupTickTask(); err == nil {
		if _, err := scheduler.Register("0 0 * * *", task); err != nil {
			log.Printf("warmup scheduler register: %v", err)
		} else {
			go func() {
				log.Printf("asynq scheduler starting (ip warmup daily UTC)")
				if err := scheduler.Run(); err != nil {
					log.Printf("scheduler: %v", err)
				}
			}()
		}
	}
	// SES → SNS → SQS consumer (optional when queue URL set)
	if cfg.SQSEventsQueueURL != "" {
		consumer, err := events.NewSQSConsumer(cfg, proc)
		if err != nil {
			log.Printf("sqs consumer disabled: %v", err)
		} else {
			go func() {
				log.Printf("sqs consumer listening on %s", cfg.SQSEventsQueueURL)
				if err := consumer.Run(ctx); err != nil && err != context.Canceled {
					log.Printf("sqs: %v", err)
				}
			}()
		}
	}

	httpSrv := &http.Server{
		Addr:              cfg.WorkerAddr,
		Handler:           h.TrackingHTTP(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("worker http (tracking) on %s", cfg.WorkerAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	server.Shutdown()
	scheduler.Shutdown()
	shCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = httpSrv.Shutdown(shCtx)
	_ = os.Stdout
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
