package model

import "time"

type ProjectContext struct {
	ID            int64      `json:"id"`
	GroupID       int64      `json:"groupId"`
	Name          string     `json:"name"`
	RAGPath       string     `json:"ragPath"`
	IndexPath     string     `json:"indexPath"`
	IndexStatus   string     `json:"indexStatus"`
	LastIndexedAt *time.Time `json:"lastIndexedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type CreateContextRequest struct {
	Name      string `json:"name"`
	RAGPath   string `json:"ragPath"`
	IndexPath string `json:"indexPath"`
}

type UpdateContextRequest struct {
	Name      *string `json:"name,omitempty"`
	RAGPath   *string `json:"ragPath,omitempty"`
	IndexPath *string `json:"indexPath,omitempty"`
}

type WorkspaceInfo struct {
	Root      string           `json:"root"`
	RAGDirs   []string         `json:"ragDirs"`
	IndexDirs []string         `json:"indexDirs"`
	Contexts  []ProjectContext `json:"contexts"`
}

// --- Context Files ---

type ContextFile struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"isDir"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type ContextFileContent struct {
	Path       string    `json:"path"`
	Content    string    `json:"content"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type CreateFileRequest struct {
	Type     string `json:"type"`
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

type UpdateFileRequest struct {
	Type     string `json:"type"`
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}
