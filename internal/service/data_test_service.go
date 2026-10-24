package service

import (
	"local-dev-workspace/internal/model"
	"local-dev-workspace/internal/repository"
)

type DataTestService struct {
	repo *repository.DataTestRepo
}

func NewDataTestService(repo *repository.DataTestRepo) *DataTestService {
	return &DataTestService{repo: repo}
}

func (s *DataTestService) Ping() (*model.DataTestPing, error) {
	if err := s.repo.Ping(); err != nil {
		return nil, err
	}
	return &model.DataTestPing{OK: true, Message: "database ok"}, nil
}

func (s *DataTestService) Summary() (*model.DataTestSummary, error) {
	tables, err := s.repo.TableCounts()
	if err != nil {
		return nil, err
	}
	return &model.DataTestSummary{OK: true, Tables: tables}, nil
}

func (s *DataTestService) Echo(req model.DataTestEchoRequest) *model.DataTestEchoResponse {
	return &model.DataTestEchoResponse{Message: req.Message}
}
