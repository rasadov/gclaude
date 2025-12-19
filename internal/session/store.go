package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/bb/gclaude/internal/config"
)

type Store struct {
	mu       sync.RWMutex
	Sessions []*Session `json:"sessions"`
	filePath string
}

var (
	store     *Store
	storeOnce sync.Once
)

func GetStore() *Store {
	storeOnce.Do(func() {
		store = &Store{
			Sessions: make([]*Session, 0),
			filePath: filepath.Join(config.GetConfigDir(), "sessions.json"),
		}
		store.load()
	})
	return store
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, s)
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := config.EnsureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) Add(session *Session) error {
	s.mu.Lock()
	s.Sessions = append(s.Sessions, session)
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	for i, sess := range s.Sessions {
		if sess.ID == id {
			s.Sessions = append(s.Sessions[:i], s.Sessions[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) FindByBranch(branch string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sess := range s.Sessions {
		if sess.Branch == branch {
			return sess
		}
	}
	return nil
}

func (s *Store) FindByID(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sess := range s.Sessions {
		if sess.ID == id {
			return sess
		}
	}
	return nil
}

func (s *Store) GetAll() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Session, len(s.Sessions))
	copy(result, s.Sessions)
	return result
}

func (s *Store) Update(session *Session) error {
	s.mu.Lock()
	for i, sess := range s.Sessions {
		if sess.ID == session.ID {
			s.Sessions[i] = session
			break
		}
	}
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) Clear() error {
	s.mu.Lock()
	s.Sessions = make([]*Session, 0)
	s.mu.Unlock()
	return s.Save()
}
