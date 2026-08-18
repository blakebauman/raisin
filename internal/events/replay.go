package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ReplaySince returns email_events after lastID for the team (within 1h), ordered chronologically.
// Used by SSE Last-Event-ID reconnect. Types filter is applied in-process by the stream handler.
func (p *Processor) ReplaySince(ctx context.Context, teamID, lastID uuid.UUID) ([]LiveEvent, error) {
	if p == nil || p.Pool == nil {
		return nil, nil
	}
	var lastCreated time.Time
	err := p.Pool.QueryRow(ctx, `
		SELECT created_at FROM email_events WHERE id = $1 AND team_id = $2
	`, lastID, teamID).Scan(&lastCreated)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := p.Pool.Query(ctx, `
		SELECT id, type, data, created_at
		FROM email_events
		WHERE team_id = $1
		  AND created_at > now() - interval '1 hour'
		  AND (created_at, id) > ($2::timestamptz, $3::uuid)
		ORDER BY created_at ASC, id ASC
		LIMIT 500
	`, teamID, lastCreated, lastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LiveEvent
	for rows.Next() {
		var e LiveEvent
		var data []byte
		if err := rows.Scan(&e.ID, &e.Type, &data, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Data = json.RawMessage(data)
		out = append(out, e)
	}
	return out, rows.Err()
}

// MatchStreamFilters returns true if the event matches type allow-list and optional email_id.
func MatchStreamFilters(ev LiveEvent, types map[string]struct{}, emailID *uuid.UUID) bool {
	if len(types) > 0 {
		if _, ok := types[ev.Type]; !ok {
			return false
		}
	}
	if emailID == nil {
		return true
	}
	var meta struct {
		EmailID string `json:"email_id"`
	}
	if err := json.Unmarshal(ev.Data, &meta); err != nil {
		return false
	}
	id, err := uuid.Parse(meta.EmailID)
	if err != nil {
		return false
	}
	return id == *emailID
}
