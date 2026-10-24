package api

import (
	"github.com/gorilla/mux"

	"local-dev-workspace/internal/config"
	"local-dev-workspace/internal/process"
	"local-dev-workspace/internal/repository"
	"local-dev-workspace/internal/service"
)

func NewRouter(cfg *config.AppConfig) *mux.Router {
	r := mux.NewRouter()

	r.Use(RecoveryMiddleware)
	r.Use(LoggingMiddleware)

	// Repos
	configRepo := repository.NewConfigRepo()
	logRepo := repository.NewLogRepo()
	appRepo := repository.NewApplicationRepo()
	gitRepo := repository.NewGitRepo()
	workspaceRepo := repository.NewWorkspaceRepo()
	dataTestRepo := repository.NewDataTestRepo()
	automationRepo := repository.NewAutomationRepo()

	// System services
	pm := process.NewProcessManager()

	// Services
	configSvc := service.NewConfigService(configRepo)
	logSvc := service.NewLogService(logRepo)
	appSvc := service.NewApplicationService(appRepo, pm, cfg.WorkspaceRoot)
	gitSvc := service.NewGitService(gitRepo, cfg.WorkspaceRoot)
	workspaceSvc := service.NewWorkspaceService(workspaceRepo, cfg.WorkspaceRoot)
	dashboardSvc := service.NewDashboardService(appSvc, gitSvc, logSvc, configSvc)
	dataTestSvc := service.NewDataTestService(dataTestRepo)
	automationHub := service.NewAutomationHub()
	automationSvc := service.NewAutomationService(automationRepo, automationHub)

	// Monitor
	monitor := process.NewApplicationMonitor(pm, appRepo)
	monitor.Start()

	// Handlers
	configH := NewConfigHandler(configSvc)
	logH := NewLogHandler(logSvc)
	appH := NewApplicationHandler(appSvc)
	gitH := NewGitHandler(gitSvc, workspaceSvc)
	workspaceH := NewWorkspaceHandler(workspaceSvc)
	dashboardH := NewDashboardHandler(dashboardSvc)
	dataTestH := NewDataTestHandler(dataTestSvc)
	automationH := NewAutomationHandler(automationSvc, automationHub, cfg.AutomationToken, cfg.AllowedOrigins)

	api := r.PathPrefix("/api").Subrouter()

	registerConfigRoutes(api, configH)
	registerLogRoutes(api, logH)
	registerApplicationRoutes(api, appH)
	registerGitRoutes(api, gitH)
	registerWorkspaceRoutes(api, workspaceH)
	registerDashboardRoutes(api, dashboardH)
	registerDataTestRoutes(api, dataTestH)
	registerAutomationRoutes(api, automationH)

	return r
}
