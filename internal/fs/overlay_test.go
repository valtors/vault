package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewOverlayCreatesDirs(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-overlay-1")
	defer os.RemoveAll(root)

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	for _, dir := range []string{o.Home(), o.Tmp(), o.Root()} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("missing dir %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a dir", dir)
		}
	}
}

func TestResolveInsideSandbox(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-overlay-2")
	defer os.RemoveAll(root)

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	testFile := filepath.Join(o.Home(), "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	resolved, err := o.Resolve(testFile)
	if err != nil {
		t.Fatalf("Resolve(%s): %v", testFile, err)
	}
	if resolved != testFile {
		t.Fatalf("resolved = %s, want %s", resolved, testFile)
	}
}

func TestResolveBlocksOutsideSandbox(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-overlay-3")
	defer os.RemoveAll(root)

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	_, err = o.Resolve("/etc/passwd")
	if err == nil {
		t.Fatal("Resolve(/etc/passwd) should fail")
	}
}

func TestResolveBlocksSensitivePaths(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-overlay-4")
	defer os.RemoveAll(root)

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	sshPath := filepath.Join(o.Home(), ".ssh", "id_rsa")
	_, err = o.Resolve(sshPath)
	if err == nil {
		t.Fatal("Resolve(.ssh/id_rsa) should fail")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-overlay-5")
	defer os.RemoveAll(root)

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	testPath := filepath.Join(o.Home(), "data", "file.txt")
	if err := o.WriteFile(testPath, []byte("payload"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := o.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("data = %s, want payload", string(data))
	}
}

func TestWriteFileBlocksSensitivePath(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-overlay-6")
	defer os.RemoveAll(root)

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	sshPath := filepath.Join(o.Home(), ".ssh", "id_rsa")
	err = o.WriteFile(sshPath, []byte("stolen"), 0644)
	if err == nil {
		t.Fatal("WriteFile to .ssh should fail")
	}
}

func TestCleanup(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-overlay-7")

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	if err := o.Cleanup(); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("Cleanup should remove root dir")
	}
}

func TestSafeRemoveRoot(t *testing.T) {
	err := SafeRemove("/")
	if err == nil {
		t.Fatal("SafeRemove(/) should fail")
	}
}

func TestSafeRemoveNormal(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "vault-test-safe-rm")
	os.MkdirAll(dir, 0700)
	if err := SafeRemove(dir); err != nil {
		t.Fatalf("SafeRemove(%s): %v", dir, err)
	}
}

func TestIsBlockedPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	blocked := filepath.Join(home, ".ssh")
	if !IsBlockedPath(blocked) {
		t.Fatalf("IsBlockedPath(%s) = false, want true", blocked)
	}

	normal := filepath.Join(home, "projects")
	if IsBlockedPath(normal) {
		t.Fatalf("IsBlockedPath(%s) = true, want false", normal)
	}
}

func TestDiskUsage(t *testing.T) {
	root := filepath.Join(os.TempDir(), "vault-test-disk-usage")
	defer os.RemoveAll(root)

	o, err := NewOverlay(root, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}

	o.WriteFile(filepath.Join(o.Home(), "a.txt"), []byte("aaaaa"), 0644)
	o.WriteFile(filepath.Join(o.Home(), "b.txt"), []byte("bb"), 0644)

	size, err := DiskUsage(o.Home())
	if err != nil {
		t.Fatalf("DiskUsage: %v", err)
	}
	if size < 7 {
		t.Fatalf("disk usage = %d, want >= 7", size)
	}
}
