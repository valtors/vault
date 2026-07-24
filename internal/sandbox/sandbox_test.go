package sandbox

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.MaxMemoryMB != 512 {
		t.Errorf("expected MaxMemoryMB 512, got %d", c.MaxMemoryMB)
	}
	if c.MaxCPUSeconds != 300 {
		t.Errorf("expected MaxCPUSeconds 300, got %d", c.MaxCPUSeconds)
	}
	if c.TimeoutSecs != 0 {
		t.Errorf("expected TimeoutSecs 0, got %d", c.TimeoutSecs)
	}
}

func TestNew_CreatesSandbox(t *testing.T) {
	cfg := Config{RootDir: t.TempDir()}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if s.ID() == 0 {
		t.Error("expected non-zero ID")
	}
	if s.Root() == "" {
		t.Error("expected non-empty Root")
	}
	if s.Home() == "" {
		t.Error("expected non-empty Home")
	}
	if s.DB() == nil {
		t.Error("expected non-nil DB")
	}
	if s.Overlay() == nil {
		t.Error("expected non-nil Overlay")
	}
	if s.IsDone() {
		t.Error("expected not done")
	}
}

func TestNew_AutoCreatesRootDir(t *testing.T) {
	cfg := Config{RootDir: ""}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if s.Root() == "" {
		t.Error("expected auto-generated Root")
	}
}

func TestStart_NoCommand(t *testing.T) {
	cfg := Config{RootDir: t.TempDir()}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if err := s.Start(); err == nil {
		t.Error("expected error when no command")
	}
}

func TestStart_AlreadyRunning(t *testing.T) {
	cfg := Config{RootDir: t.TempDir(), Command: "sleep", Args: []string{"10"}}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Kill()
	if err := s.Start(); err == nil {
		t.Error("expected error when already running")
	}
}

func TestWait_NotRunning(t *testing.T) {
	cfg := Config{RootDir: t.TempDir()}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if err := s.Wait(); err == nil {
		t.Error("expected error when not running")
	}
}

func TestKill_NotRunning(t *testing.T) {
	cfg := Config{RootDir: t.TempDir()}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if err := s.Kill(); err == nil {
		t.Error("expected error when not running")
	}
}

func TestStartWaitExit_SimpleCommand(t *testing.T) {
	cfg := Config{RootDir: t.TempDir(), Command: "true"}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}
	if !s.IsDone() {
		t.Error("expected done after Wait")
	}
}

func TestEnvSummary(t *testing.T) {
	cfg := Config{RootDir: t.TempDir()}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	env := s.EnvSummary()
	if len(env) == 0 {
		t.Error("expected non-empty env summary")
	}
}

func TestAuditLog_AfterStart(t *testing.T) {
	cfg := Config{RootDir: t.TempDir(), Command: "true"}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Cleanup()
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	s.Wait()
	entries, err := s.AuditLog("sandbox", 100)
	if err != nil {
		t.Fatalf("AuditLog: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected audit log entries after Start")
	}
}
