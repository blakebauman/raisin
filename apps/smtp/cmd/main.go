package main

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/joho/godotenv"
	"github.com/raisin-run/raisin/internal/auth"
	"github.com/raisin-run/raisin/internal/config"
	"github.com/raisin-run/raisin/internal/db"
	"github.com/raisin-run/raisin/internal/email"
	"github.com/raisin-run/raisin/internal/jobs"
)

// SMTP relay — AUTH PLAIN with API key as password (username ignored / "raisin").
type backend struct {
	pool   *db.Pool
	emails *email.Service
}

type session struct {
	backend *backend
	team    *auth.Team
	from    string
	to      []string
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{backend: b}, nil
}

func (s *session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		team, _, err := auth.LookupAPIKey(context.Background(), s.backend.pool, password)
		if err != nil {
			return smtp.ErrAuthFailed
		}
		s.team = team
		return nil
	}), nil
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	if s.team == nil {
		return smtp.ErrAuthRequired
	}
	s.from = from
	return nil
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *session) Data(r io.Reader) error {
	if s.team == nil {
		return smtp.ErrAuthRequired
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	subject, html, text := parseSimpleMIME(string(body))
	_, err = s.backend.emails.Send(context.Background(), s.team, email.SendRequest{
		From: s.from, To: s.to, Subject: subject, HTML: html, Text: text,
	}, "")
	return err
}

func (s *session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *session) Logout() error { return nil }

func parseSimpleMIME(raw string) (subject, html, text string) {
	parts := strings.SplitN(raw, "\r\n\r\n", 2)
	headers, body := raw, ""
	if len(parts) == 2 {
		headers, body = parts[0], parts[1]
	} else {
		parts = strings.SplitN(raw, "\n\n", 2)
		if len(parts) == 2 {
			headers, body = parts[0], parts[1]
		}
	}
	for _, line := range strings.Split(headers, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			subject = strings.TrimSpace(line[8:])
		}
	}
	ct := strings.ToLower(headers)
	if strings.Contains(ct, "text/html") {
		html = body
	} else {
		text = body
	}
	return
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	client := jobs.NewClient(cfg)
	defer client.Close()

	be := &backend{pool: pool, emails: &email.Service{Pool: pool, Client: client}}
	s := smtp.NewServer(be)
	s.Addr = cfg.SMTPAddr
	s.Domain = "smtp.raisin.run"
	s.ReadTimeout = 30 * time.Second
	s.WriteTimeout = 30 * time.Second
	s.MaxMessageBytes = 10 << 20
	s.AllowInsecureAuth = true

	log.Printf("smtp listening on %s (AUTH PLAIN, password = API key)", cfg.SMTPAddr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
