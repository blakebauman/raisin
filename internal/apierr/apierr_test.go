package apierr_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/raisin-run/raisin/internal/apierr"
)

func TestWrite(t *testing.T) {
	rr := httptest.NewRecorder()
	apierr.Write(rr, apierr.Validation("bad"))
	if rr.Code != 400 {
		t.Fatalf("status %d", rr.Code)
	}
	var e apierr.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &e)
	if e.Name != "validation_error" {
		t.Fatalf("name %s", e.Name)
	}
}
