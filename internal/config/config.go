package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL         string
	RedisURL            string
	HTTPAddr            string
	WorkerAddr          string
	SMTPAddr            string
	SenderDriver        string // mailpit | ses
	MailpitURL          string
	JWTSecret           string
	ConsoleOrigin       string
	TrackingBaseURL     string
	AWSRegion           string
	AWSEndpointURL      string
	S3Bucket            string
	S3InboundBucket     string
	SQSEventsQueueURL   string
	SESConfigurationSet string
	StripeSecretKey     string
	StripeWebhookSecret string
	DefaultMonthlyQuota int
}

func Load() Config {
	return Config{
		DatabaseURL:         getenv("DATABASE_URL", "postgres://raisin:raisin@localhost:5432/raisin?sslmode=disable"),
		RedisURL:            getenv("REDIS_URL", "redis://localhost:6379"),
		HTTPAddr:            getenv("HTTP_ADDR", ":8080"),
		WorkerAddr:          getenv("WORKER_ADDR", ":8081"),
		SMTPAddr:            getenv("SMTP_ADDR", ":2525"),
		SenderDriver:        strings.ToLower(getenv("SENDER_DRIVER", "mailpit")),
		MailpitURL:          getenv("MAILPIT_URL", "http://localhost:8025"),
		JWTSecret:           getenv("JWT_SECRET", "dev-jwt-secret-change-me-in-production"),
		ConsoleOrigin:       getenv("CONSOLE_ORIGIN", "http://localhost:3000"),
		TrackingBaseURL:     getenv("TRACKING_BASE_URL", "http://localhost:8081"),
		AWSRegion:           getenv("AWS_REGION", "us-east-1"),
		AWSEndpointURL:      getenv("AWS_ENDPOINT_URL", ""),
		S3Bucket:            getenv("S3_BUCKET", "raisin-attachments"),
		S3InboundBucket:     getenv("S3_INBOUND_BUCKET", ""),
		SQSEventsQueueURL:   getenv("SQS_EVENTS_QUEUE_URL", ""),
		SESConfigurationSet: getenv("SES_CONFIGURATION_SET", "raisin-events"),
		StripeSecretKey:     getenv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getenv("STRIPE_WEBHOOK_SECRET", ""),
		DefaultMonthlyQuota: getenvInt("DEFAULT_MONTHLY_QUOTA", 3000),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
