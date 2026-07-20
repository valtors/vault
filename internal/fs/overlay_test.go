package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAllowedDir(t *testing.T) {
	o := NewOverlay("/tmp/vault-test", []string{"/usr", "/bin"}, []string{"/home"})
	resolved, err := o.Resolve("/bin/sh")
	if err != nil {
		t.Fatalf("allowed dir should resolve: %v", err)
	}
	if resolved != "/bin/sh" {
		t.Fatalf("expected /bin/sh, got %s", resolved)
	}
}

func TestResolveBlockedDir(t *testing.T) {
	o := NewOverlay("/tmp/vault-test", []string{"/usr"}, []string{"/home", "/root"})
	_, err := o.Resolve("/home/user/.ssh/id_rsa")
	if err == nil {
		t.Fatal("blocked dir should be denied")
	}
}

func TestResolveOverlayDir(t *testing.T) {
	o := NewOverlay("/tmp/vault-test", []string{"/usr"}, []string{"/home"})
	resolved, err := o.Resolve("/tmp/work/file.txt")
	if err != nil {
		t.Fatalf("non-blocked, non-allowed should go to overlay: %v", err)
	}
	if resolved != "/tmp/vault-test/tmp/work/file.txt" {
		t.Fatalf("expected overlay path, got %s", resolved)
	}
}

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	o := NewOverlay(dir, nil, []string{"/home"})

	if err := o.WriteFile("/test.txt", []byte("hello")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, err := o.ReadFile("/test.txt")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected hello, got %s", string(data))
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	o := NewOverlay(dir, nil, nil)

	if o.Exists("/nofile.txt") {
		t.Fatal("should not exist")
	}

	o.WriteFile("/exists.txt", []byte("yes"))
	if !o.Exists("/exists.txt") {
		t.Fatal("should exist after write")
	}
}

func TestListDir(t *testing.T) {
	dir := t.TempDir()
	o := NewOverlay(dir, nil, nil)

	o.WriteFile("/a.txt", []byte("a"))
	o.WriteFile("/b.txt", []byte("b"))

	entries, err := o.ListDir("/")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(entries))
	}
}

func TestStat(t *testing.T) {
	dir := t.TempDir()
	o := NewOverlay(dir, nil, nil)

	o.WriteFile("/stat.txt", []byte("content"))
	info, err := o.Stat("/stat.txt")
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size() != 7 {
		t.Fatalf("expected size 7, got %d", info.Size())
	}
}

func TestBlockedPreventsWrite(t *testing.T) {
	dir := t.TempDir()
	o := NewOverlay(dir, nil, []string{"/home/container/.ssh"})

	err := o.WriteFile("/home/container/.ssh/id_rsa", []byte("PRIVATE KEY"))
	if err == nil {
		t.Fatal("write to blocked path should fail")
	}
}

func TestAllowedReadsFromReal(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real.txt")
	os.WriteFile(realFile, []byte("real content"), 0644)

	o := NewOverlay("/tmp/vault-overlay", []string{tmpDir}, nil)
	data, err := o.ReadFile(realFile)
	if err != nil {
		t.Fatalf("read from allowed dir failed: %v", err)
	}
	if string(data) != "real content" {
		t.Fatalf("expected real content, got %s", string(data))
	}
}
