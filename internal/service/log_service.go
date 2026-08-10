package service

import (
	"errors"
	"fmt"
	"os"
	"strings"

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

// ErrLogFileNotFound is returned when the DB record is missing.
var ErrLogFileNotFound = errors.New("log file not found")

// ErrLogFileOnDiskMissing is returned when the DB record exists but the
// underlying file is gone or unreadable on disk.
var ErrLogFileOnDiskMissing = errors.New("log file not found on disk")

// ErrLogFileBinary is returned when the target file appears to be binary.
var ErrLogFileBinary = errors.New("log file is not a text file")

func (s *LogService) GetContent(id int64, tail int) (*model.LogContent, error) {
	f, err := s.repo.GetFileByID(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrLogFileNotFound
	}

	// tail <= 0 means "whole file". Fall back to the full read path.
	if tail <= 0 {
		content, size, modTime, err := filesystem.ReadFile(f.Path)
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				return nil, fmt.Errorf("%w: %v", ErrLogFileOnDiskMissing, err)
			}
			return nil, err
		}
		total := int64(strings.Count(content, "\n"))
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			total++
		}
		returned := int(total)
		return &model.LogContent{
			Content:       content,
			Size:          size,
			ModifiedAt:    modTime,
			TotalLines:    total,
			ReturnedLines: returned,
			Truncated:     false,
		}, nil
	}

	res, err := filesystem.ReadTailLines(f.Path, tail, filesystem.DefaultTailMaxBytes)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return nil, fmt.Errorf("%w: %v", ErrLogFileOnDiskMissing, err)
		}
		if strings.Contains(err.Error(), "binary file not supported") {
			return nil, ErrLogFileBinary
		}
		return nil, err
	}

	return &model.LogContent{
		Content:       res.Content,
		Size:          res.Size,
		ModifiedAt:    res.ModifiedAt,
		TotalLines:    res.TotalLines,
		ReturnedLines: res.ReturnedLines,
		Truncated:     res.Truncated,
	}, nil
}
