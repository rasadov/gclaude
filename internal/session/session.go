package session

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusRunning      Status = "running"
	StatusWaitingInput Status = "waiting_input"
	StatusIdle         Status = "idle"
	StatusStopped      Status = "stopped"
)

type Session struct {
	ID           string    `json:"id"`
	Branch       string    `json:"branch"`
	RepoPath     string    `json:"repo_path"`
	WorktreePath string    `json:"worktree_path"`
	TmuxSession  string    `json:"tmux_session"`
	Status       Status    `json:"status"`
	NeedsInput   bool      `json:"needs_input"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	LastOutput   string    `json:"-"`
}

func NewSession(branch, repoPath, worktreePath string) *Session {
	id := uuid.New().String()[:8]
	return &Session{
		ID:           id,
		Branch:       branch,
		RepoPath:     repoPath,
		WorktreePath: worktreePath,
		TmuxSession:  "gclaude-" + sanitizeBranch(branch),
		Status:       StatusRunning,
		NeedsInput:   false,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
}

func sanitizeBranch(branch string) string {
	result := make([]byte, 0, len(branch))
	for i := 0; i < len(branch); i++ {
		c := branch[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '-')
		}
	}
	return string(result)
}

func (s *Session) UpdateActivity() {
	s.LastActivity = time.Now()
}

func (s *Session) SetNeedsInput(needs bool) {
	s.NeedsInput = needs
	if needs {
		s.Status = StatusWaitingInput
	} else if s.Status == StatusWaitingInput {
		s.Status = StatusRunning
	}
}
