package jobs_test

import (
	"encoding/json"
	"testing"

	"github.com/raisin-run/raisin/internal/jobs"
)

func TestNewEmailSendTask(t *testing.T) {
	task, err := jobs.NewEmailSendTask("eid", "tid")
	if err != nil {
		t.Fatal(err)
	}
	if task.Type() != jobs.TypeEmailSend {
		t.Fatalf("type %s", task.Type())
	}
	var p jobs.EmailSendPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		t.Fatal(err)
	}
	if p.EmailID != "eid" || p.TeamID != "tid" {
		t.Fatalf("%+v", p)
	}
}
