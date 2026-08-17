package jobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/blakebauman/raisin/internal/config"
)

const (
	TypeEmailSend         = "email:send"
	TypeWebhookDeliver    = "webhook:deliver"
	TypeBroadcastSend     = "broadcast:send"
	TypeDomainVerify      = "domain:verify"
	TypeAutomationStep    = "automation:step"
	TypeIPWarmupTick      = "ip:warmup-tick"
	QueueCritical         = "critical"
	QueueDefault          = "default"
	QueueLow              = "low"
)

type EmailSendPayload struct {
	EmailID string `json:"email_id"`
	TeamID  string `json:"team_id"`
}

type WebhookDeliverPayload struct {
	WebhookEventID string `json:"webhook_event_id"`
}

type BroadcastSendPayload struct {
	BroadcastID string `json:"broadcast_id"`
	TeamID      string `json:"team_id"`
}

type DomainVerifyPayload struct {
	DomainID string `json:"domain_id"`
	TeamID   string `json:"team_id"`
}

type AutomationStepPayload struct {
	RunID  string `json:"run_id"`
	TeamID string `json:"team_id"`
}

type IPWarmupTickPayload struct {
	PoolID string `json:"pool_id"`
}

func NewClient(cfg config.Config) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr(cfg.RedisURL)})
}

func NewServer(cfg config.Config) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr(cfg.RedisURL)},
		asynq.Config{
			Concurrency: 20,
			Queues: map[string]int{
				QueueCritical: 6,
				QueueDefault:  3,
				QueueLow:      1,
			},
			RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
				return time.Duration(n*n) * time.Second
			},
		},
	)
}

func NewInspector(cfg config.Config) *asynq.Inspector {
	return asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr(cfg.RedisURL)})
}

func NewEmailSendTask(emailID, teamID string) (*asynq.Task, error) {
	b, err := json.Marshal(EmailSendPayload{EmailID: emailID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEmailSend, b, asynq.Queue(QueueCritical), asynq.MaxRetry(5)), nil
}

func NewScheduledEmailSendTask(emailID, teamID string, at time.Time) (*asynq.Task, error) {
	b, err := json.Marshal(EmailSendPayload{EmailID: emailID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEmailSend, b,
		asynq.Queue(QueueCritical),
		asynq.ProcessAt(at),
		asynq.MaxRetry(5),
	), nil
}

func NewWebhookDeliverTask(eventID string) (*asynq.Task, error) {
	b, err := json.Marshal(WebhookDeliverPayload{WebhookEventID: eventID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeWebhookDeliver, b, asynq.Queue(QueueDefault), asynq.MaxRetry(8)), nil
}

func NewBroadcastSendTask(broadcastID, teamID string) (*asynq.Task, error) {
	b, err := json.Marshal(BroadcastSendPayload{BroadcastID: broadcastID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeBroadcastSend, b, asynq.Queue(QueueLow), asynq.MaxRetry(3)), nil
}

func NewDomainVerifyTask(domainID, teamID string) (*asynq.Task, error) {
	b, err := json.Marshal(DomainVerifyPayload{DomainID: domainID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeDomainVerify, b, asynq.Queue(QueueDefault), asynq.MaxRetry(3)), nil
}

func NewAutomationStepTask(runID, teamID string) (*asynq.Task, error) {
	b, err := json.Marshal(AutomationStepPayload{RunID: runID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeAutomationStep, b, asynq.Queue(QueueDefault), asynq.MaxRetry(5)), nil
}

func NewAutomationStepTaskAt(runID, teamID string, at time.Time) (*asynq.Task, error) {
	b, err := json.Marshal(AutomationStepPayload{RunID: runID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeAutomationStep, b,
		asynq.Queue(QueueDefault),
		asynq.ProcessAt(at),
		asynq.MaxRetry(5),
	), nil
}

// NewIPWarmupTickTask enqueues a global warmup day rollover (all due pools).
func NewIPWarmupTickTask() (*asynq.Task, error) {
	b, err := json.Marshal(IPWarmupTickPayload{})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeIPWarmupTick, b, asynq.Queue(QueueLow), asynq.MaxRetry(3)), nil
}

func NewScheduler(cfg config.Config) *asynq.Scheduler {
	return asynq.NewScheduler(asynq.RedisClientOpt{Addr: redisAddr(cfg.RedisURL)}, &asynq.SchedulerOpts{
		Location: time.UTC,
	})
}

func redisAddr(redisURL string) string {
	// Accept redis://host:port or host:port
	u := redisURL
	if len(u) > 8 && u[:8] == "redis://" {
		u = u[8:]
	}
	if u == "" {
		return "localhost:6379"
	}
	// strip path/db if present
	for i, c := range u {
		if c == '/' {
			return u[:i]
		}
	}
	return u
}

func Enqueue(client *asynq.Client, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	info, err := client.Enqueue(task, opts...)
	if err != nil {
		return nil, fmt.Errorf("enqueue %s: %w", task.Type(), err)
	}
	return info, nil
}
