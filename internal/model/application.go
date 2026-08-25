package model

import "time"

// CommandDef represents a named command within an application.
type CommandDef struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type Application struct {
	ID               int64        `json:"id"`
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	Commands         []CommandDef `json:"commands"`
	WorkingDirectory string       `json:"workingDirectory"`
	Environment      string       `json:"environment"`
	AutoStart        bool         `json:"autoStart"`
	Enabled          bool         `json:"enabled"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

type CreateApplicationRequest struct {
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	Commands         []CommandDef `json:"commands"`
	WorkingDirectory string       `json:"workingDirectory"`
	Environment      string       `json:"environment,omitempty"`
	AutoStart        bool         `json:"autoStart"`
}

type UpdateApplicationRequest struct {
	Name             *string      `json:"name,omitempty"`
	Description      *string      `json:"description,omitempty"`
	Commands         []CommandDef `json:"commands,omitempty"`
	WorkingDirectory *string      `json:"workingDirectory,omitempty"`
	Environment      *string      `json:"environment,omitempty"`
	AutoStart        *bool        `json:"autoStart,omitempty"`
	Enabled          *bool        `json:"enabled,omitempty"`
}

type ApplicationRun struct {
	ID            int64      `json:"id"`
	ApplicationID int64      `json:"applicationId"`
	PID           *int       `json:"pid"`
	CommandName   string     `json:"commandName"`
	StartedAt     *time.Time `json:"startedAt"`
	StoppedAt     *time.Time `json:"stoppedAt"`
	ExitCode      *int       `json:"exitCode"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type StartResponse struct {
	ApplicationID int64     `json:"applicationId"`
	PID           int       `json:"pid"`
	CommandName   string    `json:"commandName"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"startedAt"`
}

type StatusResponse struct {
	ID          int64      `json:"id"`
	PID         *int       `json:"pid"`
	CommandName string     `json:"commandName"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"startedAt"`
	Uptime      string     `json:"uptime,omitempty"`
}

// DashboardSummary is the aggregated payload for the dashboard page.
type DashboardSummary struct {
	ApplicationCount int                `json:"applicationCount"`
	RunningCount     int                `json:"runningCount"`
	GitProjectCount  int                `json:"gitProjectCount"`
	LogFileCount     int                `json:"logFileCount"`
	ConfigCount      int                `json:"configCount"`
	RecentApps       []DashboardApp     `json:"recentApps"`
	RecentProjects   []DashboardProject `json:"recentProjects"`
}

type DashboardApp struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Path             string    `json:"path"`
	Status           string    `json:"status"`
	UpdatedAt        time.Time `json:"updatedAt"`
	WorkingDirectory string    `json:"workingDirectory"`
}

type DashboardProject struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Status    string    `json:"status,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}
