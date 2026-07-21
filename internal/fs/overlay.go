package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	SandboxRoot = ".vault"
	SandboxHome = "home"
	SandboxTmp  = "tmp"
)

var blockedPaths = []string{
	".ssh",
	".aws",
	".gnupg",
	".docker",
	".kube",
	".config/gcloud",
	".config/gh",
	".npmrc",
	".pypirc",
	".netrc",
	".env",
	".gitconfig",
}

type Overlay struct {
	root    string
	home    string
	tmp     string
	allowed []string
}

func NewOverlay(root string, allowed []string) (*Overlay, error) {
	home := filepath.Join(root, SandboxHome)
	tmp := filepath.Join(root, SandboxTmp)

	for _, dir := range []string{root, home, tmp} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("create sandbox dir %s: %w", dir, err)
		}
	}

	for _, src := range allowed {
		if !isReadable(src) {
			continue
		}
		name := sanitizeName(src)
		link := filepath.Join(root, "allowed", name)
		if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
			continue
		}
		abs, _ := filepath.Abs(src)
		_ = os.Symlink(abs, link)
	}

	return &Overlay{
		root:    root,
		home:    home,
		tmp:     tmp,
		allowed: allowed,
	}, nil
}

func (o *Overlay) Home() string  { return o.home }
func (o *Overlay) Tmp() string    { return o.tmp }
func (o *Overlay) Root() string   { return o.root }

func (o *Overlay) Resolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}

	for _, blocked := range blockedPaths {
		bp := filepath.Join(o.home, blocked)
		if abs == bp || strings.HasPrefix(abs, bp+"/") {
			return "", fmt.Errorf("blocked: %s is a sensitive path", path)
		}
	}

	if strings.HasPrefix(abs, o.home) || strings.HasPrefix(abs, o.tmp) {
		return abs, nil
	}

	allowedLink := filepath.Join(o.root, "allowed")
	if strings.HasPrefix(abs, allowedLink) {
		return abs, nil
	}

	return "", fmt.Errorf("blocked: %s is outside the sandbox", path)
}

func (o *Overlay) ReadFile(path string) ([]byte, error) {
	resolved, err := o.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(resolved)
}

func (o *Overlay) WriteFile(path string, data []byte, perm os.FileMode) error {
	resolved, err := o.Resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0700); err != nil {
		return err
	}
	return os.WriteFile(resolved, data, perm)
}

func (o *Overlay) Cleanup() error {
	return os.RemoveAll(o.root)
}

func (o *Overlay) ListAllowed() []string {
	return o.allowed
}

func IsBlockedPath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	home, _ := os.UserHomeDir()
	for _, blocked := range blockedPaths {
		bp := filepath.Join(home, blocked)
		if abs == bp || strings.HasPrefix(abs, bp+"/") {
			return true
		}
	}
	return false
}

func isReadable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return true
	}
	return true
}

func sanitizeName(path string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(strings.TrimPrefix(path, "/"))
}

func CreateTempFile(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

func DiskUsage(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func SafeRemove(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if abs == "/" || abs == "" {
		return fmt.Errorf("refusing to remove root")
	}
	return os.RemoveAll(abs)
}
