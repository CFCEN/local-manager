package repository

import (
	"database/sql"

	"local-dev-workspace/internal/model"
)

type DataTestRepo struct{ db *sql.DB }

func NewDataTestRepo() *DataTestRepo { return &DataTestRepo{db: DB} }

func (r *DataTestRepo) Ping() error {
	return r.db.Ping()
}

func (r *DataTestRepo) TableCounts() ([]model.DataTestTableCount, error) {
	tables := []string{
		"config_files",
		"log_groups",
		"log_files",
		"git_groups",
		"git_projects",
		"project_contexts",
		"applications",
		"application_runs",
	}

	items := make([]model.DataTestTableCount, 0, len(tables))
	for _, name := range tables {
		var count int64
		if err := r.db.QueryRow(`SELECT COUNT(*) FROM ` + name).Scan(&count); err != nil {
			return nil, err
		}
		items = append(items, model.DataTestTableCount{Name: name, Count: count})
	}
	return items, nil
}
