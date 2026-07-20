package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Overlay struct {
	root    string
	allowed []string
	blocked []string
}

func NewOverlay(root string, allowed, blocked []string) *Overlay {
	return &Overlay{
		root:    root,
		allowed: allowed,
		blocked: blocked,
	}
}

func (o *Overlay) Resolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if o.isBlocked(abs) {
		return "", fmt.Errorf("denied: %s", path)
	}

	if o.isAllowed(abs) {
		return abs, nil
	}

	overlayPath := filepath.Join(o.root, abs)
	return overlayPath, nil
}

func (o *Overlay) isBlocked(path string) bool {
	for _, b := range o.blocked {
		if strings.HasPrefix(path, b) {
			return true
		}
	}
	return false
}

func (o *Overlay) isAllowed(path string) bool {
	for _, a := range o.allowed {
		if strings.HasPrefix(path, a) {
			return true
		}
	}
	return false
}

func (o *Overlay) ReadFile(path string) ([]byte, error) {
	resolved, err := o.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(resolved)
}

func (o *Overlay) WriteFile(path string, data []byte) error {
	resolved, err := o.Resolve(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return err
	}

	return os.WriteFile(resolved, data, 0644)
}

func (o *Overlay) ListDir(path string) ([]os.DirEntry, error) {
	resolved, err := o.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(resolved)
}

func (o *Overlay) Stat(path string) (os.FileInfo, error) {
	resolved, err := o.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.Stat(resolved)
}

func (o *Overlay) Exists(path string) bool {
	resolved, err := o.Resolve(path)
	if err != nil {
		return false
	}
	_, err = os.Stat(resolved)
	return err == nil
}
