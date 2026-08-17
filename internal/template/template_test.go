package template_test

import (
	"testing"

	"github.com/raisin-run/raisin/internal/template"
)

func TestRender(t *testing.T) {
	out := template.Render("Hi {{name}}, welcome to {{ product }}", map[string]string{
		"name":    "Blake",
		"product": "Raisin",
	})
	if out != "Hi Blake, welcome to Raisin" {
		t.Fatalf("got %q", out)
	}
}
