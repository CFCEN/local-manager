package model

import "time"

type ConfigFile struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	GroupName   string    `json:"groupName"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateConfigRequest struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	GroupName   string `json:"groupName"`
}

type UpdateConfigRequest struct {
	Name        *string `json:"name,omitempty"`
	Path        *string `json:"path,omitempty"`
	Description *string `json:"description,omitempty"`
	GroupName   *string `json:"groupName,omitempty"`
}

type UpdateConfigContentRequest struct {
	Content string `json:"content"`
}

type ConfigDetail struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	GroupName   string    `json:"groupName"`
	Content     string    `json:"content"`
	Size        int64     `json:"size"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
