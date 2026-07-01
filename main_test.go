package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveNewFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	ok, err := save(p, []byte("services: {}\n"), func() bool { return false })
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "services: {}\n" {
		t.Fatalf("content=%q", b)
	}
}

func TestSaveExistingDeclined(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	os.WriteFile(p, []byte("old"), 0o644)
	ok, err := save(p, []byte("new"), func() bool { return false })
	if err != nil || ok {
		t.Fatalf("expected declined: ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "old" {
		t.Fatalf("should not overwrite, got %q", b)
	}
}

func TestSaveExistingConfirmed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	os.WriteFile(p, []byte("old"), 0o644)
	ok, err := save(p, []byte("new"), func() bool { return true })
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "new" {
		t.Fatalf("got %q", b)
	}
}
