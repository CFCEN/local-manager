package model

import "time"

type Adapter struct {
	ID        int64     `json:"id"`
	AdapterID string    `json:"adapterId"`
	Name      string    `json:"name"`
	Version   string    `json:"version,omitempty"`
	Online    bool      `json:"online"`
	LastSeen  time.Time `json:"lastSeen"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type AutomationProject struct {
	ID        int64     `json:"id"`
	AdapterID string    `json:"adapterId"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name,omitempty"`
	Path      string    `json:"path,omitempty"`
	Payload   any       `json:"payload,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type AutomationSession struct {
	ID        int64     `json:"id"`
	AdapterID string    `json:"adapterId"`
	ProjectID string    `json:"projectId"`
	SessionID string    `json:"sessionId"`
	Status    string    `json:"status,omitempty"`
	Cursor    string    `json:"cursor,omitempty"`
	Payload   any       `json:"payload,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type AutomationEvent struct {
	ID        int64     `json:"id"`
	AdapterID string    `json:"adapterId"`
	ProjectID string    `json:"projectId,omitempty"`
	SessionID string    `json:"sessionId,omitempty"`
	EventID   string    `json:"eventId"`
	Type      string    `json:"type"`
	Cursor    string    `json:"cursor,omitempty"`
	Payload   any       `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}
type PendingAction struct {
	ID        int64     `json:"id"`
	AdapterID string    `json:"adapterId"`
	ProjectID string    `json:"projectId,omitempty"`
	SessionID string    `json:"sessionId,omitempty"`
	ActionID  string    `json:"actionId"`
	RequestID string    `json:"requestId,omitempty"`
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
	Status    string    `json:"status"`
	Result    any       `json:"result,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type Operation struct {
	ID        int64     `json:"id"`
	RequestID string    `json:"requestId"`
	AdapterID string    `json:"adapterId"`
	SessionID string    `json:"sessionId"`
	Status    string    `json:"status"`
	Error     any       `json:"error,omitempty"`
	Result    any       `json:"result,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
