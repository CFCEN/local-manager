package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/service"
)

type AutomationHandler struct {
	s       *service.AutomationService
	h       *service.AutomationHub
	token   string
	origins []string
}

func NewAutomationHandler(s *service.AutomationService, h *service.AutomationHub, token string, origins []string) *AutomationHandler {
	return &AutomationHandler{s: s, h: h, token: token, origins: origins}
}

func registerAutomationRoutes(r *mux.Router, x *AutomationHandler) {
	r.HandleFunc("/claude-automation/sessions", x.Sessions).Methods("GET")
	r.HandleFunc("/claude-automation/adapters/{adapterId}/sessions/{sessionId}", x.Session).Methods("GET")
	r.HandleFunc("/claude-automation/adapters/{adapterId}/sessions/{sessionId}/messages", x.Message).Methods("POST")
	r.HandleFunc("/claude-automation/adapters/{adapterId}/sessions/{sessionId}/events", x.Events).Methods("GET")
	r.HandleFunc("/claude-automation/adapters/{adapterId}/sessions/{sessionId}/operations", x.Operations).Methods("GET")
	r.HandleFunc("/claude-automation/adapters/{adapterId}/sessions/{sessionId}/actions/{actionId}/respond", x.Respond).Methods("POST")
	r.HandleFunc("/claude-automation/adapters/ws", x.AdapterWS).Methods("GET")
	r.HandleFunc("/claude-automation/ws", x.BrowserWS).Methods("GET")
}

func (x *AutomationHandler) authorized(r *http.Request) bool {
	return x.token != "" && (r.URL.Query().Get("token") == x.token || r.Header.Get("Authorization") == "Bearer "+x.token)
}

func (x *AutomationHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return true
	}
	for _, allowed := range x.origins {
		if strings.TrimRight(allowed, "/") == origin {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
}}

func (x *AutomationHandler) AdapterWS(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	u := upgrader
	u.CheckOrigin = x.checkOrigin
	c, err := u.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	log.Printf("automation adapter websocket accepted remote=%s", r.RemoteAddr)
	var hello struct {
		Type       string `json:"type"`
		InstanceID string `json:"instanceId"`
	}
	if err = c.ReadJSON(&hello); err != nil || hello.Type != "hello" || hello.InstanceID == "" {
		return
	}
	adapter := model.Adapter{AdapterID: hello.InstanceID, Name: hello.InstanceID, Version: "1"}
	if err = x.s.Hello(adapter); err != nil {
		log.Printf("automation adapter hello failed adapter=%s err=%v", adapter.AdapterID, err)
		return
	}
	log.Printf("automation adapter connected adapter=%s remote=%s", adapter.AdapterID, r.RemoteAddr)
	out := x.h.AddAdapter(adapter.AdapterID)
	done := make(chan struct{})
	var writeMu sync.Mutex
	write := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return c.WriteJSON(v)
	}
	go func() {
		for {
			select {
			case b := <-out:
				writeMu.Lock()
				err := c.WriteMessage(websocket.TextMessage, b)
				writeMu.Unlock()
				if err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()
	defer close(done)
	c.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.SetPongHandler(func(string) error { return c.SetReadDeadline(time.Now().Add(90 * time.Second)) })
	for {
		var m map[string]any
		if err = c.ReadJSON(&m); err != nil {
			break
		}
		c.SetReadDeadline(time.Now().Add(90 * time.Second))
		switch m["type"] {
		case "snapshot", "session.status":
			if snapshotErr := x.s.Snapshot(adapter.AdapterID, m); snapshotErr == nil {
				if sessions, ok := m["sessions"].([]any); ok {
					for _, value := range sessions {
						payload, _ := value.(map[string]any)
						sid, _ := payload["sessionId"].(string)
						item, actions, sessionErr := x.s.Session(adapter.AdapterID, sid)
						if sessionErr == nil {
							message := map[string]any{"type": "session", "adapterId": adapter.AdapterID, "sessionId": sid, "session": sessionDTO(*item, actions)}
							x.h.Broadcast(adapter.AdapterID+":"+sid, message)
							x.h.Broadcast("*", message)
						}
					}
				}
			}
		case "event":
			_ = x.s.Event(adapter.AdapterID, m)
		case "action.pending":
			_ = x.s.ActionPending(adapter.AdapterID, m)
		case "action.resolved":
			x.s.ActionResolved(adapter.AdapterID, m)
		case "command.result":
			requestID, _ := m["requestId"].(string)
			ok, _ := m["ok"].(bool)
			if !x.h.Resolve(requestID, ok, m["result"], m["error"]) {
				x.s.CommandResult(requestID, ok, m["result"], m["error"])
			}
		case "ping":
			_ = write(map[string]string{"type": "pong"})
		}
	}
	if x.h.RemoveAdapter(adapter.AdapterID, out) {
		log.Printf("automation adapter disconnected adapter=%s remote=%s", adapter.AdapterID, r.RemoteAddr)
		_ = x.s.Offline(adapter.AdapterID)
	} else {
		log.Printf("automation adapter connection replaced adapter=%s remote=%s", adapter.AdapterID, r.RemoteAddr)
	}
}

func (x *AutomationHandler) BrowserWS(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sid := r.URL.Query().Get("sessionId")
	if sid == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}
	u := upgrader
	u.CheckOrigin = x.checkOrigin
	c, err := u.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	adapterID := r.URL.Query().Get("adapterId")
	key := "*"
	if sid != "*" {
		key = adapterID + ":" + sid
	}
	log.Printf("automation browser connected key=%s adapter=%s session=%s remote=%s", key, adapterID, sid, r.RemoteAddr)
	ch := x.h.Subscribe(key)
	defer func() {
		x.h.Unsubscribe(key, ch)
		log.Printf("automation browser disconnected key=%s remote=%s", key, r.RemoteAddr)
	}()
	for b := range ch {
		if err = c.WriteMessage(websocket.TextMessage, b); err != nil {
			log.Printf("automation browser write failed key=%s err=%v", key, err)
			return
		}
	}
}

type automationProjectDTO struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type automationActionDTO struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	ToolName    string    `json:"toolName,omitempty"`
	Input       any       `json:"input,omitempty"`
	Plan        string    `json:"plan,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type automationSessionDTO struct {
	AdapterID            string                `json:"adapterId"`
	ID                   string                `json:"id"`
	Project              automationProjectDTO  `json:"project"`
	Title                string                `json:"title,omitempty"`
	Prompt               string                `json:"prompt,omitempty"`
	Status               string                `json:"status"`
	Model                string                `json:"model,omitempty"`
	LastAssistantMessage string                `json:"lastAssistantMessage,omitempty"`
	CreatedAt            time.Time             `json:"createdAt"`
	UpdatedAt            time.Time             `json:"updatedAt"`
	Cwd                  string                `json:"cwd,omitempty"`
	PendingAction        *automationActionDTO  `json:"pendingAction,omitempty"`
	PendingActions       []automationActionDTO `json:"pendingActions,omitempty"`
}

type automationEventDTO struct {
	ID        string    `json:"id"`
	AdapterID string    `json:"adapterId"`
	SessionID string    `json:"sessionId"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Data      any       `json:"data,omitempty"`
}

func stringValue(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key].(string); ok {
			return value
		}
	}
	return ""
}

func sessionDTO(s model.AutomationSession, actions []model.PendingAction) automationSessionDTO {
	payload, _ := s.Payload.(map[string]any)
	path := stringValue(payload, "projectPath", "cwd")
	name := stringValue(payload, "projectName")
	if name == "" {
		name = s.ProjectID
	}
	status := s.Status
	if len(actions) > 0 {
		status = "waiting"
	} else if status == "" {
		status = "completed"
	}
	createdAt := s.UpdatedAt
	if raw := stringValue(payload, "createdAt"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			createdAt = parsed
		}
	}
	updatedAt := s.UpdatedAt
	if raw := stringValue(payload, "updatedAt"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			updatedAt = parsed
		}
	}
	dto := automationSessionDTO{AdapterID: s.AdapterID, ID: s.SessionID, Project: automationProjectDTO{ID: s.ProjectID, Name: name, Path: path}, Title: stringValue(payload, "title"), Prompt: stringValue(payload, "prompt"), Status: status, Model: stringValue(payload, "model"), LastAssistantMessage: stringValue(payload, "lastAssistantMessage"), CreatedAt: createdAt, UpdatedAt: updatedAt, Cwd: path}
	for _, action := range actions {
		p, _ := action.Payload.(map[string]any)
		dto.PendingActions = append(dto.PendingActions, automationActionDTO{ID: action.ActionID, Type: action.Type, Title: stringValue(p, "title"), Description: stringValue(p, "description"), ToolName: stringValue(p, "toolName"), Input: p["input"], Plan: stringValue(p, "plan"), CreatedAt: action.CreatedAt})
	}
	if len(dto.PendingActions) > 0 {
		dto.PendingAction = &dto.PendingActions[0]
	}
	return dto
}

func (x *AutomationHandler) Sessions(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		WriteError(w, http.StatusUnauthorized, ErrCodeBadRequest, "unauthorized")
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	items, err := x.s.Sessions(after, r.URL.Query().Get("adapterId"), r.URL.Query().Get("sessionId"))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	out := make([]automationSessionDTO, 0, len(items))
	for _, item := range items {
		_, actions, actionErr := x.s.Session(item.AdapterID, item.SessionID)
		if actionErr != nil {
			actions = nil
		}
		out = append(out, sessionDTO(item, actions))
	}
	WriteList(w, out, int64(len(out)))
}

func (x *AutomationHandler) Session(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		WriteError(w, http.StatusUnauthorized, ErrCodeBadRequest, "unauthorized")
		return
	}
	adapterID := mux.Vars(r)["adapterId"]
	sessionID := mux.Vars(r)["sessionId"]
	item, actions, err := x.s.Session(adapterID, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		WriteError(w, http.StatusNotFound, ErrCodeBadRequest, "session not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, sessionDTO(*item, actions))
}

func (x *AutomationHandler) Message(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		WriteError(w, http.StatusUnauthorized, ErrCodeBadRequest, "unauthorized")
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	if decoder.Decode(&body) != nil {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid message")
		return
	}
	body.Content = strings.TrimSpace(body.Content)
	if body.Content == "" || len(body.Content) > 32000 {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "content must be 1-32000 characters")
		return
	}
	requestID := fmt.Sprintf("message-%d", time.Now().UnixNano())
	sid := mux.Vars(r)["sessionId"]
	if err := x.s.SendMessage(mux.Vars(r)["adapterId"], sid, requestID, body.Content); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			WriteError(w, http.StatusNotFound, ErrCodeBadRequest, "session not found")
		} else {
			WriteError(w, http.StatusConflict, ErrCodeBadRequest, err.Error())
		}
		return
	}
	x.h.Broadcast(mux.Vars(r)["adapterId"]+":"+sid, map[string]any{"type": "session.message", "requestId": requestID, "sessionId": sid, "content": body.Content})
	WriteSuccess(w, http.StatusAccepted, map[string]any{"requestId": requestID, "accepted": true})
}

func (x *AutomationHandler) Operations(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		WriteError(w, http.StatusUnauthorized, ErrCodeBadRequest, "unauthorized")
		return
	}
	items, err := x.s.Operations(mux.Vars(r)["adapterId"], mux.Vars(r)["sessionId"])
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, items)
}

func (x *AutomationHandler) Events(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		WriteError(w, http.StatusUnauthorized, ErrCodeBadRequest, "unauthorized")
		return
	}
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if limit <= 0 {
		limit = 50
	}
	items, hasMore, err := x.s.Events(mux.Vars(r)["adapterId"], mux.Vars(r)["sessionId"], after, before, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	out := make([]automationEventDTO, 0, len(items))
	for _, item := range items {
		out = append(out, automationEventDTO{ID: item.EventID, AdapterID: item.AdapterID, SessionID: item.SessionID, Type: item.Type, CreatedAt: item.CreatedAt, Data: item.Payload})
	}
	var nextBefore int64
	if len(items) > 0 {
		nextBefore = items[0].ID
	}
	WriteSuccess(w, http.StatusOK, map[string]any{"items": out, "total": len(out), "hasMore": hasMore, "nextBefore": nextBefore})
}

func (x *AutomationHandler) Respond(w http.ResponseWriter, r *http.Request) {
	if !x.authorized(r) {
		WriteError(w, http.StatusUnauthorized, ErrCodeBadRequest, "unauthorized")
		return
	}
	var body struct {
		RequestID string `json:"requestId"`
		Decision  string `json:"decision"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body) != nil || (body.Decision != "approve" && body.Decision != "reject" && body.Decision != "continue") {
		WriteError(w, http.StatusBadRequest, ErrCodeBadRequest, "invalid decision")
		return
	}
	vars := mux.Vars(r)
	requestID := fmt.Sprintf("action-%s-%d", vars["actionId"], time.Now().UnixNano())
	if err := x.s.DispatchResponse(vars["adapterId"], vars["sessionId"], vars["actionId"], requestID, body.Decision); err != nil {
		status := http.StatusConflict
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		WriteError(w, status, ErrCodeBadRequest, err.Error())
		return
	}
	item, actions, err := x.s.Session(vars["adapterId"], vars["sessionId"])
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		return
	}
	WriteSuccess(w, http.StatusAccepted, sessionDTO(*item, actions))
}
