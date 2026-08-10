package rag

import (
	"os"
	"path/filepath"
)

// EnsureDir creates a directory if it does not exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ListDirs lists subdirectories in the given path.
func ListDirs(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if dirs == nil {
		dirs = []string{}
	}
	return dirs, nil
}

// GetWorkspaceDirs returns the full RAG and Index paths for a group.
func GetWorkspaceDirs(workspaceRoot, groupName string) (ragPath, indexPath string) {
	ragPath = filepath.Join(workspaceRoot, "rag", groupName)
	indexPath = filepath.Join(workspaceRoot, "index", groupName)
	return
}
