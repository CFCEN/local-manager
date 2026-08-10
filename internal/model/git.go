package model

import "time"

type GitProject struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Branch     string    `json:"branch"`
	Remote     string    `json:"remote"`
	GroupID    *int64    `json:"groupId"`
	GroupName  string    `json:"groupName"`
	Status     string    `json:"status,omitempty"`
	LastCommit string    `json:"lastCommit,omitempty"`
	RAGPath    string    `json:"ragPath"`
	IndexPath  string    `json:"indexPath"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type GitProjectDetail struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Branch      string    `json:"branch"`
	Remote      string    `json:"remote"`
	Status      string    `json:"status"`
	LastCommit  string    `json:"lastCommit"`
	GroupID     *int64    `json:"groupId"`
	GroupName   string    `json:"groupName"`
	RAGPath     string    `json:"ragPath"`
	IndexPath   string    `json:"indexPath"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ScanRequest struct {
	Path     string `json:"path"`
	MaxDepth int    `json:"maxDepth"`
}

type ScanResult struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Branch string `json:"branch"`
	Remote string `json:"remote"`
}

type UpdateGitProjectRequest struct {
	Name      *string `json:"name,omitempty"`
	GroupID   *int64  `json:"groupId,omitempty"`
	RAGPath   *string `json:"ragPath,omitempty"`
	IndexPath *string `json:"indexPath,omitempty"`
}

type GitGroup struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	RAGPath      string    `json:"ragPath"`
	IndexPath    string    `json:"indexPath"`
	ProjectCount int64     `json:"projectCount"`
	ContextCount int64     `json:"contextCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// GitProjectStatusItem is a lightweight status payload for batch enrich.
type GitProjectStatusItem struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	LastCommit string `json:"lastCommit"`
}

type BatchGitStatusRequest struct {
	IDs []int64 `json:"ids"`
}

type CreateGitGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RAGPath     string `json:"ragPath"`
	IndexPath   string `json:"indexPath"`
}

type UpdateGitGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	RAGPath     *string `json:"ragPath,omitempty"`
	IndexPath   *string `json:"indexPath,omitempty"`
}

type AssignGroupRequest struct {
	GroupID int64 `json:"groupId"`
}

type BatchAssignGroupRequest struct {
	ProjectIDs []int64 `json:"projectIds"`
	GroupID    int64   `json:"groupId"`
}
