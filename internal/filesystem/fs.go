package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NormalizePath expands ~ and $HOME, returns absolute canonical path.
func NormalizePath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[2:])
	}
	if strings.HasPrefix(p, "$HOME/") {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable not set")
		}
		p = filepath.Join(home, p[6:])
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// ValidatePath checks that the path exists, is a regular file, and is readable.
func ValidatePath(p string) error {
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", p)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", p)
	}
	return nil
}

// ReadFile reads file content. Returns content, size, and mod time.
func ReadFile(p string) (string, int64, time.Time, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	return string(data), info.Size(), info.ModTime(), nil
}

// SafeWrite writes content to path atomically: backup → temp → rename.
func SafeWrite(p, content string) error {
	// read old content
	oldData, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// create backup
	if err == nil {
		backupPath := p + ".bak"
		if err := os.WriteFile(backupPath, oldData, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	dir := filepath.Dir(p)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, p); err != nil {
		return fmt.Errorf("failed to replace file: %w", err)
	}
	return nil
}

// DirEntry describes a file or directory in a listing.
type DirEntry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"isDir"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// ListDir returns the contents of a directory (files only, no recursion).
func ListDir(dir string) ([]DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var result []DirEntry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, DirEntry{
			Name:       e.Name(),
			Path:       filepath.Join(dir, e.Name()),
			Size:       info.Size(),
			IsDir:      e.IsDir(),
			ModifiedAt: info.ModTime(),
		})
	}
	if result == nil {
		result = []DirEntry{}
	}
	return result, nil
}

// DeleteFile removes a file at the given path.
func DeleteFile(p string) error {
	return os.Remove(p)
}

// AllowedTextExtensions lists file extensions that can be read/written.
var AllowedTextExtensions = map[string]bool{
	".md":   true,
	".txt":  true,
	".json": true,
	".yaml": true,
	".yml":  true,
	".xml":  true,
	".csv":  true,
	".log":  true,
	".toml": true,
	".ini":  true,
	".cfg":  true,
	".conf": true,
}

// MaxFileSize is the maximum file size for context file operations (1MB).
const MaxFileSize = 1 << 20

// IsTextFile checks whether the filename has an allowed text extension.
func IsTextFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return AllowedTextExtensions[ext]
}

// ValidateFileName checks that a file name doesn't contain path traversal characters.
func ValidateFileName(name string) error {
	if name == "" {
		return fmt.Errorf("file name cannot be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid file name")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("file name contains path separators")
	}
	return nil
}

// FileExists checks whether a file exists and is not a directory.
func FileExists(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ToDisplayPath converts absolute path to ~/ form if under home dir.
func ToDisplayPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
