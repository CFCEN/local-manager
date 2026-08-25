package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"local-dev-workspace/internal/filesystem"
	"local-dev-workspace/internal/gitutil"
	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/repository"
)

type GitService struct {
	repo *repository.GitRepo
	root string // workspace root
}

func NewGitService(repo *repository.GitRepo, workspaceRoot string) *GitService {
	return &GitService{repo: repo, root: workspaceRoot}
}

// --- Scan ---

func (s *GitService) Scan(req model.ScanRequest) ([]model.ScanResult, error) {
	normalized, err := filesystem.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid scan path: %w", err)
	}

	info, err := os.Stat(normalized)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("scan path is not a valid directory")
	}

	maxDepth := req.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 5
	}

	repos, err := gitutil.ScanDirectory(normalized, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	var results []model.ScanResult
	for _, r := range repos {
		results = append(results, model.ScanResult{
			Name:   r.Name,
			Path:   r.Path,
			Branch: r.Branch,
			Remote: r.Remote,
		})

		project := &model.GitProject{
			Name:   r.Name,
			Path:   r.Path,
			Branch: r.Branch,
			Remote: r.Remote,
		}
		if err := s.repo.UpsertProject(project); err != nil {
			log.Printf("failed to upsert project %s: %v", r.Path, err)
		}
	}
	if results == nil {
		results = []model.ScanResult{}
	}
	return results, nil
}

// --- Git Groups ---

func (s *GitService) ListGroups() ([]model.GitGroup, error) {
	return s.repo.ListGroups()
}

func (s *GitService) GetGroup(id int64) (*model.GitGroup, error) {
	g, err := s.repo.GetGroupByID(id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("group not found")
	}
	return g, nil
}

func (s *GitService) CreateGroup(req model.CreateGitGroupRequest) (*model.GitGroup, error) {
	g := &model.GitGroup{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.RAGPath != "" {
		g.RAGPath = req.RAGPath
	} else {
		g.RAGPath = filepath.Join(s.root, "rag", req.Name)
	}
	if req.IndexPath != "" {
		g.IndexPath = req.IndexPath
	} else {
		g.IndexPath = filepath.Join(s.root, "index", req.Name)
	}

	if err := s.repo.CreateGroup(g); err != nil {
		return nil, err
	}

	os.MkdirAll(g.RAGPath, 0755)
	os.MkdirAll(g.IndexPath, 0755)

	return g, nil
}

func (s *GitService) UpdateGroup(id int64, req model.UpdateGitGroupRequest) (*model.GitGroup, error) {
	g, err := s.repo.GetGroupByID(id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("group not found")
	}
	if req.Name != nil {
		g.Name = *req.Name
	}
	if req.Description != nil {
		g.Description = *req.Description
	}
	if req.RAGPath != nil {
		g.RAGPath = *req.RAGPath
	}
	if req.IndexPath != nil {
		g.IndexPath = *req.IndexPath
	}
	if err := s.repo.UpdateGroup(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *GitService) DeleteGroup(id int64) error {
	return s.repo.DeleteGroup(id)
}

// --- Git Projects ---

// ListProjects returns all git projects. groupName is always populated (cheap
// DB join). When enrichStatus is true, each project's live git status and last
// commit are computed with bounded concurrency and a per-repo timeout so a
// broken repo cannot stall the whole request.
func (s *GitService) ListProjects(enrichStatus bool) ([]model.GitProject, error) {
	items, err := s.repo.ListProjects()
	if err != nil {
		return nil, err
	}
	if !enrichStatus || len(items) == 0 {
		return items, nil
	}

	statuses := s.computeStatuses(collectPaths(items))
	for i := range items {
		if st, ok := statuses[items[i].Path]; ok {
			items[i].Status = st.Status
			items[i].LastCommit = st.LastCommit
		}
	}
	return items, nil
}

// BatchStatus computes live git status/lastCommit for the given project IDs.
func (s *GitService) BatchStatus(ids []int64) ([]model.GitProjectStatusItem, error) {
	result := make([]model.GitProjectStatusItem, 0, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	type idPath struct {
		id   int64
		path string
	}
	var targets []idPath
	for _, id := range ids {
		p, err := s.repo.GetProjectByID(id)
		if err != nil || p == nil {
			// Unknown project → report UNKNOWN so the frontend can render something.
			result = append(result, model.GitProjectStatusItem{ID: id, Status: "UNKNOWN"})
			continue
		}
		targets = append(targets, idPath{id: id, path: p.Path})
	}

	paths := make([]string, 0, len(targets))
	for _, t := range targets {
		paths = append(paths, t.path)
	}
	statuses := s.computeStatuses(paths)
	for _, t := range targets {
		st := statuses[t.path]
		if st.Status == "" {
			st.Status = "UNKNOWN"
		}
		result = append(result, model.GitProjectStatusItem{ID: t.id, Status: st.Status, LastCommit: st.LastCommit})
	}
	return result, nil
}

type gitStatusResult struct {
	Status     string
	LastCommit string
}

// computeStatuses runs git status/last-commit for each unique path with a
// bounded worker pool (concurrency 8) and a per-repo timeout.
func (s *GitService) computeStatuses(paths []string) map[string]gitStatusResult {
	const concurrency = 8
	out := make(map[string]gitStatusResult, len(paths))

	// De-duplicate paths.
	unique := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, p := range unique {
		wg.Add(1)
		sem <- struct{}{}
		go func(path string) {
			defer wg.Done()
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), gitutil.DefaultGitTimeout)
			defer cancel()

			status := gitutil.GetStatusContext(ctx, path)
			commit := gitutil.GetLastCommitContext(ctx, path)

			mu.Lock()
			out[path] = gitStatusResult{Status: status, LastCommit: commit}
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	return out
}

func collectPaths(items []model.GitProject) []string {
	paths := make([]string, 0, len(items))
	for i := range items {
		paths = append(paths, items[i].Path)
	}
	return paths
}

func (s *GitService) GetProjectDetail(id int64) (*model.GitProjectDetail, error) {
	p, err := s.repo.GetProjectByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}

	detail := &model.GitProjectDetail{
		ID:        p.ID,
		Name:      p.Name,
		Path:      p.Path,
		Branch:    p.Branch,
		Remote:    p.Remote,
		GroupID:   p.GroupID,
		RAGPath:   p.RAGPath,
		IndexPath: p.IndexPath,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	// Get live git info
	detail.Status = gitutil.GetStatus(p.Path)
	detail.LastCommit = gitutil.GetLastCommit(p.Path)
	if detail.Branch == "" {
		detail.Branch = getBranch(p.Path)
	}

	// Group name
	if p.GroupID != nil {
		g, _ := s.repo.GetGroupByID(*p.GroupID)
		if g != nil {
			detail.GroupName = g.Name
		}
	}

	return detail, nil
}

func (s *GitService) UpdateProject(id int64, req model.UpdateGitProjectRequest) (*model.GitProject, error) {
	p, err := s.repo.GetProjectByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.GroupID != nil {
		p.GroupID = req.GroupID
	}
	if req.RAGPath != nil {
		p.RAGPath = *req.RAGPath
	}
	if req.IndexPath != nil {
		p.IndexPath = *req.IndexPath
	}
	if err := s.repo.UpdateProject(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *GitService) DeleteProject(id int64) error {
	return s.repo.DeleteProject(id)
}

func (s *GitService) AssignGroup(projectID int64, req model.AssignGroupRequest) (*model.GitProject, error) {
	p, err := s.repo.GetProjectByID(projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}
	return s.assignProjectToGroup(p, req.GroupID)
}

func (s *GitService) BatchAssignGroup(req model.BatchAssignGroupRequest) ([]model.GitProject, error) {
	if len(req.ProjectIDs) == 0 {
		return nil, fmt.Errorf("projectIds cannot be empty")
	}

	// Verify group exists
	g, err := s.repo.GetGroupByID(req.GroupID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("group not found")
	}

	var results []model.GitProject
	for _, pid := range req.ProjectIDs {
		p, err := s.repo.GetProjectByID(pid)
		if err != nil {
			log.Printf("failed to get project %d: %v", pid, err)
			continue
		}
		if p == nil {
			log.Printf("project %d not found, skipped", pid)
			continue
		}
		result, err := s.assignProjectToGroup(p, req.GroupID)
		if err != nil {
			log.Printf("failed to assign project %d to group: %v", pid, err)
			continue
		}
		results = append(results, *result)
	}
	if results == nil {
		results = []model.GitProject{}
	}
	return results, nil
}

func (s *GitService) ListProjectsByGroup(groupID int64) ([]model.GitProject, error) {
	g, err := s.repo.GetGroupByID(groupID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("group not found")
	}
	return s.repo.ListProjectsByGroup(groupID)
}

func (s *GitService) assignProjectToGroup(p *model.GitProject, groupID int64) (*model.GitProject, error) {
	g, err := s.repo.GetGroupByID(groupID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("group not found")
	}

	p.GroupID = &groupID

	if g.RAGPath != "" {
		os.MkdirAll(g.RAGPath, 0755)
		p.RAGPath = g.RAGPath
	}
	if g.IndexPath != "" {
		os.MkdirAll(g.IndexPath, 0755)
		p.IndexPath = g.IndexPath
	}

	if err := s.repo.UpdateProject(p); err != nil {
		return nil, err
	}
	return p, nil
}

func getBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
