package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/blakebauman/raisin/internal/config"
)

type Store interface {
	Put(ctx context.Context, key string, body []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, string, error)
	// GetFrom reads from an explicit bucket (SES inbound). Empty bucket uses the default.
	GetFrom(ctx context.Context, bucket, key string) ([]byte, string, error)
	Delete(ctx context.Context, key string) error
}

func New(cfg config.Config) (Store, error) {
	driver := strings.ToLower(os.Getenv("STORAGE_DRIVER"))
	if driver == "" {
		if cfg.SenderDriver == "mailpit" {
			driver = "local"
		} else {
			driver = "s3"
		}
	}
	if driver == "local" {
		dir := getenv("LOCAL_STORAGE_DIR", "./.data/attachments")
		_ = os.MkdirAll(dir, 0o755)
		return &Local{Root: dir}, nil
	}
	return NewS3(cfg)
}

type Local struct {
	Root string
}

func (l *Local) path(key string) string {
	clean := filepath.Clean("/" + key)
	return filepath.Join(l.Root, clean)
}

func (l *Local) Put(ctx context.Context, key string, body []byte, contentType string) error {
	p := l.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	meta := p + ".ctype"
	_ = os.WriteFile(meta, []byte(contentType), 0o644)
	return os.WriteFile(p, body, 0o644)
}

func (l *Local) Get(ctx context.Context, key string) ([]byte, string, error) {
	return l.GetFrom(ctx, "", key)
}

func (l *Local) GetFrom(ctx context.Context, bucket, key string) ([]byte, string, error) {
	_ = bucket
	p := l.path(key)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, "", err
	}
	ct, _ := os.ReadFile(p + ".ctype")
	if len(ct) == 0 {
		ct = []byte("application/octet-stream")
	}
	return b, string(ct), nil
}

func (l *Local) Delete(ctx context.Context, key string) error {
	p := l.path(key)
	_ = os.Remove(p + ".ctype")
	return os.Remove(p)
}

type S3 struct {
	client        *s3.Client
	bucket        string
	inboundBucket string
}

func NewS3(cfg config.Config) (*S3, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}
	var opts []func(*s3.Options)
	if cfg.AWSEndpointURL != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.AWSEndpointURL)
			o.UsePathStyle = true
		})
	}
	inbound := cfg.S3InboundBucket
	if inbound == "" {
		inbound = cfg.S3Bucket
	}
	return &S3{
		client:        s3.NewFromConfig(awsCfg, opts...),
		bucket:        cfg.S3Bucket,
		inboundBucket: inbound,
	}, nil
}

func (s *S3) Put(ctx context.Context, key string, body []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	return err
}

func (s *S3) Get(ctx context.Context, key string) ([]byte, string, error) {
	return s.GetFrom(ctx, s.bucket, key)
}

func (s *S3) GetFrom(ctx context.Context, bucket, key string) ([]byte, string, error) {
	bkt := bucket
	if bkt == "" {
		bkt = s.inboundBucket
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	defer out.Body.Close()
	b, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return b, ct, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func KeyForEmail(teamID, emailID, filename string) string {
	safe := strings.ReplaceAll(filepath.Base(filename), "..", "")
	return fmt.Sprintf("teams/%s/emails/%s/%s", teamID, emailID, safe)
}

func KeyForInbound(teamID, messageID string) string {
	return fmt.Sprintf("teams/%s/inbound/%s.eml", teamID, messageID)
}

func KeyForInboundAttachment(teamID, receivedID, filename string) string {
	safe := strings.ReplaceAll(filepath.Base(filename), "..", "")
	return fmt.Sprintf("teams/%s/inbound/%s/%s", teamID, receivedID, safe)
}
