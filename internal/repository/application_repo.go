package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"local-dev-workspace/internal/model"
)

type ApplicationRepo struct{ db *sql.DB }

func NewApplicationRepo() *ApplicationRepo { return &ApplicationRepo{db: DB} }

// --- helpers ---

func marshalCommands(commands []model.CommandDef) string {
	if len(commands) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(commands)
	return string(data)
}

func unmarshalCommands(raw string) []model.CommandDef {
	var cmds []model.CommandDef
	if raw == "" || raw == "[]" {
		return []model.CommandDef{}
	}
	if err := json.Unmarshal([]byte(raw), &cmds); err != nil {
		// Backward compat: old single-string command → wrap as default CommandDef
		return []model.CommandDef{{Name: "default", Command: raw}}
	}
	if cmds == nil {
		return []model.CommandDef{}
	}
	return cmds
}

// --- Applications ---

func (r *ApplicationRepo) List() ([]model.Application, error) {
	rows, err := r.db.Query(`SELECT id, name, description, commands, working_directory, environment, auto_start, enabled, created_at, updated_at FROM applications ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Application
	for rows.Next() {
		var a model.Application
		var autoStart, enabled int
		var commandsRaw string
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &commandsRaw, &a.WorkingDirectory, &a.Environment, &autoStart, &enabled, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Commands = unmarshalCommands(commandsRaw)
		a.AutoStart = autoStart == 1
		a.Enabled = enabled == 1
		items = append(items, a)
	}
	if items == nil {
		items = []model.Application{}
	}
	return items, rows.Err()
}

func (r *ApplicationRepo) GetByID(id int64) (*model.Application, error) {
	var a model.Application
	var autoStart, enabled int
	var commandsRaw string
	err := r.db.QueryRow(`SELECT id, name, description, commands, working_directory, environment, auto_start, enabled, created_at, updated_at FROM applications WHERE id=?`, id).
		Scan(&a.ID, &a.Name, &a.Description, &commandsRaw, &a.WorkingDirectory, &a.Environment, &autoStart, &enabled, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Commands = unmarshalCommands(commandsRaw)
	a.AutoStart = autoStart == 1
	a.Enabled = enabled == 1
	return &a, nil
}

func (r *ApplicationRepo) Create(a *model.Application) error {
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	autoStart := 0
	if a.AutoStart {
		autoStart = 1
	}
	enabled := 0
	if a.Enabled {
		enabled = 1
	}
	commandsJSON := marshalCommands(a.Commands)
	res, err := r.db.Exec(`INSERT INTO applications (name, description, commands, working_directory, environment, auto_start, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Name, a.Description, commandsJSON, a.WorkingDirectory, a.Environment, autoStart, enabled, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

func (r *ApplicationRepo) Update(a *model.Application) error {
	a.UpdatedAt = time.Now()
	autoStart := 0
	if a.AutoStart {
		autoStart = 1
	}
	enabled := 0
	if a.Enabled {
		enabled = 1
	}
	commandsJSON := marshalCommands(a.Commands)
	_, err := r.db.Exec(`UPDATE applications SET name=?, description=?, commands=?, working_directory=?, environment=?, auto_start=?, enabled=?, updated_at=? WHERE id=?`,
		a.Name, a.Description, commandsJSON, a.WorkingDirectory, a.Environment, autoStart, enabled, a.UpdatedAt, a.ID)
	return err
}

func (r *ApplicationRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM applications WHERE id=?`, id)
	return err
}

// --- Application Runs ---

func (r *ApplicationRepo) CreateRun(run *model.ApplicationRun) error {
	now := time.Now()
	run.CreatedAt = now
	res, err := r.db.Exec(`INSERT INTO application_runs (application_id, pid, command_name, started_at, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		run.ApplicationID, run.PID, run.CommandName, run.StartedAt, run.Status, run.CreatedAt)
	if err != nil {
		return err
	}
	run.ID, _ = res.LastInsertId()
	return nil
}

func (r *ApplicationRepo) GetLatestRun(appID int64) (*model.ApplicationRun, error) {
	var run model.ApplicationRun
	err := r.db.QueryRow(`SELECT id, application_id, pid, command_name, started_at, stopped_at, exit_code, status, created_at FROM application_runs WHERE application_id=? ORDER BY id DESC LIMIT 1`, appID).
		Scan(&run.ID, &run.ApplicationID, &run.PID, &run.CommandName, &run.StartedAt, &run.StoppedAt, &run.ExitCode, &run.Status, &run.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *ApplicationRepo) UpdateRun(run *model.ApplicationRun) error {
	_, err := r.db.Exec(`UPDATE application_runs SET pid=?, command_name=?, started_at=?, stopped_at=?, exit_code=?, status=? WHERE id=?`,
		run.PID, run.CommandName, run.StartedAt, run.StoppedAt, run.ExitCode, run.Status, run.ID)
	return err
}

func (r *ApplicationRepo) ListRunningRuns() ([]model.ApplicationRun, error) {
	rows, err := r.db.Query(`SELECT id, application_id, pid, command_name, started_at, stopped_at, exit_code, status, created_at FROM application_runs WHERE status IN ('STARTING','RUNNING')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.ApplicationRun
	for rows.Next() {
		var run model.ApplicationRun
		if err := rows.Scan(&run.ID, &run.ApplicationID, &run.PID, &run.CommandName, &run.StartedAt, &run.StoppedAt, &run.ExitCode, &run.Status, &run.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, run)
	}
	if items == nil {
		items = []model.ApplicationRun{}
	}
	return items, rows.Err()
}
