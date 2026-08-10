package service

import (
	"fmt"

	"local-dev-workspace/internal/filesystem"
	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/repository"
)

type ConfigService struct {
	repo *repository.ConfigRepo
}

func NewConfigService(repo *repository.ConfigRepo) *ConfigService {
	return &ConfigService{repo: repo}
}

func (s *ConfigService) List() ([]model.ConfigFile, error) {
	return s.repo.List()
}

func (s *ConfigService) GetDetail(id int64) (*model.ConfigDetail, error) {
	cf, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if cf == nil {
		return nil, fmt.Errorf("config not found")
	}

	content, size, modTime, err := filesystem.ReadFile(cf.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &model.ConfigDetail{
		ID:          cf.ID,
		Name:        cf.Name,
		Path:        cf.Path,
		Description: cf.Description,
		GroupName:   cf.GroupName,
		Content:     content,
		Size:        size,
		ModifiedAt:  modTime,
		CreatedAt:   cf.CreatedAt,
		UpdatedAt:   cf.UpdatedAt,
	}, nil
}

func (s *ConfigService) Create(req model.CreateConfigRequest) (*model.ConfigFile, error) {
	normalized, err := filesystem.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	if err := filesystem.ValidatePath(normalized); err != nil {
		return nil, err
	}

	cf := &model.ConfigFile{
		Name:        req.Name,
		Path:        normalized,
		Description: req.Description,
		GroupName:   req.GroupName,
	}
	if err := s.repo.Create(cf); err != nil {
		return nil, err
	}
	return cf, nil
}

func (s *ConfigService) Update(id int64, req model.UpdateConfigRequest) (*model.ConfigFile, error) {
	cf, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if cf == nil {
		return nil, fmt.Errorf("config not found")
	}
	if req.Name != nil {
		cf.Name = *req.Name
	}
	if req.Path != nil {
		normalized, err := filesystem.NormalizePath(*req.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
		if err := filesystem.ValidatePath(normalized); err != nil {
			return nil, err
		}
		cf.Path = normalized
	}
	if req.Description != nil {
		cf.Description = *req.Description
	}
	if req.GroupName != nil {
		cf.GroupName = *req.GroupName
	}
	if err := s.repo.Update(cf); err != nil {
		return nil, err
	}
	return cf, nil
}

func (s *ConfigService) UpdateContent(id int64, req model.UpdateConfigContentRequest) error {
	cf, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if cf == nil {
		return fmt.Errorf("config not found")
	}
	return filesystem.SafeWrite(cf.Path, req.Content)
}

func (s *ConfigService) Delete(id int64) error {
	// Only delete DB record, NOT the file.
	return s.repo.Delete(id)
}
