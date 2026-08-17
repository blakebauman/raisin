package metrics

import (
	"context"

	"github.com/google/uuid"
	"github.com/blakebauman/raisin/internal/db"
)

type Service struct {
	Pool *db.Pool
}

type EmailMetrics struct {
	Sent       int64 `json:"sent"`
	Delivered  int64 `json:"delivered"`
	Bounced    int64 `json:"bounced"`
	Complained int64 `json:"complained"`
	Opened     int64 `json:"opened"`
	Clicked    int64 `json:"clicked"`
	Queued     int64 `json:"queued"`
	Failed     int64 `json:"failed"`
}

func (s *Service) TeamEmailMetrics(ctx context.Context, teamID uuid.UUID) (*EmailMetrics, error) {
	m := &EmailMetrics{}
	rows, err := s.Pool.Query(ctx, `
		SELECT status, COUNT(*) FROM emails WHERE team_id = $1 GROUP BY status
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		switch status {
		case "queued", "scheduled":
			m.Queued += n
		case "sent":
			m.Sent += n
		case "delivered":
			m.Delivered += n
			m.Sent += n // delivered implies it was sent
		case "bounced":
			m.Bounced += n
			m.Sent += n
		case "complained":
			m.Complained += n
			m.Sent += n
		case "failed", "canceled", "suppressed":
			m.Failed += n
		}
	}
	// Unique emails with at least one open/click event
	_ = s.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT email_id) FROM email_events
		WHERE team_id = $1 AND type = 'email.opened'
	`, teamID).Scan(&m.Opened)
	_ = s.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT email_id) FROM email_events
		WHERE team_id = $1 AND type = 'email.clicked'
	`, teamID).Scan(&m.Clicked)
	return m, nil
}
