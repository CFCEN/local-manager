package service

import (
	"fmt"

	"local-dev-workspace/internal/filesystem"
	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/repository"
)

type LogService struct {
	repo *repository.LogRepo
}

func NewLogService(repo *repository.LogRepo) *LogService {
	return &LogService{repo: repo}
}

// --- Log Groups ---

func (s *LogService) ListGroups() ([]model.LogGroup, error) {
	return s.repo.ListGroups()
}

func (s *LogService) CreateGroup(req model.CreateLogGroupRequest) (*model.LogGroup, error) {
	g := &model.LogGroup{Name: req.Name, Description: req.Description}
	if err := s.repo.CreateGroup(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *LogService) UpdateGroup(id int64, req model.UpdateLogGroupRequest) (*model.LogGroup, error) {
	g, err := s.repo.GetGroupByID(id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("log group not found")
	}
	if req.Name != nil {
		g.Name = *req.Name
	}
	if req.Description != nil {
		g.Description = *req.Description
	}
	if err := s.repo.UpdateGroup(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *LogService) DeleteGroup(id int64) error {
	return s.repo.DeleteGroup(id)
}

// --- Log Files ---

func (s *LogService) ListFiles() ([]model.LogFile, error) {
	return s.repo.ListFiles()
}

func (s *LogService) GetFile(id int64) (*model.LogFile, error) {
	f, err := s.repo.GetFileByID(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("log file not found")
	}
	return f, nil
}

func (s *LogService) CreateFile(req model.CreateLogFileRequest) (*model.LogFile, error) {
	normalized, err := filesystem.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	if err := filesystem.ValidatePath(normalized); err != nil {
		return nil, err
	}

	f := &model.LogFile{
		Name:        req.Name,
		Path:        normalized,
		GroupID:     req.GroupID,
		Description: req.Description,
	}
	if err := s.repo.CreateFile(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *LogService) UpdateFile(id int64, req model.UpdateLogFileRequest) (*model.LogFile, error) {
	f, err := s.repo.GetFileByID(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("log file not found")
	}
	if req.Name != nil {
		f.Name = *req.Name
	}
	if req.Path != nil {
		normalized, err := filesystem.NormalizePath(*req.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
		if err := filesystem.ValidatePath(normalized); err != nil {
			return nil, err
		}
		f.Path = normalized
	}
	if req.GroupID != nil {
		f.GroupID = req.GroupID
	}
	if req.Description != nil {
		f.Description = *req.Description
	}
	if err := s.repo.UpdateFile(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *LogService) DeleteFile(id int64) error {
	return s.repo.DeleteFile(id)
}

func (s *LogService) GetContent(id int64, tail int) (*model.LogContent, error) {
	f, err := s.repo.GetFileByID(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("log file not found")
	}

	content, size, modTime, err := filesystem.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}

	// Apply tail if requested
	if tail > 0 && len(content) > 0 {
		runes := []rune(content)
		if tail < len(runes) {
			content = string(runes[len(runes)-tail:])
		}
	}

	return &model.LogContent{
		Content:    content,
		Size:       size,
		ModifiedAt: modTime,
	}, nil
}
