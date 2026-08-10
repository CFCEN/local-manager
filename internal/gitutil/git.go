package gitutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ScanDirectory recursively finds .git repos under root up to maxDepth.
func ScanDirectory(root string, maxDepth int) ([]GitInfo, error) {
	var repos []GitInfo
	err := scanRecursive(root, root, 0, maxDepth, &repos)
	return repos, err
}

type GitInfo struct {
	Name   string
	Path   string
	Branch string
	Remote string
}

func scanRecursive(baseDir, currentDir string, depth, maxDepth int, repos *[]GitInfo) error {
	if maxDepth > 0 && depth > maxDepth {
		return filepath.SkipDir
	}

	gitDir := filepath.Join(currentDir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		name := filepath.Base(currentDir)
		branch := getBranch(currentDir)
		remote := getRemote(currentDir)
		*repos = append(*repos, GitInfo{Name: name, Path: currentDir, Branch: branch, Remote: remote})
		return filepath.SkipDir
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			subDir := filepath.Join(currentDir, entry.Name())
			scanRecursive(baseDir, subDir, depth+1, maxDepth, repos)
		}
	}
	return nil
}

func getBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getRemote(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func GetStatus(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return "UNKNOWN"
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return "Clean"
	}
	// check ahead/behind
	cmd2 := exec.Command("git", "-C", repoPath, "status", "-sb")
	out2, err := cmd2.Output()
	if err != nil {
		return "Modified"
	}
	s := string(out2)
	if strings.Contains(s, "[ahead") {
		return "Ahead"
	}
	if strings.Contains(s, "[behind") {
		return "Behind"
	}
	return "Modified"
}

func GetLastCommit(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%h %s")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GetStatusContext is like GetStatus but honors a context deadline so a single
// slow/broken repo cannot stall a batch request. Returns "UNKNOWN" on any
// error or timeout.
func GetStatusContext(ctx context.Context, repoPath string) string {
	out, err := runGit(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return "UNKNOWN"
	}
	if len(strings.TrimSpace(out)) == 0 {
		return "Clean"
	}
	out2, err := runGit(ctx, repoPath, "status", "-sb")
	if err != nil {
		return "Modified"
	}
	if strings.Contains(out2, "[ahead") {
		return "Ahead"
	}
	if strings.Contains(out2, "[behind") {
		return "Behind"
	}
	return "Modified"
}

// GetLastCommitContext is the context-aware variant of GetLastCommit.
func GetLastCommitContext(ctx context.Context, repoPath string) string {
	out, err := runGit(ctx, repoPath, "log", "-1", "--format=%h %s")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func runGit(ctx context.Context, repoPath string, args ...string) (string, error) {
	full := append([]string{"-C", repoPath}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	out, err := cmd.Output()
	return string(out), err
}

// DefaultGitTimeout bounds a single git invocation during enrichment.
const DefaultGitTimeout = 2 * time.Second
