package tracking_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/raisin-run/raisin/internal/tracking"
)

func TestInject(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	html := `<html><body><a href="https://raisin.run">x</a></body></html>`
	out := tracking.Inject(html, id, "http://localhost:18081", true, true)
	if !strings.Contains(out, "/t/c/"+id.String()) {
		t.Fatalf("missing click tracker: %s", out)
	}
	if !strings.Contains(out, "/t/o/"+id.String()) {
		t.Fatalf("missing open pixel: %s", out)
	}
	if !strings.Contains(out, "mailto:") && strings.Contains(html, "mailto:") {
		t.Fatal("mailto rewritten")
	}
}
