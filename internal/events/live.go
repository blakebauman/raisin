package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const liveChannelPrefix = "raisin:events:"

// DefaultStreamTypes are delivered on /events/stream when types= is omitted.
var DefaultStreamTypes = []string{
	"email.sent",
	"email.delivered",
	"email.bounced",
	"email.complained",
	"email.failed",
	"email.delivery_delayed",
	"email.opened",
	"email.clicked",
}

// LiveEvent is the SSE / Redis pub/sub payload (same shape as webhook envelope + id).
type LiveEvent struct {
	ID        uuid.UUID       `json:"id"`
	Type      string          `json:"type"`
	CreatedAt time.Time       `json:"created_at"`
	Data      json.RawMessage `json:"data"`
}

// LiveChannel returns the Redis pub/sub channel for a team's email events.
func LiveChannel(teamID uuid.UUID) string {
	return liveChannelPrefix + teamID.String()
}

func (p *Processor) publishLive(ctx context.Context, teamID uuid.UUID, ev LiveEvent) {
	if p == nil || p.Redis == nil {
		return
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = p.Redis.Publish(ctx, LiveChannel(teamID), b).Err()
}
