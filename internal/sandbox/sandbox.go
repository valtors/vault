package sandbox

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/valtors/vault/internal/store"
)

type Sandbox struct {
	cfg    Config
	db     *store.DB
	cmd    *exec.Cmd
	stopCh chan struct{}
}

func New(cfg Config, db *store.DB) *Sandbox {
	return &Sandbox{
		cfg:    cfg,
		db:     db,
		stopCh: make(chan struct{}),
	}
}

func (s *Sandbox) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command given")
	}

	if err := s.setupRoot(); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = s.cfg.RootDir
	cmd.Env = s.filteredEnv()

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
	}

	s.cmd = cmd
	s.db.Log("sandbox", "start", fmt.Sprintf("cmd=%s args=%v", args[0], args[1:]))

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	status := "ok"
	if err != nil {
		status = err.Error()
	}

	s.db.Log("sandbox", "stop", fmt.Sprintf("status=%s duration=%s", status, duration))
	return err
}

func (s *Sandbox) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Signal(syscall.SIGTERM)
		s.db.Log("sandbox", "stop", "signal=SIGTERM")
	}
	close(s.stopCh)
}

func (s *Sandbox) setupRoot() error {
	if err := os.MkdirAll(s.cfg.RootDir, 0755); err != nil {
		return err
	}

	for _, dir := range []string{"bin", "tmp", "dev", "proc", "home"} {
		if err := os.MkdirAll(filepath.Join(s.cfg.RootDir, dir), 0755); err != nil {
			return err
		}
	}

	nullDev := filepath.Join(s.cfg.RootDir, "dev", "null")
	if _, err := os.Stat(nullDev); os.IsNotExist(err) {
		syscall.Mknod(nullDev, syscall.S_IFCHR|0666, 0)
	}

	return nil
}

func (s *Sandbox) filteredEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") {
			env = append(env, "HOME=/home")
			continue
		}
		if strings.HasPrefix(e, "USER=") {
			env = append(env, "USER=agent")
			continue
		}
		if strings.HasPrefix(e, "VAULT_") {
			continue
		}
		if strings.Contains(e, "KEY") || strings.Contains(e, "TOKEN") || strings.Contains(e, "SECRET") || strings.Contains(e, "PASSWORD") {
			continue
		}
		env = append(env, e)
	}
	return env
}

func (s *Sandbox) PipeLogs(r io.Reader, source string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s.db.Log("pipe", source, scanner.Text())
	}
}
