package validate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var blocked = []string{"/etc", "/proc", "/sys", "/root", "/mnt", "/media", "/host", "/var/run/docker.sock"}

func SafePath(p string) (string, error) {
	if p == "" || !strings.HasPrefix(p, "/") {
		return "", errors.New("path must be absolute")
	}
	if strings.Contains(p, "..") {
		return "", errors.New("path traversal is not allowed")
	}
	clean := filepath.Clean(p)
	for _, b := range blocked {
		if clean == b || strings.HasPrefix(clean, b+"/") {
			return "", errors.New("path is blocked")
		}
	}
	if fi, err := os.Lstat(clean); err == nil && fi.Mode()&02000000000 != 0 {
		return "", errors.New("symlinks are not allowed")
	}
	return clean, nil
}
