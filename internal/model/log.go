package model

import "time"

type LogGroup struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FileCount   int64     `json:"fileCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateLogGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateLogGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type LogFile struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	GroupID     *int64    `json:"groupId"`
	GroupName   string    `json:"groupName"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateLogFileRequest struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	GroupID     *int64 `json:"groupId"`
	Description string `json:"description"`
}

type UpdateLogFileRequest struct {
	Name        *string `json:"name,omitempty"`
	Path        *string `json:"path,omitempty"`
	GroupID     *int64  `json:"groupId,omitempty"`
	Description *string `json:"description,omitempty"`
}

type LogContent struct {
	Content       string    `json:"content"`
	Size          int64     `json:"size"`
	ModifiedAt    time.Time `json:"modifiedAt"`
	TotalLines    int64     `json:"totalLines"`
	ReturnedLines int       `json:"returnedLines"`
	Truncated     bool      `json:"truncated"`
}
