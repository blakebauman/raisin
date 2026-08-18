package tracking_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/blakebauman/raisin/internal/tracking"
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

func TestInjectSkipsUnsubscribe(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	html := `<html><body><a href="http://localhost:18081/unsubscribe/` + id.String() + `?token=abc">Unsubscribe</a></body></html>`
	out := tracking.Inject(html, id, "http://localhost:18081", true, true)
	if strings.Contains(out, "/t/c/") {
		t.Fatalf("unsub link was click-tracked: %s", out)
	}
}

func TestAppendUnsubscribe(t *testing.T) {
	html, text := tracking.AppendUnsubscribe("<html><body><p>hi</p></body></html>", "hi", "http://x/unsubscribe/1?token=t")
	if !strings.Contains(html, "Unsubscribe</a>") || !strings.Contains(html, "token=t") {
		t.Fatalf("html footer: %s", html)
	}
	if !strings.Contains(text, "Unsubscribe: http://x/unsubscribe/1?token=t") {
		t.Fatalf("text footer: %s", text)
	}
	// idempotent
	html2, _ := tracking.AppendUnsubscribe(html, text, "http://x/unsubscribe/1?token=t")
	if strings.Count(strings.ToLower(html2), "/unsubscribe/") != 1 {
		t.Fatalf("duplicated footer: %s", html2)
	}
}
