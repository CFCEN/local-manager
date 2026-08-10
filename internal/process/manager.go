package process

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ProcessManager struct{}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{}
}

// Start launches a command via /bin/zsh -ilc (interactive login shell).
// It returns the long-lived child PID when a CLI daemonizes and prints one,
// otherwise it returns the shell PID.
// stdout/stderr are redirected to the given file paths if non-empty.
func (pm *ProcessManager) Start(command, workDir, envJSON string, stdoutPath, stderrPath string) (int, error) {
	// Build environment: inherit os.Environ() + parse envJSON
	env, err := buildEnv(envJSON)
	if err != nil {
		return 0, err
	}

	if workDir != "" {
		resolved, err := expandPath(workDir)
		if err != nil {
			return 0, err
		}
		workDir = resolved
	}

	cmd := exec.Command("/bin/zsh", "-ilc", command)
	cmd.Dir = workDir
	cmd.Env = env

	var stdoutFile *os.File
	if stdoutPath != "" {
		stdoutFile, err = os.Create(stdoutPath)
		if err != nil {
			return 0, fmt.Errorf("failed to create stdout file: %w", err)
		}
		cmd.Stdout = stdoutFile
	}
	var stderrFile *os.File
	if stderrPath != "" {
		stderrFile, err = os.Create(stderrPath)
		if err != nil {
			if stdoutFile != nil {
				stdoutFile.Close()
			}
			return 0, fmt.Errorf("failed to create stderr file: %w", err)
		}
		cmd.Stderr = stderrFile
	}

	// Set process group so we can kill the whole tree later
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		if stdoutFile != nil {
			stdoutFile.Close()
		}
		if stderrFile != nil {
			stderrFile.Close()
		}
		return 0, fmt.Errorf("failed to start command: %w", err)
	}

	pid := cmd.Process.Pid
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		if stdoutFile != nil {
			stdoutFile.Close()
		}
		if stderrFile != nil {
			stderrFile.Close()
		}
		close(done)
	}()

	// Some local CLIs, such as claude-adapter start, daemonize and exit after
	// printing the real background PID. Prefer that PID so status/stop stay useful.
	if stdoutPath != "" {
		if childPID, ok := waitForPrintedBackgroundPID(stdoutPath, done, 1500*time.Millisecond); ok {
			pid = childPID
		}
	}

	return pid, nil
}

// Stop kills the entire process tree rooted at pid.
// On macOS, uses ps to find children, then kills the process group.
func (pm *ProcessManager) Stop(pid int) error {
	// Try to kill the process group first (negative PID signals whole group)
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
	}

	// Also find and kill child processes individually
	children := findChildPIDs(pid)
	for _, cp := range children {
		p, err := os.FindProcess(cp)
		if err == nil {
			p.Signal(syscall.SIGTERM)
		}
	}

	// Verify the parent is dead
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil // already gone
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return nil // already dead
	}
	return nil
}

// IsRunning checks if a process with the given PID exists.
func (pm *ProcessManager) IsRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func buildEnv(envJSON string) ([]string, error) {
	env := os.Environ()
	if strings.TrimSpace(envJSON) == "" {
		return env, nil
	}

	var extra map[string]string
	if err := json.Unmarshal([]byte(envJSON), &extra); err != nil {
		return nil, fmt.Errorf("invalid environment JSON: %w", err)
	}
	for key, value := range extra {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env, nil
}

func expandPath(pathValue string) (string, error) {
	if pathValue == "~" || strings.HasPrefix(pathValue, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		if pathValue == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(pathValue, "~/")), nil
	}
	return pathValue, nil
}

func waitForPrintedBackgroundPID(stdoutPath string, done <-chan struct{}, timeout time.Duration) (int, bool) {
	deadline := time.Now().Add(timeout)
	pidPattern := regexp.MustCompile(`(?i)background \(pid (\d+)\)|running \(pid (\d+)\)|started.*pid[ =:]+(\d+)`)

	for {
		if pid, ok := parsePrintedPID(stdoutPath, pidPattern); ok {
			return pid, true
		}

		select {
		case <-done:
			if pid, ok := parsePrintedPID(stdoutPath, pidPattern); ok {
				return pid, true
			}
			return 0, false
		default:
		}

		if time.Now().After(deadline) {
			return 0, false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func parsePrintedPID(stdoutPath string, pattern *regexp.Regexp) (int, bool) {
	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		return 0, false
	}
	matches := pattern.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		return 0, false
	}
	last := matches[len(matches)-1]
	for _, value := range last[1:] {
		if value == "" {
			continue
		}
		pid, err := strconv.Atoi(value)
		if err == nil && pid > 0 {
			return pid, true
		}
	}
	return 0, false
}

// findChildPIDs returns all descendant PIDs of the given PID using ps.
func findChildPIDs(ppid int) []int {
	// macOS-compatible: ps -eo pid,ppid
	out, err := exec.Command("ps", "-eo", "pid,ppid").Output()
	if err != nil {
		return nil
	}

	var children []int
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		parent, err2 := strconv.Atoi(fields[1])
		if err1 == nil && err2 == nil && parent == ppid && pid != ppid {
			children = append(children, pid)
			// recursively find grandchildren
			children = append(children, findChildPIDs(pid)...)
		}
	}
	return children
}
