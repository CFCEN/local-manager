package service

import (
	"encoding/json"
	"fmt"
	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/repository"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type commandResult struct {
	OK            bool
	Result, Error any
}
type AutomationHub struct {
	mu       sync.RWMutex
	adapters map[string]chan []byte
	browsers map[string]map[chan []byte]struct{}
	pending  map[string]chan commandResult
}

func NewAutomationHub() *AutomationHub {
	return &AutomationHub{adapters: map[string]chan []byte{}, browsers: map[string]map[chan []byte]struct{}{}, pending: map[string]chan commandResult{}}
}
func (h *AutomationHub) AddAdapter(id string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := make(chan []byte, 16)
	h.adapters[id] = c
	log.Printf("automation adapter channel registered adapter=%s", id)
	return c
}
func (h *AutomationHub) RemoveAdapter(id string, c chan []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.adapters[id] == c {
		delete(h.adapters, id)
		return true
	}
	return false
}
func (h *AutomationHub) Command(id string, v any) error {
	b, _ := json.Marshal(v)
	log.Printf("automation command adapter=%s bytes=%d", id, len(b))
	h.mu.RLock()
	c := h.adapters[id]
	h.mu.RUnlock()
	if c == nil {
		return fmt.Errorf("adapter offline")
	}
	select {
	case c <- b:
		return nil
	default:
		return fmt.Errorf("adapter busy")
	}
}
func (h *AutomationHub) Request(a, id, t string, p any) (any, error) {
	ch := make(chan commandResult, 1)
	h.mu.Lock()
	if _, ok := h.pending[id]; ok {
		h.mu.Unlock()
		return nil, fmt.Errorf("request already pending")
	}
	h.pending[id] = ch
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.pending, id); h.mu.Unlock() }()
	if e := h.Command(a, map[string]any{"type": "command", "payload": map[string]any{"requestId": id, "type": t, "payload": p}}); e != nil {
		return nil, e
	}
	select {
	case r := <-ch:
		if !r.OK {
			return nil, fmt.Errorf("adapter command failed: %v", r.Error)
		}
		return r.Result, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("adapter command timeout")
	}
}
func (h *AutomationHub) Resolve(id string, ok bool, r, e any) bool {
	h.mu.RLock()
	c := h.pending[id]
	h.mu.RUnlock()
	if c == nil {
		return false
	}
	select {
	case c <- commandResult{ok, r, e}:
		return true
	default:
		return false
	}
}
func (h *AutomationHub) Subscribe(s string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := make(chan []byte, 16)
	if h.browsers[s] == nil {
		h.browsers[s] = map[chan []byte]struct{}{}
	}
	h.browsers[s][c] = struct{}{}
	return c
}
func (h *AutomationHub) Unsubscribe(s string, c chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.browsers[s], c)
	if len(h.browsers[s]) == 0 {
		delete(h.browsers, s)
	}
}
func (h *AutomationHub) Broadcast(s string, v any) {
	b, _ := json.Marshal(v)
	h.mu.RLock()
	defer h.mu.RUnlock()
	sent, dropped := 0, 0
	for c := range h.browsers[s] {
		select {
		case c <- b:
			sent++
		default:
			dropped++
		}
	}
	if sent > 0 || dropped > 0 {
		log.Printf("automation broadcast key=%s bytes=%d subscribers=%d sent=%d dropped=%d", s, len(b), sent+dropped, sent, dropped)
	}
}

type AutomationService struct {
	repo     *repository.AutomationRepo
	hub      *AutomationHub
	mu       sync.Mutex
	commands map[string]commandRef
}

type commandRef struct{ adapterID, sessionID, actionID, operationID string }

func NewAutomationService(r *repository.AutomationRepo, h *AutomationHub) *AutomationService {
	return &AutomationService{repo: r, hub: h, commands: map[string]commandRef{}}
}
func (s *AutomationService) Hello(a model.Adapter) error { return s.repo.UpsertAdapter(&a) }
func (s *AutomationService) Offline(a string) error {
	_ = s.repo.FailAdapter(a)
	return s.repo.SetAdapterOffline(a)
}
func (s *AutomationService) Snapshot(a string, m map[string]any) error {
	xs, _ := m["sessions"].([]any)
	for _, v := range xs {
		p, _ := v.(map[string]any)
		sid, _ := p["sessionId"].(string)
		if sid == "" {
			continue
		}
		pid, _ := p["projectId"].(string)
		if pid == "" {
			pid, _ = p["projectPath"].(string)
		}
		status, _ := p["status"].(string)
		if e := s.repo.UpsertSession(a, pid, sid, status, p); e != nil {
			return e
		}
	}
	return nil
}

func (s *AutomationService) broadcastSession(a, sid string) {
	session, actions, err := s.Session(a, sid)
	if err != nil {
		return
	}
	payload, _ := session.Payload.(map[string]any)
	projectPath := stringValue(payload, "projectPath", "cwd")
	projectName := stringValue(payload, "projectName")
	if projectName == "" {
		projectName = session.ProjectID
	}
	status := session.Status
	if len(actions) > 0 {
		status = "waiting"
	}
	item := map[string]any{
		"adapterId":            a,
		"id":                   sid,
		"project":              map[string]any{"id": session.ProjectID, "name": projectName, "path": projectPath},
		"title":                stringValue(payload, "title"),
		"prompt":               stringValue(payload, "prompt"),
		"status":               status,
		"model":                stringValue(payload, "model"),
		"lastAssistantMessage": stringValue(payload, "lastAssistantMessage"),
		"updatedAt":            session.UpdatedAt,
		"pendingActions":       actions,
	}
	if len(actions) > 0 {
		item["pendingAction"] = actions[0]
	}
	message := map[string]any{"type": "session", "adapterId": a, "sessionId": sid, "session": item}
	s.hub.Broadcast(a+":"+sid, message)
	s.hub.Broadcast("*", message)
}

func stringValue(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok {
			return value
		}
	}
	return ""
}
func (s *AutomationService) SendMessage(a, sid, id, content string) error {
	now := time.Now()
	if err := s.repo.CreateOperation(&model.Operation{RequestID: id, AdapterID: a, SessionID: sid, CreatedAt: now, UpdatedAt: now}); err != nil {
		return err
	}
	s.mu.Lock()
	s.commands[id] = commandRef{adapterID: a, sessionID: sid, operationID: id}
	s.mu.Unlock()
	if err := s.hub.Command(a, map[string]any{"type": "command", "payload": map[string]any{"requestId": id, "type": "session.message", "payload": map[string]any{"sessionId": sid, "content": content}}}); err != nil {
		_ = s.repo.UpdateOperation(id, "failed", nil, err.Error())
		s.mu.Lock()
		delete(s.commands, id)
		s.mu.Unlock()
		return err
	}
	_ = s.repo.UpdateOperation(id, "running", nil, nil)
	return nil
}
func (s *AutomationService) ActionPending(a string, m map[string]any) error {
	sid, _ := m["sessionId"].(string)
	id, _ := m["actionId"].(string)
	typ, _ := m["actionType"].(string)
	ss, e := s.repo.GetSession(a, sid)
	if e != nil {
		return e
	}
	x := &model.PendingAction{AdapterID: a, ProjectID: ss.ProjectID, SessionID: sid, ActionID: id, RequestID: fmt.Sprint(m["requestId"]), Type: typ, Payload: m, Status: "pending"}
	ok, e := s.repo.AddAction(x)
	if e == nil && ok {
		s.broadcastSession(a, sid)
	}
	return e
}
func (s *AutomationService) Event(a string, m map[string]any) error {
	sid, _ := m["sessionId"].(string)
	log.Printf("automation event received adapter=%s session=%s event=%v type=%v", a, sid, m["eventId"], m["eventType"])
	typ, _ := m["eventType"].(string)
	payload, _ := m["payload"].(map[string]any)
	message, _ := payload["message"].(map[string]any)
	role, _ := message["role"].(string)
	if (typ != "user" && typ != "assistant") || (role != "user" && role != "assistant") {
		if cursor := fmt.Sprint(m["cursor"]); cursor != "" {
			return s.repo.SetCursor(a, sid, "v2:"+cursor)
		}
		return nil
	}
	id, _ := m["eventId"].(string)
	if id == "" {
		return fmt.Errorf("eventId required")
	}
	e := &model.AutomationEvent{AdapterID: a, ProjectID: fmt.Sprint(m["projectId"]), SessionID: sid, EventID: id, Type: typ, Cursor: fmt.Sprint(m["cursor"]), Payload: payload, CreatedAt: time.Now()}
	if err := s.repo.AddEvent(e); err != nil {
		return err
	}
	s.hub.Broadcast(a+":"+sid, map[string]any{
		"type": "event", "adapterId": a, "sessionId": sid,
		"event": map[string]any{"id": id, "adapterId": a, "sessionId": sid, "type": typ, "createdAt": e.CreatedAt, "data": e.Payload},
	})
	s.hub.Broadcast("*", map[string]any{
		"type": "event", "adapterId": a, "sessionId": sid,
		"event": map[string]any{"id": id, "adapterId": a, "sessionId": sid, "type": typ, "createdAt": e.CreatedAt, "data": e.Payload},
	})
	s.broadcastSession(a, sid)
	return nil
}
func (s *AutomationService) Sessions(after int64, a, sid string) ([]model.AutomationSession, error) {
	return s.repo.ListSessions(after, a, sid)
}
func (s *AutomationService) Session(a, sid string) (*model.AutomationSession, []model.PendingAction, error) {
	x, e := s.repo.GetSession(a, sid)
	if e != nil {
		return nil, nil, e
	}
	z, e := s.repo.ListPendingActions(a, sid)
	return x, z, e
}
func (s *AutomationService) Operations(a, sid string) ([]model.Operation, error) {
	return s.repo.ListOperations(a, sid)
}

func (s *AutomationService) syncSession(a, sid string) error {
	x, err := s.repo.GetSession(a, sid)
	if err != nil {
		return err
	}
	var cursor int64
	if strings.HasPrefix(x.Cursor, "v2:") {
		cursor, err = strconv.ParseInt(strings.TrimPrefix(x.Cursor, "v2:"), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid session cursor %q: %w", x.Cursor, err)
		}
	}

	for {
		requestID := fmt.Sprintf("history-%d", time.Now().UnixNano())
		request := map[string]any{"sessionId": sid, "maxBytes": 4 * 1024 * 1024}
		if cursor > 0 {
			request["afterCursor"] = cursor
		}
		result, err := s.hub.Request(a, requestID, "sessions.read", request)
		if err != nil {
			return err
		}
		payload, ok := result.(map[string]any)
		if !ok {
			return nil
		}
		records, _ := payload["records"].([]any)
		for index, record := range records {
			item, ok := record.(map[string]any)
			if !ok {
				continue
			}
			eventID := fmt.Sprintf("%s:%d:%d", sid, cursor, index)
			if uuid, ok := item["uuid"].(string); ok && uuid != "" {
				eventID = uuid
			}
			_ = s.Event(a, map[string]any{"sessionId": sid, "eventId": eventID, "eventType": item["type"], "payload": item})
		}
		next, ok := payload["nextCursor"].(float64)
		if !ok || int64(next) <= cursor {
			return nil
		}
		cursor = int64(next)
		if err := s.repo.SetCursor(a, sid, fmt.Sprintf("v2:%d", cursor)); err != nil {
			return err
		}
	}
}

func (s *AutomationService) SyncSession(a, sid string) error {
	return s.syncSession(a, sid)
}

func (s *AutomationService) Events(a, sid string, after, before, limit int64) ([]model.AutomationEvent, bool, error) {
	if err := s.syncSession(a, sid); err != nil {
		return nil, false, err
	}
	return s.repo.ListEvents(a, sid, after, before, limit)
}
func (s *AutomationService) DispatchResponse(a, sid, id, rid, decision string) error {
	if _, e := s.repo.GetPendingAction(a, sid, id); e != nil {
		return e
	}
	if e := s.repo.ClaimAction(a, sid, id); e != nil {
		return e
	}
	s.mu.Lock()
	s.commands[rid] = commandRef{adapterID: a, sessionID: sid, actionID: id}
	s.mu.Unlock()
	if e := s.hub.Command(a, map[string]any{"type": "command", "payload": map[string]any{"requestId": rid, "type": "action.respond", "payload": map[string]any{"sessionId": sid, "actionId": id, "decision": decision}}}); e != nil {
		_ = s.repo.RespondAction(a, sid, id, "failed", e.Error())
		s.mu.Lock()
		delete(s.commands, rid)
		s.mu.Unlock()
		return e
	}
	return nil
}

func (s *AutomationService) ActionResolved(a string, m map[string]any) {
	sid, _ := m["sessionId"].(string)
	id, _ := m["actionId"].(string)
	status, _ := m["status"].(string)
	if status == "" {
		status = "failed"
	}
	_ = s.repo.ResolveAction(a, sid, id, status, m)
	s.broadcastSession(a, sid)
}

func (s *AutomationService) CommandResult(requestID string, ok bool, result, commandError any) {
	s.mu.Lock()
	ref, correlated := s.commands[requestID]
	if correlated {
		delete(s.commands, requestID)
	}
	s.mu.Unlock()
	if !correlated {
		return
	}
	if ref.operationID == "history" {
		if payload, ok := result.(map[string]any); ok {
			if records, ok := payload["records"].([]any); ok {
				for index, record := range records {
					item, ok := record.(map[string]any)
					if !ok {
						continue
					}
					eventID := fmt.Sprintf("%s:%d", ref.sessionID, index)
					if uuid, ok := item["uuid"].(string); ok && uuid != "" {
						eventID = uuid
					}
					_ = s.Event(ref.adapterID, map[string]any{"sessionId": ref.sessionID, "eventId": eventID, "eventType": item["type"], "payload": item})
				}
			}
		}
		return
	}
	if ref.actionID != "" {
		status := "completed"
		value := result
		if !ok {
			status, value = "failed", commandError
		}
		_ = s.repo.RespondAction(ref.adapterID, ref.sessionID, ref.actionID, status, value)
		return
	}
	status := "completed"
	if !ok {
		status = "failed"
	}
	_ = s.repo.UpdateOperation(requestID, status, result, commandError)
	s.hub.Broadcast(ref.adapterID+":"+ref.sessionID, map[string]any{"type": "operation", "adapterId": ref.adapterID, "sessionId": ref.sessionID, "requestId": requestID, "status": status, "result": result, "error": commandError})
	s.hub.Broadcast("*", map[string]any{"type": "operation", "adapterId": ref.adapterID, "sessionId": ref.sessionID, "requestId": requestID, "status": status, "result": result, "error": commandError})
}
func (s *AutomationService) CompleteResponse(a, sid, id, status string, r any) error {
	return s.repo.RespondAction(a, sid, id, status, r)
}
