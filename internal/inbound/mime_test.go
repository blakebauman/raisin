package inbound

import (
	"strings"
	"testing"
)

func TestParseMIMEMessageWithAttachment(t *testing.T) {
	raw := strings.ReplaceAll(`From: a@b.c
To: inbox@acme.test
Subject: Hi
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="bound"

--bound
Content-Type: text/plain; charset=utf-8

hello body
--bound
Content-Type: text/plain; name="note.txt"
Content-Disposition: attachment; filename="note.txt"
Content-Transfer-Encoding: base64

aGVsbG8=
--bound--
`, "\n", "\r\n")

	html, text, atts := parseMIMEMessage([]byte(raw))
	if text == "" && html == "" {
		t.Fatal("expected body text")
	}
	if !strings.Contains(text, "hello body") && html == "" {
		t.Fatalf("text=%q html=%q", text, html)
	}
	if len(atts) != 1 {
		t.Fatalf("atts=%d", len(atts))
	}
	if atts[0].Filename != "note.txt" {
		t.Fatalf("filename=%s", atts[0].Filename)
	}
	if string(atts[0].Content) != "hello" {
		t.Fatalf("content=%q", atts[0].Content)
	}
}
