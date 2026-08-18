package raisin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LiveEmailEvent is one SSE payload from GET /events/stream.
type LiveEmailEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	CreatedAt time.Time       `json:"created_at"`
	Data      json.RawMessage `json:"data"`
}

// StreamOpts configures EventsService.Stream.
type StreamOpts struct {
	Types       []string
	EmailID     string
	LastEventID string // Last-Event-ID for reconnect replay (within 1h)
}

type EventsService struct{ c *Client }

// Stream opens GET /events/stream and returns channels for events and terminal errors.
// Cancel ctx to disconnect. Use a Client with HTTPClient.Timeout == 0 (or very large)
// so the connection is not killed mid-stream; this method uses a no-timeout client clone.
// At-least-once: pass LastEventID on reconnect. Prefer webhooks for durable delivery.
func (s *EventsService) Stream(ctx context.Context, opts *StreamOpts) (<-chan LiveEmailEvent, <-chan error, error) {
	q := url.Values{}
	if opts != nil {
		if len(opts.Types) > 0 {
			q.Set("types", strings.Join(opts.Types, ","))
		}
		if opts.EmailID != "" {
			q.Set("email_id", opts.EmailID)
		}
	}
	path := "/events/stream"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.c.BaseURL+path, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.c.APIKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "raisin-go/0.1.0")
	if opts != nil && opts.LastEventID != "" {
		req.Header.Set("Last-Event-ID", opts.LastEventID)
	}

	client := &http.Client{Timeout: 0}
	if s.c.HTTPClient != nil && s.c.HTTPClient.Transport != nil {
		client.Transport = s.c.HTTPClient.Transport
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode >= 300 {
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		var ae APIError
		_ = json.Unmarshal(data, &ae)
		if ae.Message == "" {
			ae.Message = string(data)
			ae.StatusCode = res.StatusCode
		}
		return nil, nil, &ae
	}

	out := make(chan LiveEmailEvent, 16)
	errs := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errs)
		defer res.Body.Close()
		if err := readSSE(res.Body, out); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()
	return out, errs, nil
}

func readSSE(r io.Reader, out chan<- LiveEmailEvent) error {
	sc := bufio.NewScanner(r)
	// SSE frames can be larger than default 64k
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	var dataLines []string
	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		raw := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		var ev LiveEmailEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			return nil // skip malformed
		}
		out <- ev
		return nil
	}

	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	_ = flush()
	if err := sc.Err(); err != nil {
		return fmt.Errorf("sse read: %w", err)
	}
	return nil
}
