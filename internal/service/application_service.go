package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/process"
	"local-dev-workspace/internal/repository"
)

type ApplicationService struct {
	repo *repository.ApplicationRepo
	pm   *process.ProcessManager
	root string // workspace root
}

func NewApplicationService(repo *repository.ApplicationRepo, pm *process.ProcessManager, workspaceRoot string) *ApplicationService {
	return &ApplicationService{repo: repo, pm: pm, root: workspaceRoot}
}

// resolveCommand returns the CommandDef for a given name, or the first command if name is empty.
func resolveCommand(commands []model.CommandDef, name string) (model.CommandDef, error) {
	if len(commands) == 0 {
		return model.CommandDef{}, fmt.Errorf("no commands defined")
	}
	if name == "" {
		return commands[0], nil
	}
	for _, c := range commands {
		if c.Name == name {
			return c, nil
		}
	}
	return model.CommandDef{}, fmt.Errorf("command not found: %s", name)
}

func (s *ApplicationService) List() ([]model.Application, error) {
	return s.repo.List()
}

func (s *ApplicationService) Get(id int64) (*model.Application, error) {
	a, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("application not found")
	}
	return a, nil
}

func (s *ApplicationService) Create(req model.CreateApplicationRequest) (*model.Application, error) {
	if len(req.Commands) == 0 {
		return nil, fmt.Errorf("at least one command is required")
	}
	a := &model.Application{
		Name:             req.Name,
		Description:      req.Description,
		Commands:         req.Commands,
		WorkingDirectory: req.WorkingDirectory,
		Environment:      req.Environment,
		AutoStart:        req.AutoStart,
		Enabled:          true,
	}
	if a.Environment == "" {
		a.Environment = "{}"
	}
	if err := s.repo.Create(a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *ApplicationService) Update(id int64, req model.UpdateApplicationRequest) (*model.Application, error) {
	a, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("application not found")
	}
	if req.Name != nil {
		a.Name = *req.Name
	}
	if req.Description != nil {
		a.Description = *req.Description
	}
	if req.Commands != nil {
		if len(req.Commands) == 0 {
			return nil, fmt.Errorf("commands cannot be empty")
		}
		a.Commands = req.Commands
	}
	if req.WorkingDirectory != nil {
		a.WorkingDirectory = *req.WorkingDirectory
	}
	if req.Environment != nil {
		a.Environment = *req.Environment
	}
	if req.AutoStart != nil {
		a.AutoStart = *req.AutoStart
	}
	if req.Enabled != nil {
		a.Enabled = *req.Enabled
	}
	if err := s.repo.Update(a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *ApplicationService) Delete(id int64) error {
	a, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if a == nil {
		return fmt.Errorf("application not found")
	}

	// If running, stop first.
	run, _ := s.repo.GetLatestRun(id)
	if run != nil && (run.Status == "RUNNING" || run.Status == "STARTING") {
		s.Stop(id)
	}

	return s.repo.Delete(id)
}

func (s *ApplicationService) Start(id int64, commandName string) (*model.StartResponse, error) {
	a, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("application not found")
	}

	// Resolve which command to run
	cmd, err := resolveCommand(a.Commands, commandName)
	if err != nil {
		return nil, err
	}

	// Check if already running
	run, _ := s.repo.GetLatestRun(id)
	if run != nil && (run.Status == "RUNNING" || run.Status == "STARTING") {
		return nil, fmt.Errorf("application already running")
	}

	// Prepare output dirs
	appDir := filepath.Join(s.root, "applications", fmt.Sprintf("%d", id))
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app output dir: %w", err)
	}
	stdoutPath := filepath.Join(appDir, "stdout.log")
	stderrPath := filepath.Join(appDir, "stderr.log")

	// Create run record
	now := time.Now()
	run = &model.ApplicationRun{
		ApplicationID: id,
		CommandName:   cmd.Name,
		Status:        "STARTING",
		StartedAt:     &now,
	}
	if err := s.repo.CreateRun(run); err != nil {
		return nil, err
	}

	// Start process
	pid, err := s.pm.Start(cmd.Command, a.WorkingDirectory, a.Environment, stdoutPath, stderrPath)
	if err != nil {
		run.Status = "FAILED"
		s.repo.UpdateRun(run)
		return nil, fmt.Errorf("start failed: %w", err)
	}

	// Update run record
	run.PID = &pid
	run.Status = "RUNNING"
	if err := s.repo.UpdateRun(run); err != nil {
		return nil, err
	}

	return &model.StartResponse{
		ApplicationID: id,
		PID:           pid,
		CommandName:   cmd.Name,
		Status:        "RUNNING",
		StartedAt:     now,
	}, nil
}

func (s *ApplicationService) Stop(id int64) error {
	run, err := s.repo.GetLatestRun(id)
	if err != nil {
		return err
	}
	if run == nil || run.Status != "RUNNING" && run.Status != "STARTING" {
		return fmt.Errorf("application not running")
	}
	if run.PID == nil {
		return fmt.Errorf("no pid recorded")
	}

	run.Status = "STOPPING"
	s.repo.UpdateRun(run)

	if err := s.pm.Stop(*run.PID); err != nil {
		run.Status = "FAILED"
		s.repo.UpdateRun(run)
		return fmt.Errorf("stop failed: %w", err)
	}

	now := time.Now()
	run.Status = "STOPPED"
	run.StoppedAt = &now
	if err := s.repo.UpdateRun(run); err != nil {
		return err
	}
	return nil
}

func (s *ApplicationService) Restart(id int64, commandName string) (*model.StartResponse, error) {
	// Try to stop; ignore "not running" errors
	_ = s.Stop(id)

	// Wait a moment for port release
	time.Sleep(500 * time.Millisecond)

	return s.Start(id, commandName)
}

func (s *ApplicationService) Status(id int64) (*model.StatusResponse, error) {
	a, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("application not found")
	}

	return s.buildStatus(id, a), nil
}

// buildStatus computes the runtime status for an application. When the app is
// not actively RUNNING, runtime-only fields (pid, commandName, startedAt,
// uptime) are cleared so the frontend never shows stale "running" info on a
// stopped app.
func (s *ApplicationService) buildStatus(id int64, a *model.Application) *model.StatusResponse {
	run, _ := s.repo.GetLatestRun(id)
	status := "STOPPED"
	if run != nil {
		if run.Status == "RUNNING" || run.Status == "STARTING" {
			if run.PID != nil && s.pm.IsRunning(*run.PID) {
				status = "RUNNING"
			} else {
				status = "STOPPED"
			}
		} else {
			status = run.Status
		}
	}

	resp := &model.StatusResponse{
		ID:     id,
		Status: status,
	}

	// Only expose runtime fields while genuinely running.
	if status == "RUNNING" && run != nil {
		resp.PID = run.PID
		resp.CommandName = run.CommandName
		resp.StartedAt = run.StartedAt
		if run.StartedAt != nil {
			resp.Uptime = time.Since(*run.StartedAt).Round(time.Second).String()
		}
	}
	return resp
}

// StatusAll returns the runtime status for every application in one shot,
// eliminating the per-app N+1 polling on the frontend.
func (s *ApplicationService) StatusAll() ([]model.StatusResponse, error) {
	apps, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	items := make([]model.StatusResponse, 0, len(apps))
	for i := range apps {
		items = append(items, *s.buildStatus(apps[i].ID, &apps[i]))
	}
	return items, nil
}
