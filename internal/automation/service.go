package automation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/blakebauman/raisin/internal/apierr"
	"github.com/blakebauman/raisin/internal/db"
	"github.com/blakebauman/raisin/internal/jobs"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	Pool   *db.Pool
	Client *asynq.Client
}

type Automation struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	TriggerType   string          `json:"trigger_type"`
	TriggerFilter json.RawMessage `json:"trigger_filter"`
	Enabled       bool            `json:"enabled"`
	Steps         []Step          `json:"steps,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type Step struct {
	ID       uuid.UUID       `json:"id"`
	Position int             `json:"position"`
	Type     string          `json:"type"`
	Config   json.RawMessage `json:"config"`
}

type CreateRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	TriggerType   string          `json:"trigger_type"`
	TriggerFilter json.RawMessage `json:"trigger_filter"`
	Steps         []StepInput     `json:"steps"`
}

type StepInput struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type Run struct {
	ID          uuid.UUID `json:"id"`
	AutomationID uuid.UUID `json:"automation_id"`
	Status      string    `json:"status"`
	CurrentStep int       `json:"current_step"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) Create(ctx context.Context, teamID uuid.UUID, req CreateRequest) (*Automation, error) {
	if req.Name == "" || req.TriggerType == "" {
		return nil, apierr.Validation("name and trigger_type are required")
	}
	if len(req.TriggerFilter) == 0 {
		req.TriggerFilter = []byte("{}")
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO automations (team_id, name, description, trigger_type, trigger_filter, enabled)
		VALUES ($1,$2,$3,$4,$5,false) RETURNING id
	`, teamID, req.Name, nullStr(req.Description), req.TriggerType, req.TriggerFilter).Scan(&id)
	if err != nil {
		return nil, err
	}
	for i, st := range req.Steps {
		cfg := st.Config
		if len(cfg) == 0 {
			cfg = []byte("{}")
		}
		_, err := s.Pool.Exec(ctx, `
			INSERT INTO automation_steps (automation_id, position, step_type, config)
			VALUES ($1,$2,$3,$4)
		`, id, i, st.Type, cfg)
		if err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Get(ctx context.Context, teamID, id uuid.UUID) (*Automation, error) {
	var a Automation
	var desc *string
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, description, trigger_type, trigger_filter, enabled, created_at
		FROM automations WHERE id = $1 AND team_id = $2
	`, id, teamID).Scan(&a.ID, &a.Name, &desc, &a.TriggerType, &a.TriggerFilter, &a.Enabled, &a.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apierr.NotFound
	}
	if err != nil {
		return nil, err
	}
	a.Description = desc
	rows, err := s.Pool.Query(ctx, `
		SELECT id, position, step_type, config FROM automation_steps
		WHERE automation_id = $1 ORDER BY position
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st Step
		if err := rows.Scan(&st.ID, &st.Position, &st.Type, &st.Config); err != nil {
			return nil, err
		}
		a.Steps = append(a.Steps, st)
	}
	return &a, nil
}

func (s *Service) List(ctx context.Context, teamID uuid.UUID) ([]*Automation, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id FROM automations WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Automation
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		a, err := s.Get(ctx, teamID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (s *Service) SetEnabled(ctx context.Context, teamID, id uuid.UUID, enabled bool) (*Automation, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE automations SET enabled = $3, updated_at = now() WHERE id = $1 AND team_id = $2
	`, id, teamID, enabled)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, apierr.NotFound
	}
	return s.Get(ctx, teamID, id)
}

func (s *Service) Delete(ctx context.Context, teamID, id uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM automations WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apierr.NotFound
	}
	return nil
}

// Trigger starts runs for all enabled automations matching triggerType for the team.
func (s *Service) Trigger(ctx context.Context, teamID uuid.UUID, triggerType string, contactID, emailID, receivedID *uuid.UUID, contextData map[string]any) error {
	rows, err := s.Pool.Query(ctx, `
		SELECT id FROM automations WHERE team_id = $1 AND trigger_type = $2 AND enabled
	`, teamID, triggerType)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	ctxBytes, _ := json.Marshal(contextData)
	if ctxBytes == nil {
		ctxBytes = []byte("{}")
	}
	for _, autoID := range ids {
		var runID uuid.UUID
		err := s.Pool.QueryRow(ctx, `
			INSERT INTO automation_runs (automation_id, team_id, contact_id, email_id, received_email_id, context)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id
		`, autoID, teamID, contactID, emailID, receivedID, ctxBytes).Scan(&runID)
		if err != nil {
			return err
		}
		task, err := jobs.NewAutomationStepTask(runID.String(), teamID.String())
		if err != nil {
			return err
		}
		if _, err := jobs.Enqueue(s.Client, task); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListRuns(ctx context.Context, teamID, automationID uuid.UUID) ([]*Run, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, automation_id, status, current_step, created_at
		FROM automation_runs WHERE team_id = $1 AND automation_id = $2
		ORDER BY created_at DESC LIMIT 50
	`, teamID, automationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Run
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.AutomationID, &r.Status, &r.CurrentStep, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, nil
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
