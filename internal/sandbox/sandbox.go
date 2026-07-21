package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valtors/vault/internal/env"
	"github.com/valtors/vault/internal/fs"
	"github.com/valtors/vault/internal/store"
)

type Sandbox struct {
	cfg     Config
	overlay *fs.Overlay
	db      *store.DB
	cmd     *exec.Cmd
	id      int64
	started time.Time
	mu      sync.Mutex
	done    atomic.Bool
}

var nextID int64

func New(cfg Config) (*Sandbox, error) {
	id := atomic.AddInt64(&nextID, 1)

	if cfg.RootDir == "" {
		cfg.RootDir = filepath.Join(os.TempDir(), fmt.Sprintf("vault-sandbox-%d", id))
	}

	dbPath := filepath.Join(cfg.RootDir, "audit.db")
	if err := os.MkdirAll(cfg.RootDir, 0700); err != nil {
		return nil, fmt.Errorf("create root: %w", err)
	}

	db, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init audit log: %w", err)
	}

	overlay, err := fs.NewOverlay(filepath.Join(cfg.RootDir, "fs"), cfg.AllowedDirs)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create overlay: %w", err)
	}

	return &Sandbox{
		cfg:     cfg,
		overlay: overlay,
		db:      db,
		id:      id,
	}, nil
}

func (s *Sandbox) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil {
		return fmt.Errorf("already running")
	}

	if s.cfg.Command == "" {
		return fmt.Errorf("no command")
	}

	s.db.Log("sandbox", "START", fmt.Sprintf("#%d command=%s args=%v", s.id, s.cfg.Command, s.cfg.Args))

	ctx := context.Background()
	if s.cfg.TimeoutSecs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(s.cfg.TimeoutSecs)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, s.cfg.Command, s.cfg.Args...)

	cmd.Env = env.SanitizeOS(s.overlay.Home())
	cmd.Dir = s.overlay.Home()

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if s.cfg.TimeoutSecs > 0 {
		go func() {
			<-ctx.Done()
			s.db.Log("sandbox", "TIMEOUT", fmt.Sprintf("#%d exceeded %ds", s.id, s.cfg.TimeoutSecs))
		}()
	}

	s.cmd = cmd
	s.started = time.Now()

	if err := cmd.Start(); err != nil {
		s.db.Log("sandbox", "ERROR", fmt.Sprintf("start failed: %v", err))
		return fmt.Errorf("start: %w", err)
	}

	s.db.Log("sandbox", "RUNNING", fmt.Sprintf("#%d pid=%d", s.id, cmd.Process.Pid))

	return nil
}

func (s *Sandbox) Wait() error {
	s.mu.Lock()
	if s.cmd == nil {
		s.mu.Unlock()
		return fmt.Errorf("not running")
	}
	s.mu.Unlock()

	err := s.cmd.Wait()
	s.done.Store(true)

	elapsed := time.Since(s.started)
	s.db.Log("sandbox", "EXIT", fmt.Sprintf("#%d duration=%s err=%v", s.id, elapsed, err))

	return err
}

func (s *Sandbox) Kill() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("not running")
	}

	s.db.Log("sandbox", "KILL", fmt.Sprintf("#%d pid=%d", s.id, s.cmd.Process.Pid))
	return s.cmd.Process.Kill()
}

func (s *Sandbox) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		s.db.Close()
	}
	if s.overlay != nil {
		s.overlay.Cleanup()
	}
	return nil
}

func (s *Sandbox) ID() int64        { return s.id }
func (s *Sandbox) Home() string     { return s.overlay.Home() }
func (s *Sandbox) Root() string      { return s.cfg.RootDir }
func (s *Sandbox) DB() *store.DB    { return s.db }
func (s *Sandbox) Overlay() *fs.Overlay { return s.overlay }
func (s *Sandbox) IsDone() bool      { return s.done.Load() }

func (s *Sandbox) AuditLog(category string, limit int) ([]store.Entry, error) {
	return s.db.Query(category, limit)
}

func (s *Sandbox) EnvSummary() []string {
	return env.SanitizeOS(s.overlay.Home())
}
