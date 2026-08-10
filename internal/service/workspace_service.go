package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"local-dev-workspace/internal/filesystem"
	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/rag"
	"local-dev-workspace/internal/repository"
)

type WorkspaceService struct {
	repo *repository.WorkspaceRepo
	root string
}

func NewWorkspaceService(repo *repository.WorkspaceRepo, workspaceRoot string) *WorkspaceService {
	return &WorkspaceService{repo: repo, root: workspaceRoot}
}

func (s *WorkspaceService) GetInfo() (*model.WorkspaceInfo, error) {
	ragDirs, _ := rag.ListDirs(s.root + "/rag")
	indexDirs, _ := rag.ListDirs(s.root + "/index")
	contexts, _ := s.repo.ListContexts()
	if contexts == nil {
		contexts = []model.ProjectContext{}
	}

	return &model.WorkspaceInfo{
		Root:      s.root,
		RAGDirs:   ragDirs,
		IndexDirs: indexDirs,
		Contexts:  contexts,
	}, nil
}

// --- Context CRUD ---

func (s *WorkspaceService) ListContextsByGroup(groupID int64) ([]model.ProjectContext, error) {
	return s.repo.ListContextsByGroup(groupID)
}

func (s *WorkspaceService) GetContext(contextID int64) (*model.ProjectContext, error) {
	c, err := s.repo.GetContextByID(contextID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("context not found")
	}
	return c, nil
}

func (s *WorkspaceService) CreateContext(groupID int64, req model.CreateContextRequest) (*model.ProjectContext, error) {
	if req.RAGPath == "" || req.IndexPath == "" {
		return nil, fmt.Errorf("ragPath and indexPath are required")
	}

	// Resolve to absolute paths for consistent storage
	ragPath, err := filepath.Abs(req.RAGPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ragPath: %w", err)
	}
	indexPath, err := filepath.Abs(req.IndexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve indexPath: %w", err)
	}

	// Ensure directories exist
	os.MkdirAll(ragPath, 0755)
	os.MkdirAll(indexPath, 0755)

	c := &model.ProjectContext{
		GroupID:     groupID,
		Name:        req.Name,
		RAGPath:     ragPath,
		IndexPath:   indexPath,
		IndexStatus: "NOT_INDEXED",
	}
	if err := s.repo.CreateContext(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *WorkspaceService) UpdateContext(contextID int64, req model.UpdateContextRequest) (*model.ProjectContext, error) {
	c, err := s.repo.GetContextByID(contextID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("context not found")
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.RAGPath != nil {
		abs, err := filepath.Abs(*req.RAGPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve ragPath: %w", err)
		}
		c.RAGPath = abs
	}
	if req.IndexPath != nil {
		abs, err := filepath.Abs(*req.IndexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve indexPath: %w", err)
		}
		c.IndexPath = abs
	}
	// indexStatus / lastIndexedAt are NOT user-writable

	if err := s.repo.UpdateContext(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *WorkspaceService) DeleteContext(contextID int64) error {
	// Only delete DB record, not disk directories.
	return s.repo.DeleteContext(contextID)
}

// --- Helpers ---

// resolveContextDir returns the absolute directory path for the given type.
func (s *WorkspaceService) resolveContextDir(contextID int64, fileType string) (string, error) {
	c, err := s.repo.GetContextByID(contextID)
	if err != nil {
		return "", err
	}
	if c == nil {
		return "", fmt.Errorf("context not found")
	}
	var p string
	switch fileType {
	case "rag":
		p = c.RAGPath
	case "index":
		p = c.IndexPath
	default:
		return "", fmt.Errorf("invalid type: must be 'rag' or 'index'")
	}
	return filepath.Abs(p)
}

// isPathWithin checks that child is within parent directory, respecting path boundaries.
func isPathWithin(child, parent string) bool {
	return strings.HasPrefix(child, parent+string(filepath.Separator)) || child == parent
}

// --- Context Files ---

// ListContextFiles lists files in a context's rag or index directory.
func (s *WorkspaceService) ListContextFiles(contextID int64, fileType string) ([]model.ContextFile, error) {
	dir, err := s.resolveContextDir(contextID, fileType)
	if err != nil {
		return nil, err
	}

	entries, err := filesystem.ListDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.ContextFile{}, nil
		}
		return nil, err
	}

	var result []model.ContextFile
	for _, e := range entries {
		result = append(result, model.ContextFile{
			Name:       e.Name,
			Path:       e.Path,
			Size:       e.Size,
			IsDir:      e.IsDir,
			ModifiedAt: e.ModifiedAt,
		})
	}
	if result == nil {
		result = []model.ContextFile{}
	}
	return result, nil
}

// GetContextFileContent reads a file's content from a context directory.
func (s *WorkspaceService) GetContextFileContent(contextID int64, fileType, fileName string) (*model.ContextFileContent, error) {
	if err := filesystem.ValidateFileName(fileName); err != nil {
		return nil, fmt.Errorf("invalid file name: %w", err)
	}
	if !filesystem.IsTextFile(fileName) {
		return nil, fmt.Errorf("file type not supported: only text files can be read")
	}

	dir, err := s.resolveContextDir(contextID, fileType)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(dir, fileName)
	// Security: ensure resolved path is within the context directory
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found")
		}
		return nil, err
	}
	if !isPathWithin(resolved, dir) {
		return nil, fmt.Errorf("invalid path")
	}

	content, size, modTime, err := filesystem.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found")
		}
		return nil, err
	}

	return &model.ContextFileContent{
		Path:       resolved,
		Content:    content,
		Size:       size,
		ModifiedAt: modTime,
	}, nil
}

// CreateContextFile creates a new file in a context directory.
func (s *WorkspaceService) CreateContextFile(contextID int64, req model.CreateFileRequest) (*model.ContextFileContent, error) {
	if err := filesystem.ValidateFileName(req.FileName); err != nil {
		return nil, fmt.Errorf("invalid file name: %w", err)
	}
	if !filesystem.IsTextFile(req.FileName) {
		return nil, fmt.Errorf("file type not supported")
	}
	if len(req.Content) > filesystem.MaxFileSize {
		return nil, fmt.Errorf("file too large: max 1MB")
	}

	dir, err := s.resolveContextDir(contextID, req.Type)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(dir, req.FileName)
	resolved, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, err
	}
	if !isPathWithin(resolved, dir) {
		return nil, fmt.Errorf("invalid path")
	}

	if filesystem.FileExists(resolved) {
		return nil, fmt.Errorf("file already exists")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	if err := filesystem.SafeWrite(resolved, req.Content); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Re-read to get accurate size and mod time
	content, size, modTime, err := filesystem.ReadFile(resolved)
	if err != nil {
		return nil, err
	}

	return &model.ContextFileContent{
		Path:       resolved,
		Content:    content,
		Size:       size,
		ModifiedAt: modTime,
	}, nil
}

// UpdateContextFile updates an existing file in a context directory.
func (s *WorkspaceService) UpdateContextFile(contextID int64, req model.UpdateFileRequest) (*model.ContextFileContent, error) {
	if err := filesystem.ValidateFileName(req.FileName); err != nil {
		return nil, fmt.Errorf("invalid file name: %w", err)
	}
	if !filesystem.IsTextFile(req.FileName) {
		return nil, fmt.Errorf("file type not supported")
	}
	if len(req.Content) > filesystem.MaxFileSize {
		return nil, fmt.Errorf("file too large: max 1MB")
	}

	dir, err := s.resolveContextDir(contextID, req.Type)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(dir, req.FileName)
	resolved, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, err
	}
	if !isPathWithin(resolved, dir) {
		return nil, fmt.Errorf("invalid path")
	}

	if !filesystem.FileExists(resolved) {
		return nil, fmt.Errorf("file not found")
	}

	if err := filesystem.SafeWrite(resolved, req.Content); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	content, size, modTime, err := filesystem.ReadFile(resolved)
	if err != nil {
		return nil, err
	}

	return &model.ContextFileContent{
		Path:       resolved,
		Content:    content,
		Size:       size,
		ModifiedAt: modTime,
	}, nil
}

// DeleteContextFile deletes a file from a context directory.
func (s *WorkspaceService) DeleteContextFile(contextID int64, fileType, fileName string) error {
	if err := filesystem.ValidateFileName(fileName); err != nil {
		return fmt.Errorf("invalid file name: %w", err)
	}

	dir, err := s.resolveContextDir(contextID, fileType)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(dir, fileName)
	resolved, err := filepath.Abs(fullPath)
	if err != nil {
		return err
	}
	if !isPathWithin(resolved, dir) {
		return fmt.Errorf("invalid path")
	}

	if !filesystem.FileExists(resolved) {
		return fmt.Errorf("file not found")
	}

	return filesystem.DeleteFile(resolved)
}
