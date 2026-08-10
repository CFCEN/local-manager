package repository

import (
	"database/sql"
	"time"

	"local-dev-workspace/internal/model"
)

type WorkspaceRepo struct{ db *sql.DB }

func NewWorkspaceRepo() *WorkspaceRepo { return &WorkspaceRepo{db: DB} }

func (r *WorkspaceRepo) ListContextsByGroup(groupID int64) ([]model.ProjectContext, error) {
	rows, err := r.db.Query(`SELECT id, group_id, name, rag_path, index_path, index_status, last_indexed_at, created_at, updated_at FROM project_contexts WHERE group_id=? ORDER BY created_at`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.ProjectContext
	for rows.Next() {
		var c model.ProjectContext
		if err := rows.Scan(&c.ID, &c.GroupID, &c.Name, &c.RAGPath, &c.IndexPath, &c.IndexStatus, &c.LastIndexedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if items == nil {
		items = []model.ProjectContext{}
	}
	return items, rows.Err()
}

func (r *WorkspaceRepo) GetContextByID(id int64) (*model.ProjectContext, error) {
	var c model.ProjectContext
	err := r.db.QueryRow(`SELECT id, group_id, name, rag_path, index_path, index_status, last_indexed_at, created_at, updated_at FROM project_contexts WHERE id=?`, id).
		Scan(&c.ID, &c.GroupID, &c.Name, &c.RAGPath, &c.IndexPath, &c.IndexStatus, &c.LastIndexedAt, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *WorkspaceRepo) CreateContext(c *model.ProjectContext) error {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.IndexStatus == "" {
		c.IndexStatus = "NOT_INDEXED"
	}
	res, err := r.db.Exec(`INSERT INTO project_contexts (group_id, name, rag_path, index_path, index_status, last_indexed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.GroupID, c.Name, c.RAGPath, c.IndexPath, c.IndexStatus, c.LastIndexedAt, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (r *WorkspaceRepo) UpdateContext(c *model.ProjectContext) error {
	c.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE project_contexts SET name=?, rag_path=?, index_path=?, index_status=?, last_indexed_at=?, updated_at=? WHERE id=?`,
		c.Name, c.RAGPath, c.IndexPath, c.IndexStatus, c.LastIndexedAt, c.UpdatedAt, c.ID)
	return err
}

func (r *WorkspaceRepo) DeleteContext(id int64) error {
	_, err := r.db.Exec(`DELETE FROM project_contexts WHERE id=?`, id)
	return err
}

func (r *WorkspaceRepo) ListContexts() ([]model.ProjectContext, error) {
	rows, err := r.db.Query(`SELECT id, group_id, name, rag_path, index_path, index_status, last_indexed_at, created_at, updated_at FROM project_contexts ORDER BY group_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.ProjectContext
	for rows.Next() {
		var c model.ProjectContext
		if err := rows.Scan(&c.ID, &c.GroupID, &c.Name, &c.RAGPath, &c.IndexPath, &c.IndexStatus, &c.LastIndexedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if items == nil {
		items = []model.ProjectContext{}
	}
	return items, rows.Err()
}
