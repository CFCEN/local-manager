package repository

import (
	"database/sql"
	"encoding/json"
	"local-dev-workspace/internal/model"
	"time"
)

type AutomationRepo struct{ db *sql.DB }

func NewAutomationRepo() *AutomationRepo { return &AutomationRepo{DB} }
func enc(v any) string                   { b, _ := json.Marshal(v); return string(b) }
func dec(s string) any                   { var v any; _ = json.Unmarshal([]byte(s), &v); return v }
func (r *AutomationRepo) UpsertAdapter(a *model.Adapter) error {
	now := time.Now()
	a.LastSeen = now
	a.UpdatedAt = now
	_, e := r.db.Exec(`INSERT INTO automation_adapters(adapter_id,name,version,online,last_seen,created_at,updated_at) VALUES(?,?,?,1,?,?,?) ON CONFLICT(adapter_id) DO UPDATE SET name=excluded.name,version=excluded.version,online=1,last_seen=excluded.last_seen,updated_at=excluded.updated_at`, a.AdapterID, a.Name, a.Version, now, now, now)
	return e
}
func (r *AutomationRepo) SetAdapterOffline(id string) error {
	_, e := r.db.Exec(`UPDATE automation_adapters SET online=0,updated_at=? WHERE adapter_id=?`, time.Now(), id)
	return e
}
func (r *AutomationRepo) UpsertProject(a, pid, name, path string, payload any) error {
	_, e := r.db.Exec(`INSERT INTO automation_projects(adapter_id,project_id,name,path,payload,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(adapter_id,project_id) DO UPDATE SET name=excluded.name,path=excluded.path,payload=excluded.payload,updated_at=excluded.updated_at`, a, pid, name, path, enc(payload), time.Now())
	return e
}
func (r *AutomationRepo) UpsertSession(a, p, sid, status string, payload any) error {
	_, e := r.db.Exec(`INSERT INTO automation_sessions(adapter_id,project_id,session_id,status,payload,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(adapter_id,session_id) DO UPDATE SET project_id=excluded.project_id,status=excluded.status,payload=excluded.payload,updated_at=excluded.updated_at`, a, p, sid, status, enc(payload), time.Now())
	return e
}
func (r *AutomationRepo) SetCursor(a, sid, c string) error {
	_, e := r.db.Exec(`UPDATE automation_sessions SET cursor=? WHERE adapter_id=? AND session_id=?`, c, a, sid)
	return e
}
func (r *AutomationRepo) AddEvent(e *model.AutomationEvent) error {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO automation_events(adapter_id,project_id,session_id,event_id,type,payload,created_at) VALUES(?,?,?,?,?,?,?)`, e.AdapterID, e.ProjectID, e.SessionID, e.EventID, e.Type, enc(e.Payload), e.CreatedAt)
	if err == nil && e.Cursor != "" {
		err = r.SetCursor(e.AdapterID, e.SessionID, e.Cursor)
	}
	return err
}
func (r *AutomationRepo) ListSessions(after int64, adapter, sid string) ([]model.AutomationSession, error) {
	q := `SELECT id,adapter_id,project_id,session_id,status,payload,cursor,updated_at FROM automation_sessions WHERE id>?`
	args := []any{after}
	if adapter != "" {
		q += ` AND adapter_id=?`
		args = append(args, adapter)
	}
	if sid != "" {
		q += ` AND session_id=?`
		args = append(args, sid)
	}
	q += ` ORDER BY id`
	rows, e := r.db.Query(q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []model.AutomationSession{}
	for rows.Next() {
		var x model.AutomationSession
		var p string
		if e = rows.Scan(&x.ID, &x.AdapterID, &x.ProjectID, &x.SessionID, &x.Status, &p, &x.Cursor, &x.UpdatedAt); e != nil {
			return nil, e
		}
		x.Payload = dec(p)
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *AutomationRepo) GetSession(a, sid string) (*model.AutomationSession, error) {
	var x model.AutomationSession
	var p string
	e := r.db.QueryRow(`SELECT id,adapter_id,project_id,session_id,status,payload,cursor,updated_at FROM automation_sessions WHERE adapter_id=? AND session_id=?`, a, sid).Scan(&x.ID, &x.AdapterID, &x.ProjectID, &x.SessionID, &x.Status, &p, &x.Cursor, &x.UpdatedAt)
	if e != nil {
		return nil, e
	}
	x.Payload = dec(p)
	return &x, nil
}
func (r *AutomationRepo) HasEvents(a, sid string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM automation_events WHERE adapter_id=? AND session_id=? AND type IN ('user','assistant'))`, a, sid).Scan(&count)
	return count == 1, err
}

func (r *AutomationRepo) ListEvents(a, sid string, after, before, limit int64) ([]model.AutomationEvent, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	args := []any{a, sid, after}
	q := `SELECT id,adapter_id,project_id,session_id,event_id,type,payload,created_at FROM automation_events WHERE adapter_id=? AND session_id=? AND type IN ('user','assistant') AND id>?`
	if before > 0 {
		q += ` AND id<?`
		args = append(args, before)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, e := r.db.Query(q, args...)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []model.AutomationEvent{}
	for rows.Next() {
		var x model.AutomationEvent
		var p string
		if e = rows.Scan(&x.ID, &x.AdapterID, &x.ProjectID, &x.SessionID, &x.EventID, &x.Type, &p, &x.CreatedAt); e != nil {
			return nil, false, e
		}
		x.Payload = dec(p)
		out = append(out, x)
	}
	if e = rows.Err(); e != nil {
		return nil, false, e
	}
	hasMore := len(out) > int(limit)
	if hasMore {
		out = out[:limit]
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out, hasMore, nil
}
func (r *AutomationRepo) AddAction(a *model.PendingAction) (bool, error) {
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	res, e := r.db.Exec(`INSERT INTO automation_actions(adapter_id,project_id,session_id,action_id,type,payload,status,result,created_at,updated_at) VALUES(?,?,?,?,?,?,?, '{}',?,?) ON CONFLICT(adapter_id,action_id) DO NOTHING`, a.AdapterID, a.ProjectID, a.SessionID, a.ActionID, a.Type, enc(a.Payload), a.Status, now, now)
	if e != nil {
		return false, e
	}
	n, e := res.RowsAffected()
	return n == 1, e
}
func (r *AutomationRepo) ListPendingActions(a, sid string) ([]model.PendingAction, error) {
	rows, e := r.db.Query(`SELECT id,adapter_id,project_id,session_id,action_id,type,payload,status,result,created_at,updated_at FROM automation_actions WHERE adapter_id=? AND session_id=? AND status IN ('pending','responding') ORDER BY id`, a, sid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []model.PendingAction{}
	for rows.Next() {
		var x model.PendingAction
		var p, z string
		if e = rows.Scan(&x.ID, &x.AdapterID, &x.ProjectID, &x.SessionID, &x.ActionID, &x.Type, &p, &x.Status, &z, &x.CreatedAt, &x.UpdatedAt); e != nil {
			return nil, e
		}
		x.Payload, x.Result = dec(p), dec(z)
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *AutomationRepo) GetPendingAction(a, sid, id string) (*model.PendingAction, error) {
	var x model.PendingAction
	var p, z string
	e := r.db.QueryRow(`SELECT id,adapter_id,project_id,session_id,action_id,type,payload,status,result,created_at,updated_at FROM automation_actions WHERE adapter_id=? AND session_id=? AND action_id=? AND status='pending'`, a, sid, id).Scan(&x.ID, &x.AdapterID, &x.ProjectID, &x.SessionID, &x.ActionID, &x.Type, &p, &x.Status, &z, &x.CreatedAt, &x.UpdatedAt)
	if e != nil {
		return nil, e
	}
	x.Payload, x.Result = dec(p), dec(z)
	return &x, nil
}
func (r *AutomationRepo) ClaimAction(a, sid, id string) error {
	res, e := r.db.Exec(`UPDATE automation_actions SET status='responding',updated_at=? WHERE adapter_id=? AND session_id=? AND action_id=? AND status='pending'`, time.Now(), a, sid, id)
	if e == nil {
		n, _ := res.RowsAffected()
		if n == 0 {
			e = sql.ErrNoRows
		}
	}
	return e
}
func (r *AutomationRepo) RespondAction(a, sid, id, status string, result any) error {
	res, e := r.db.Exec(`UPDATE automation_actions SET status=?,result=?,updated_at=? WHERE adapter_id=? AND session_id=? AND action_id=? AND status='responding'`, status, enc(result), time.Now(), a, sid, id)
	if e == nil {
		n, _ := res.RowsAffected()
		if n == 0 {
			e = sql.ErrNoRows
		}
	}
	return e
}
func (r *AutomationRepo) ResolveAction(a, sid, id, status string, result any) error {
	_, e := r.db.Exec(`UPDATE automation_actions SET status=?,result=?,updated_at=? WHERE adapter_id=? AND session_id=? AND action_id=? AND status IN ('pending','responding')`, status, enc(result), time.Now(), a, sid, id)
	return e
}
func (r *AutomationRepo) FailAdapter(a string) error {
	_, e := r.db.Exec(`UPDATE automation_actions SET status='failed',result=?,updated_at=? WHERE adapter_id=? AND status IN ('pending','responding')`, enc(map[string]string{"error": "adapter offline"}), time.Now(), a)
	return e
}
func (r *AutomationRepo) CreateOperation(o *model.Operation) error {
	_, e := r.db.Exec(`INSERT INTO automation_operations(request_id,adapter_id,session_id,status,error,result,created_at,updated_at) VALUES(?,?,?,'accepted','{}','{}',?,?)`, o.RequestID, o.AdapterID, o.SessionID, o.CreatedAt, o.UpdatedAt)
	return e
}
func (r *AutomationRepo) UpdateOperation(id, status string, result, er any) error {
	_, e := r.db.Exec(`UPDATE automation_operations SET status=?,result=?,error=?,updated_at=? WHERE request_id=?`, status, enc(result), enc(er), time.Now(), id)
	return e
}
func (r *AutomationRepo) ListOperations(a, sid string) ([]model.Operation, error) {
	rows, e := r.db.Query(`SELECT id,request_id,adapter_id,session_id,status,error,result,created_at,updated_at FROM automation_operations WHERE adapter_id=? AND session_id=? ORDER BY id DESC LIMIT 50`, a, sid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []model.Operation{}
	for rows.Next() {
		var o model.Operation
		var er, re string
		if e = rows.Scan(&o.ID, &o.RequestID, &o.AdapterID, &o.SessionID, &o.Status, &er, &re, &o.CreatedAt, &o.UpdatedAt); e != nil {
			return nil, e
		}
		o.Error, o.Result = dec(er), dec(re)
		out = append(out, o)
	}
	return out, rows.Err()
}
