package process

import (
	"log"
	"time"

	"local-dev-workspace/internal/repository"
)

type ApplicationMonitor struct {
	pm   *ProcessManager
	repo *repository.ApplicationRepo
	stop chan struct{}
}

func NewApplicationMonitor(pm *ProcessManager, repo *repository.ApplicationRepo) *ApplicationMonitor {
	return &ApplicationMonitor{
		pm:   pm,
		repo: repo,
		stop: make(chan struct{}),
	}
}

// Start begins polling every 5 seconds.
func (m *ApplicationMonitor) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.check()
			case <-m.stop:
				return
			}
		}
	}()
	log.Println("application monitor started (5s interval)")
}

// Stop the monitor.
func (m *ApplicationMonitor) Stop() {
	close(m.stop)
}

func (m *ApplicationMonitor) check() {
	runs, err := m.repo.ListRunningRuns()
	if err != nil {
		log.Printf("monitor: failed to list running runs: %v", err)
		return
	}

	for _, run := range runs {
		if run.PID == nil {
			continue
		}
		alive := m.pm.IsRunning(*run.PID)
		if !alive {
			now := time.Now()
			run.Status = "STOPPED"
			run.StoppedAt = &now
			if err := m.repo.UpdateRun(&run); err != nil {
				log.Printf("monitor: failed to update run %d: %v", run.ID, err)
			}
			log.Printf("monitor: application %d process %d stopped", run.ApplicationID, *run.PID)
		}
	}
}
