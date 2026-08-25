package repository

import (
	"database/sql"
	"time"

	"local-dev-workspace/internal/model"
)

type LogRepo struct{ db *sql.DB }

func NewLogRepo() *LogRepo { return &LogRepo{db: DB} }

// --- Log Groups ---

func (r *LogRepo) ListGroups() ([]model.LogGroup, error) {
	rows, err := r.db.Query(`SELECT g.id, g.name, g.description, g.created_at, g.updated_at, COUNT(f.id) AS file_count
		FROM log_groups g LEFT JOIN log_files f ON f.group_id = g.id
		GROUP BY g.id ORDER BY g.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.LogGroup
	for rows.Next() {
		var g model.LogGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt, &g.FileCount); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	if items == nil {
		items = []model.LogGroup{}
	}
	return items, rows.Err()
}

func (r *LogRepo) GetGroupByID(id int64) (*model.LogGroup, error) {
	var g model.LogGroup
	err := r.db.QueryRow(`SELECT id, name, description, created_at, updated_at FROM log_groups WHERE id=?`, id).
		Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &g, err
}

func (r *LogRepo) CreateGroup(g *model.LogGroup) error {
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now
	res, err := r.db.Exec(`INSERT INTO log_groups (name, description, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		g.Name, g.Description, g.CreatedAt, g.UpdatedAt)
	if err != nil {
		return err
	}
	g.ID, _ = res.LastInsertId()
	return nil
}

func (r *LogRepo) UpdateGroup(g *model.LogGroup) error {
	g.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE log_groups SET name=?, description=?, updated_at=? WHERE id=?`, g.Name, g.Description, g.UpdatedAt, g.ID)
	return err
}

func (r *LogRepo) DeleteGroup(id int64) error {
	_, err := r.db.Exec(`DELETE FROM log_groups WHERE id=?`, id)
	return err
}

// --- Log Files ---

func (r *LogRepo) ListFiles() ([]model.LogFile, error) {
	rows, err := r.db.Query(`SELECT f.id, f.name, f.path, f.group_id, COALESCE(g.name, '') AS group_name, f.description, f.created_at, f.updated_at
		FROM log_files f LEFT JOIN log_groups g ON g.id = f.group_id
		ORDER BY f.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.LogFile
	for rows.Next() {
		var f model.LogFile
		if err := rows.Scan(&f.ID, &f.Name, &f.Path, &f.GroupID, &f.GroupName, &f.Description, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, f)
	}
	if items == nil {
		items = []model.LogFile{}
	}
	return items, rows.Err()
}

func (r *LogRepo) GetFileByID(id int64) (*model.LogFile, error) {
	var f model.LogFile
	err := r.db.QueryRow(`SELECT id, name, path, group_id, description, created_at, updated_at FROM log_files WHERE id=?`, id).
		Scan(&f.ID, &f.Name, &f.Path, &f.GroupID, &f.Description, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

func (r *LogRepo) CreateFile(f *model.LogFile) error {
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
	res, err := r.db.Exec(`INSERT INTO log_files (name, path, group_id, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		f.Name, f.Path, f.GroupID, f.Description, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		return err
	}
	f.ID, _ = res.LastInsertId()
	return nil
}

func (r *LogRepo) UpdateFile(f *model.LogFile) error {
	f.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE log_files SET name=?, path=?, group_id=?, description=?, updated_at=? WHERE id=?`,
		f.Name, f.Path, f.GroupID, f.Description, f.UpdatedAt, f.ID)
	return err
}

func (r *LogRepo) DeleteFile(id int64) error {
	_, err := r.db.Exec(`DELETE FROM log_files WHERE id=?`, id)
	return err
}
