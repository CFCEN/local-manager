package repository

import (
	"database/sql"
	"time"

	"local-dev-workspace/internal/model"
)

type GitRepo struct{ db *sql.DB }

func NewGitRepo() *GitRepo { return &GitRepo{db: DB} }

// --- Git Groups ---

func (r *GitRepo) ListGroups() ([]model.GitGroup, error) {
	rows, err := r.db.Query(`SELECT g.id, g.name, g.description, g.rag_path, g.index_path, g.created_at, g.updated_at, COUNT(p.id) as project_count
		FROM git_groups g LEFT JOIN git_projects p ON p.group_id = g.id GROUP BY g.id ORDER BY g.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.GitGroup
	for rows.Next() {
		var g model.GitGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.RAGPath, &g.IndexPath, &g.CreatedAt, &g.UpdatedAt, &g.ProjectCount); err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	if items == nil {
		items = []model.GitGroup{}
	}
	return items, rows.Err()
}

func (r *GitRepo) GetGroupByID(id int64) (*model.GitGroup, error) {
	var g model.GitGroup
	err := r.db.QueryRow(`SELECT g.id, g.name, g.description, g.rag_path, g.index_path, g.created_at, g.updated_at, COUNT(p.id) as project_count
		FROM git_groups g LEFT JOIN git_projects p ON p.group_id = g.id WHERE g.id=? GROUP BY g.id`, id).
		Scan(&g.ID, &g.Name, &g.Description, &g.RAGPath, &g.IndexPath, &g.CreatedAt, &g.UpdatedAt, &g.ProjectCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &g, err
}

func (r *GitRepo) CreateGroup(g *model.GitGroup) error {
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now
	res, err := r.db.Exec(`INSERT INTO git_groups (name, description, rag_path, index_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		g.Name, g.Description, g.RAGPath, g.IndexPath, g.CreatedAt, g.UpdatedAt)
	if err != nil {
		return err
	}
	g.ID, _ = res.LastInsertId()
	return nil
}

func (r *GitRepo) UpdateGroup(g *model.GitGroup) error {
	g.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE git_groups SET name=?, description=?, rag_path=?, index_path=?, updated_at=? WHERE id=?`,
		g.Name, g.Description, g.RAGPath, g.IndexPath, g.UpdatedAt, g.ID)
	return err
}

func (r *GitRepo) DeleteGroup(id int64) error {
	_, err := r.db.Exec(`DELETE FROM git_groups WHERE id=?`, id)
	return err
}

// --- Git Projects ---

func (r *GitRepo) ListProjects() ([]model.GitProject, error) {
	rows, err := r.db.Query(`SELECT id, name, path, branch, remote, group_id, rag_path, index_path, created_at, updated_at FROM git_projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.GitProject
	for rows.Next() {
		var p model.GitProject
		if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.Branch, &p.Remote, &p.GroupID, &p.RAGPath, &p.IndexPath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if items == nil {
		items = []model.GitProject{}
	}
	return items, rows.Err()
}

func (r *GitRepo) GetProjectByID(id int64) (*model.GitProject, error) {
	var p model.GitProject
	err := r.db.QueryRow(`SELECT id, name, path, branch, remote, group_id, rag_path, index_path, created_at, updated_at FROM git_projects WHERE id=?`, id).
		Scan(&p.ID, &p.Name, &p.Path, &p.Branch, &p.Remote, &p.GroupID, &p.RAGPath, &p.IndexPath, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (r *GitRepo) GetProjectByPath(path string) (*model.GitProject, error) {
	var p model.GitProject
	err := r.db.QueryRow(`SELECT id, name, path, branch, remote, group_id, rag_path, index_path, created_at, updated_at FROM git_projects WHERE path=?`, path).
		Scan(&p.ID, &p.Name, &p.Path, &p.Branch, &p.Remote, &p.GroupID, &p.RAGPath, &p.IndexPath, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (r *GitRepo) UpsertProject(p *model.GitProject) error {
	existing, err := r.GetProjectByPath(p.Path)
	if err != nil {
		return err
	}
	now := time.Now()
	if existing != nil {
		p.ID = existing.ID
		p.CreatedAt = existing.CreatedAt
		p.UpdatedAt = now
		_, err := r.db.Exec(`UPDATE git_projects SET name=?, branch=?, remote=?, updated_at=? WHERE id=?`,
			p.Name, p.Branch, p.Remote, p.UpdatedAt, p.ID)
		return err
	}
	p.CreatedAt = now
	p.UpdatedAt = now
	res, err := r.db.Exec(`INSERT INTO git_projects (name, path, branch, remote, group_id, rag_path, index_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Path, p.Branch, p.Remote, p.GroupID, p.RAGPath, p.IndexPath, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return err
	}
	p.ID, _ = res.LastInsertId()
	return nil
}

func (r *GitRepo) UpdateProject(p *model.GitProject) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE git_projects SET name=?, branch=?, remote=?, group_id=?, rag_path=?, index_path=?, updated_at=? WHERE id=?`,
		p.Name, p.Branch, p.Remote, p.GroupID, p.RAGPath, p.IndexPath, p.UpdatedAt, p.ID)
	return err
}

func (r *GitRepo) DeleteProject(id int64) error {
	_, err := r.db.Exec(`DELETE FROM git_projects WHERE id=?`, id)
	return err
}

func (r *GitRepo) ListProjectsByGroup(groupID int64) ([]model.GitProject, error) {
	rows, err := r.db.Query(`SELECT id, name, path, branch, remote, group_id, rag_path, index_path, created_at, updated_at FROM git_projects WHERE group_id=? ORDER BY name`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.GitProject
	for rows.Next() {
		var p model.GitProject
		if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.Branch, &p.Remote, &p.GroupID, &p.RAGPath, &p.IndexPath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if items == nil {
		items = []model.GitProject{}
	}
	return items, rows.Err()
}
