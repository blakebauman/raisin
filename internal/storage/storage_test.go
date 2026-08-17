package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/raisin-run/raisin/internal/storage"
)

func TestLocalPutGet(t *testing.T) {
	dir := t.TempDir()
	s := &storage.Local{Root: dir}
	ctx := context.Background()
	key := "teams/t1/emails/e1/note.txt"
	if err := s.Put(ctx, key, []byte("hello"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	body, ct, err := s.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" || ct != "text/plain" {
		t.Fatalf("%q %q", body, ct)
	}
	if _, err := os.Stat(filepath.Join(dir, "teams/t1/emails/e1/note.txt")); err != nil {
		t.Fatal(err)
	}
}
