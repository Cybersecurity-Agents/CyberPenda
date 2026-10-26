package session

// outputFlushRegistration identifies one Harness run. Its cleanup cannot remove
// a newer run's barrier for the same Session.
type outputFlushRegistration struct{ flush func() error }

// RegisterRuntimeOutputFlush installs the active Harness run's pre-write barrier.
// The callback must append its batch without calling another pre-write barrier.
func (s *Service) RegisterRuntimeOutputFlush(id string, flush func() error) func() {
	registration := &outputFlushRegistration{flush: flush}
	s.outputMu.Lock()
	if s.outputFlushes == nil {
		s.outputFlushes = map[string]*outputFlushRegistration{}
	}
	s.outputFlushes[id] = registration
	s.outputMu.Unlock()
	return func() {
		s.outputMu.Lock()
		defer s.outputMu.Unlock()
		if s.outputFlushes[id] == registration {
			delete(s.outputFlushes, id)
		}
	}
}

// FlushRuntimeOutputs must run before a caller opens a transaction that writes
// Session Events or changes a Continuation. Calling it inside a transaction can
// deadlock the Store's single SQLite connection.
func (s *Service) FlushRuntimeOutputs(id string) error {
	s.outputMu.Lock()
	registration := s.outputFlushes[id]
	s.outputMu.Unlock()
	if registration == nil || registration.flush == nil {
		return nil
	}
	return registration.flush()
}
