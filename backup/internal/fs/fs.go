package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Returns true if the given directory does not exist or is empty.
func IsDirEmpty(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	return len(files) == 0, nil
}

func DirExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, errors.New("not a directory")
	}
	return true, nil
}

func Exists(file string) (bool, error) {
	_, err := os.Stat(file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func CreateDir(dir string) error {
	err := os.MkdirAll(dir, 0775)
	return err
}

func JoinPath(a, b string) string {
	return filepath.Join(a, b)
}

func ParentPath(path string) string {
	return filepath.Dir(filepath.Clean(path))
}

func BasePath(path string) string {
	return filepath.Base(filepath.Clean(path))
}

func AbsPath(path string) (string, error) {
	path = filepath.Clean(path)
	switch {
	case path == "~":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	case strings.HasPrefix(path, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	case filepath.IsAbs(path):
		return path, nil
	default:
		path, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return path, nil
	}
}

func DefaultBackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	date := time.Now().Format(time.DateOnly)
	dir := filepath.Join(home, fmt.Sprintf("backup-%v", date))
	return AbsPath(dir)
}

func FileSize(file string) (int64, error) {
	info, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
