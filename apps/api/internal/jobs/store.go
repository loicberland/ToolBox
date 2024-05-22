package jobs

import (
	"sync"

	"toolBox/pkg/modulecontract"
)

type Store struct {
	mu   sync.RWMutex
	jobs map[string]modulecontract.JobStatus
}

func NewStore() *Store {
	return &Store{jobs: map[string]modulecontract.JobStatus{}}
}

func (s *Store) Save(job modulecontract.JobStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

func (s *Store) Get(id string) (modulecontract.JobStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}
