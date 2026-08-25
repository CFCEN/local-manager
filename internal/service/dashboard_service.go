package service

import (
	"local-dev-workspace/internal/model"
)

// DashboardService aggregates counts and recent items for the dashboard page,
// so the frontend can issue a single request instead of four.
type DashboardService struct {
	appSvc    *ApplicationService
	gitSvc    *GitService
	logSvc    *LogService
	configSvc *ConfigService
}

func NewDashboardService(appSvc *ApplicationService, gitSvc *GitService, logSvc *LogService, configSvc *ConfigService) *DashboardService {
	return &DashboardService{appSvc: appSvc, gitSvc: gitSvc, logSvc: logSvc, configSvc: configSvc}
}

const dashboardRecentLimit = 5

func (s *DashboardService) Summary() (*model.DashboardSummary, error) {
	apps, err := s.appSvc.List()
	if err != nil {
		return nil, err
	}
	projects, err := s.gitSvc.ListProjects(false)
	if err != nil {
		return nil, err
	}
	logFiles, err := s.logSvc.ListFiles()
	if err != nil {
		return nil, err
	}
	configs, err := s.configSvc.List()
	if err != nil {
		return nil, err
	}

	// Live running status via ProcessManager (not guessed from DB).
	statuses, err := s.appSvc.StatusAll()
	if err != nil {
		return nil, err
	}
	statusByID := make(map[int64]string, len(statuses))
	runningCount := 0
	for _, st := range statuses {
		statusByID[st.ID] = st.Status
		if st.Status == "RUNNING" {
			runningCount++
		}
	}

	summary := &model.DashboardSummary{
		ApplicationCount: len(apps),
		RunningCount:     runningCount,
		GitProjectCount:  len(projects),
		LogFileCount:     len(logFiles),
		ConfigCount:      len(configs),
		RecentApps:       make([]model.DashboardApp, 0, dashboardRecentLimit),
		RecentProjects:   make([]model.DashboardProject, 0, dashboardRecentLimit),
	}

	// apps and projects are already ordered by updated_at DESC from the repo.
	for i := 0; i < len(apps) && i < dashboardRecentLimit; i++ {
		a := apps[i]
		status := statusByID[a.ID]
		if status == "" {
			status = "STOPPED"
		}
		summary.RecentApps = append(summary.RecentApps, model.DashboardApp{
			ID:               a.ID,
			Name:             a.Name,
			Path:             a.WorkingDirectory,
			Status:           status,
			UpdatedAt:        a.UpdatedAt,
			WorkingDirectory: a.WorkingDirectory,
		})
	}
	for i := 0; i < len(projects) && i < dashboardRecentLimit; i++ {
		p := projects[i]
		summary.RecentProjects = append(summary.RecentProjects, model.DashboardProject{
			ID:        p.ID,
			Name:      p.Name,
			Path:      p.Path,
			UpdatedAt: p.UpdatedAt,
		})
	}

	return summary, nil
}
