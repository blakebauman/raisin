package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/events"
	"github.com/google/uuid"
)

const maxEventStreamsPerTeam = 10

// streamEmailEvents serves GET /events/stream as Server-Sent Events.
func (s *Server) streamEmailEvents(w http.ResponseWriter, r *http.Request) {
	team := teamOrWrite(w, r)
	if team == nil {
		return
	}
	if s.Redis == nil {
		apierr.Write(w, apierr.New(503, "unavailable", "event stream requires Redis"))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		apierr.Write(w, apierr.New(500, "internal_error", "streaming unsupported"))
		return
	}

	typeSet := parseStreamTypes(r.URL.Query().Get("types"))
	var emailFilter *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("email_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			apierr.Write(w, apierr.Validation("invalid email_id"))
			return
		}
		emailFilter = &id
	}

	streamKey := "raisin:events:streams:" + team.ID.String()
	n, err := s.Redis.Incr(r.Context(), streamKey).Result()
	if err != nil {
		apierr.Write(w, apierr.New(503, "unavailable", "could not reserve stream slot"))
		return
	}
	if n == 1 {
		_ = s.Redis.Expire(r.Context(), streamKey, 24*time.Hour).Err()
	}
	if n > int64(maxEventStreamsPerTeam) {
		_, _ = s.Redis.Decr(r.Context(), streamKey).Result()
		apierr.Write(w, apierr.New(429, "too_many_streams", "too many concurrent event streams for this team"))
		return
	}
	defer func() {
		_, _ = s.Redis.Decr(r.Context(), streamKey).Result()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	writeSSE := func(ev events.LiveEvent) bool {
		if !events.MatchStreamFilters(ev, typeSet, emailFilter) {
			return true
		}
		body, err := json.Marshal(ev)
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", ev.ID.String(), ev.Type, body); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// Last-Event-ID replay from Postgres, then live Redis.
	if last := strings.TrimSpace(r.Header.Get("Last-Event-ID")); last != "" {
		lastID, err := uuid.Parse(last)
		if err == nil && s.Events != nil {
			missed, err := s.Events.ReplaySince(r.Context(), team.ID, lastID)
			if err == nil {
				for _, ev := range missed {
					if r.Context().Err() != nil {
						return
					}
					if !writeSSE(ev) {
						return
					}
				}
			}
		}
	}

	pubsub := s.Redis.Subscribe(r.Context(), events.LiveChannel(team.ID))
	defer pubsub.Close()
	ch := pubsub.Channel()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var ev events.LiveEvent
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
				continue
			}
			if !writeSSE(ev) {
				return
			}
		}
	}
}

func parseStreamTypes(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		for _, t := range events.DefaultStreamTypes {
			out[t] = struct{}{}
		}
		return out
	}
	for _, p := range strings.Split(raw, ",") {
		t := strings.TrimSpace(p)
		if t != "" {
			out[t] = struct{}{}
		}
	}
	return out
}
