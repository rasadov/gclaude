package session

import (
	"fmt"

	"github.com/bb/gclaude/internal/tmux"
	"github.com/bb/gclaude/internal/worktree"
)

type Manager struct {
	store *Store
}

func NewManager() *Manager {
	return &Manager{
		store: GetStore(),
	}
}

func (m *Manager) Start(branch, repoPath string, createWorktree bool) (*Session, error) {
	if existing := m.store.FindByBranch(branch); existing != nil {
		exists, _ := tmux.SessionExists(existing.TmuxSession)
		if exists {
			return nil, fmt.Errorf("session for branch '%s' already exists", branch)
		}
		m.store.Remove(existing.ID)
	}

	repoRoot, err := worktree.GetRepoRoot(repoPath)
	if err != nil {
		return nil, err
	}

	var sessionPath string

	if createWorktree {
		if worktree.Exists(repoRoot, branch) {
			sessionPath = worktree.GetWorktreePath(repoRoot, branch)
		} else {
			sessionPath, err = worktree.Create(repoRoot, branch)
			if err != nil {
				return nil, fmt.Errorf("failed to create worktree: %w", err)
			}
		}
	} else {
		sessionPath = repoRoot
	}

	sess := NewSession(branch, repoRoot, sessionPath)

	if err := tmux.CreateSession(sess.TmuxSession, sessionPath, "claude"); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	if err := m.store.Add(sess); err != nil {
		tmux.KillSession(sess.TmuxSession)
		return nil, err
	}

	return sess, nil
}

func (m *Manager) Stop(branch string, removeWorktree bool) error {
	sess := m.store.FindByBranch(branch)
	if sess == nil {
		return fmt.Errorf("no session found for branch '%s'", branch)
	}

	if exists, _ := tmux.SessionExists(sess.TmuxSession); exists {
		if err := tmux.KillSession(sess.TmuxSession); err != nil {
			return fmt.Errorf("failed to kill tmux session: %w", err)
		}
	}

	if removeWorktree && sess.WorktreePath != sess.RepoPath {
		worktree.Remove(sess.RepoPath, sess.Branch)
	}

	return m.store.Remove(sess.ID)
}

func (m *Manager) StopAll(removeWorktrees bool) error {
	sessions := m.store.GetAll()
	var lastErr error

	for _, sess := range sessions {
		if err := m.Stop(sess.Branch, removeWorktrees); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (m *Manager) Attach(branch string) error {
	sess := m.store.FindByBranch(branch)
	if sess == nil {
		return fmt.Errorf("no session found for branch '%s'", branch)
	}

	exists, _ := tmux.SessionExists(sess.TmuxSession)
	if !exists {
		m.store.Remove(sess.ID)
		return fmt.Errorf("tmux session no longer exists")
	}

	return tmux.AttachSession(sess.TmuxSession)
}

func (m *Manager) List() []*Session {
	sessions := m.store.GetAll()

	for _, sess := range sessions {
		if exists, _ := tmux.SessionExists(sess.TmuxSession); !exists {
			sess.Status = StatusStopped
		}
	}

	return sessions
}

func (m *Manager) Cleanup() (int, error) {
	sessions := m.store.GetAll()
	removed := 0

	for _, sess := range sessions {
		exists, _ := tmux.SessionExists(sess.TmuxSession)
		if !exists {
			m.store.Remove(sess.ID)
			removed++
		}
	}

	return removed, nil
}

func (m *Manager) GetStore() *Store {
	return m.store
}
