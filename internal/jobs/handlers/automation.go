package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blakebauman/raisin/internal/auth"
	"github.com/blakebauman/raisin/internal/email"
	"github.com/blakebauman/raisin/internal/jobs"
	tmpl "github.com/blakebauman/raisin/internal/template"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func (h *Handlers) HandleAutomationStep(ctx context.Context, t *asynq.Task) error {
	var p jobs.AutomationStepPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	runID, err := uuid.Parse(p.RunID)
	if err != nil {
		return err
	}
	teamID, err := uuid.Parse(p.TeamID)
	if err != nil {
		return err
	}

	var automationID uuid.UUID
	var status string
	var currentStep int
	var contactID, emailID *uuid.UUID
	var runCtx []byte
	err = h.Pool.QueryRow(ctx, `
		SELECT automation_id, status, current_step, contact_id, email_id, context
		FROM automation_runs WHERE id = $1 AND team_id = $2
	`, runID, teamID).Scan(&automationID, &status, &currentStep, &contactID, &emailID, &runCtx)
	if err != nil {
		return err
	}
	if status != "running" {
		return nil
	}

	var stepID uuid.UUID
	var stepType string
	var configRaw []byte
	err = h.Pool.QueryRow(ctx, `
		SELECT id, step_type, config FROM automation_steps
		WHERE automation_id = $1 AND position = $2
	`, automationID, currentStep).Scan(&stepID, &stepType, &configRaw)
	if err != nil {
		// no more steps
		_, _ = h.Pool.Exec(ctx, `
			UPDATE automation_runs SET status = 'completed', updated_at = now() WHERE id = $1
		`, runID)
		return nil
	}

	_, _ = h.Pool.Exec(ctx, `
		INSERT INTO automation_run_steps (run_id, step_id, status, started_at)
		VALUES ($1,$2,'running', now())
		ON CONFLICT (run_id, step_id) DO UPDATE SET status = 'running', started_at = now()
	`, runID, stepID)

	var cfg map[string]any
	_ = json.Unmarshal(configRaw, &cfg)

	switch stepType {
	case "wait":
		secs := int(num(cfg["seconds"], 60))
		next := currentStep + 1
		_, _ = h.Pool.Exec(ctx, `
			UPDATE automation_runs SET current_step = $2, updated_at = now() WHERE id = $1
		`, runID, next)
		_, _ = h.Pool.Exec(ctx, `
			UPDATE automation_run_steps SET status = 'completed', finished_at = now() WHERE run_id = $1 AND step_id = $2
		`, runID, stepID)
		task, err := jobs.NewAutomationStepTaskAt(runID.String(), teamID.String(), time.Now().Add(time.Duration(secs)*time.Second))
		if err != nil {
			return err
		}
		_, err = jobs.Enqueue(h.Asynq, task)
		return err

	case "send_email":
		from, _ := cfg["from"].(string)
		subject, _ := cfg["subject"].(string)
		html, _ := cfg["html"].(string)
		to, _ := cfg["to"].(string)
		if to == "" && contactID != nil {
			_ = h.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1`, *contactID).Scan(&to)
		}
		if from == "" || to == "" || subject == "" {
			_, _ = h.Pool.Exec(ctx, `
				UPDATE automation_run_steps SET status = 'failed', error = $3, finished_at = now()
				WHERE run_id = $1 AND step_id = $2
			`, runID, stepID, "missing from/to/subject")
			_, _ = h.Pool.Exec(ctx, `UPDATE automation_runs SET status = 'failed', updated_at = now() WHERE id = $1`, runID)
			return fmt.Errorf("automation send_email missing fields")
		}
		tags := map[string]string{}
		if contactID != nil {
			tags = h.mergeContactTags(ctx, teamID, to, tags)
		}
		subject = tmpl.Render(subject, tags)
		html = tmpl.Render(html, tags)
		if h.Emails != nil {
			team, err := auth.LoadTeam(ctx, h.Pool, teamID)
			if err != nil {
				return err
			}
			_, err = h.Emails.Send(ctx, team, email.SendRequest{
				From: from, To: []string{to}, Subject: subject, HTML: html, Tags: tags,
			}, "")
			if err != nil {
				_, _ = h.Pool.Exec(ctx, `
					UPDATE automation_run_steps SET status = 'failed', error = $3, finished_at = now()
					WHERE run_id = $1 AND step_id = $2
				`, runID, stepID, err.Error())
				return err
			}
		}
		_, _ = h.Pool.Exec(ctx, `
			UPDATE automation_run_steps SET status = 'completed', finished_at = now() WHERE run_id = $1 AND step_id = $2
		`, runID, stepID)

	case "add_to_segment":
		segStr, _ := cfg["segment_id"].(string)
		segID, err := uuid.Parse(segStr)
		if err != nil || contactID == nil {
			_, _ = h.Pool.Exec(ctx, `
				UPDATE automation_run_steps SET status = 'skipped', finished_at = now() WHERE run_id = $1 AND step_id = $2
			`, runID, stepID)
		} else {
			_, _ = h.Pool.Exec(ctx, `
				INSERT INTO segment_members (segment_id, contact_id) VALUES ($1,$2) ON CONFLICT DO NOTHING
			`, segID, *contactID)
			_, _ = h.Pool.Exec(ctx, `
				UPDATE automation_run_steps SET status = 'completed', finished_at = now() WHERE run_id = $1 AND step_id = $2
			`, runID, stepID)
		}

	case "condition":
		ok := true
		propKey, _ := cfg["property"].(string)
		want, _ := cfg["equals"].(string)
		if propKey != "" && contactID != nil {
			var props []byte
			_ = h.Pool.QueryRow(ctx, `SELECT properties FROM contacts WHERE id = $1`, *contactID).Scan(&props)
			var m map[string]any
			_ = json.Unmarshal(props, &m)
			got, _ := m[propKey].(string)
			if got == "" {
				if v, exists := m[propKey]; exists {
					got = fmt.Sprint(v)
				}
			}
			ok = got == want
		}
		if !ok {
			_, _ = h.Pool.Exec(ctx, `
				UPDATE automation_run_steps SET status = 'skipped', finished_at = now() WHERE run_id = $1 AND step_id = $2
			`, runID, stepID)
			_, _ = h.Pool.Exec(ctx, `
				UPDATE automation_runs SET status = 'completed', updated_at = now() WHERE id = $1
			`, runID)
			return nil
		}
		_, _ = h.Pool.Exec(ctx, `
			UPDATE automation_run_steps SET status = 'completed', finished_at = now() WHERE run_id = $1 AND step_id = $2
		`, runID, stepID)

	default:
		_, _ = h.Pool.Exec(ctx, `
			UPDATE automation_run_steps SET status = 'skipped', finished_at = now() WHERE run_id = $1 AND step_id = $2
		`, runID, stepID)
	}

	next := currentStep + 1
	_, _ = h.Pool.Exec(ctx, `
		UPDATE automation_runs SET current_step = $2, updated_at = now() WHERE id = $1
	`, runID, next)
	task, err := jobs.NewAutomationStepTask(runID.String(), teamID.String())
	if err != nil {
		return err
	}
	_, err = jobs.Enqueue(h.Asynq, task)
	return err
}

func num(v any, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return def
	}
}
