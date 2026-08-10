package repository

import (
	"database/sql"
	"time"

	"local-dev-workspace/internal/model"
)

type ConfigRepo struct{ db *sql.DB }

func NewConfigRepo() *ConfigRepo { return &ConfigRepo{db: DB} }

func (r *ConfigRepo) List() ([]model.ConfigFile, error) {
	rows, err := r.db.Query(`SELECT id, name, path, description, group_name, created_at, updated_at FROM config_files ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ConfigFile
	for rows.Next() {
		var cf model.ConfigFile
		if err := rows.Scan(&cf.ID, &cf.Name, &cf.Path, &cf.Description, &cf.GroupName, &cf.CreatedAt, &cf.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, cf)
	}
	if items == nil {
		items = []model.ConfigFile{}
	}
	return items, rows.Err()
}

func (r *ConfigRepo) GetByID(id int64) (*model.ConfigFile, error) {
	var cf model.ConfigFile
	err := r.db.QueryRow(`SELECT id, name, path, description, group_name, created_at, updated_at FROM config_files WHERE id = ?`, id).
		Scan(&cf.ID, &cf.Name, &cf.Path, &cf.Description, &cf.GroupName, &cf.CreatedAt, &cf.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cf, nil
}

func (r *ConfigRepo) Create(cf *model.ConfigFile) error {
	now := time.Now()
	cf.CreatedAt = now
	cf.UpdatedAt = now
	res, err := r.db.Exec(`INSERT INTO config_files (name, path, description, group_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		cf.Name, cf.Path, cf.Description, cf.GroupName, cf.CreatedAt, cf.UpdatedAt)
	if err != nil {
		return err
	}
	cf.ID, _ = res.LastInsertId()
	return nil
}

func (r *ConfigRepo) Update(cf *model.ConfigFile) error {
	cf.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE config_files SET name=?, path=?, description=?, group_name=?, updated_at=? WHERE id=?`,
		cf.Name, cf.Path, cf.Description, cf.GroupName, cf.UpdatedAt, cf.ID)
	return err
}

func (r *ConfigRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM config_files WHERE id=?`, id)
	return err
}
