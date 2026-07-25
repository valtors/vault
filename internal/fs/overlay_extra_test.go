package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListAllowed(t *testing.T) {
	dir := t.TempDir()
	allowed := []string{dir + "/readable"}
	os.MkdirAll(allowed[0], 0755)
	o, err := NewOverlay(dir, allowed)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}
	defer o.Cleanup()
	list := o.ListAllowed()
	if len(list) != 1 {
		t.Errorf("expected 1, got %d", len(list))
	}
}

func TestIsBlockedPath_Sensitive(t *testing.T) {
	home, _ := os.UserHomeDir()
	if !IsBlockedPath(filepath.Join(home, ".ssh")) {
		t.Error("expected .ssh to be blocked")
	}
	if !IsBlockedPath(filepath.Join(home, ".ssh", "id_rsa")) {
		t.Error("expected .ssh/id_rsa to be blocked")
	}
	if !IsBlockedPath(filepath.Join(home, ".aws")) {
		t.Error("expected .aws to be blocked")
	}
}

func TestIsBlockedPath_Sensitive_Safe(t *testing.T) {
	if IsBlockedPath("/tmp") {
		t.Error("expected /tmp to be safe")
	}
	if IsBlockedPath("/usr/bin") {
		t.Error("expected /usr/bin to be safe")
	}
}

func TestIsReadable(t *testing.T) {
	dir := t.TempDir()
	if !isReadable(dir) {
		t.Error("expected readable dir")
	}
	if isReadable("/nonexistent/path/12345") {
		t.Error("expected false for nonexistent path")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/test", "home_user_test"},
		{"hello world", "hello_world"},
		{"C:\\Users\\test", "C__Users_test"},
		{"a:b", "a_b"},
		{"/leading/slash", "leading_slash"},
	}
	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCreateTempFile(t *testing.T) {
	dir := t.TempDir()
	f, err := CreateTempFile(dir, "test_*")
	if err != nil {
		t.Fatalf("CreateTempFile: %v", err)
	}
	defer f.Close()
	if f.Name() == "" {
		t.Error("expected non-empty name")
	}
}

func TestNewOverlay_NoAllowed(t *testing.T) {
	dir := t.TempDir()
	o, err := NewOverlay(dir, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}
	defer o.Cleanup()
	if o == nil {
		t.Fatal("expected non-nil overlay")
	}
	if len(o.ListAllowed()) != 0 {
		t.Error("expected empty allowed list")
	}
}

func TestOverlay_Home_Tmp_Root(t *testing.T) {
	dir := t.TempDir()
	o, err := NewOverlay(dir, nil)
	if err != nil {
		t.Fatalf("NewOverlay: %v", err)
	}
	defer o.Cleanup()
	if o.Home() == "" {
		t.Error("expected non-empty home")
	}
	if o.Tmp() == "" {
		t.Error("expected non-empty tmp")
	}
	if o.Root() == "" {
		t.Error("expected non-empty root")
	}
}
