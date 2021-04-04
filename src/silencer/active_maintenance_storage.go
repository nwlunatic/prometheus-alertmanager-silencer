package silencer

import "sync"

type ActiveMaintenanceStorage struct {
	items map[MaintenanceHash]struct{}
	mux   sync.Mutex
}

func NewActiveMaintenanceStorage() *ActiveMaintenanceStorage {
	return &ActiveMaintenanceStorage{
		make(map[MaintenanceHash]struct{}),
		sync.Mutex{},
	}
}

func (s *ActiveMaintenanceStorage) Add(hash MaintenanceHash) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.items[hash] = struct{}{}
}

func (s *ActiveMaintenanceStorage) Delete(hash MaintenanceHash) {
	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.items, hash)
}

func (s *ActiveMaintenanceStorage) IsActive(hash MaintenanceHash) bool {
	_, ok := s.items[hash]
	return ok
}
