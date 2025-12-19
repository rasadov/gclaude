package monitor

import (
	"sync"
	"time"

	"github.com/bb/gclaude/internal/config"
	"github.com/bb/gclaude/internal/notify"
	"github.com/bb/gclaude/internal/session"
	"github.com/bb/gclaude/internal/tmux"
)

type sessionState struct {
	lastOutput    string
	lastChange    time.Time
	notified      bool
	wasActive     bool
}

type Monitor struct {
	store    *session.Store
	cfg      *config.Config
	stopChan chan struct{}
	wg       sync.WaitGroup
	states   map[string]*sessionState
	mu       sync.Mutex
}

func New(store *session.Store, cfg *config.Config) *Monitor {
	return &Monitor{
		store:    store,
		cfg:      cfg,
		stopChan: make(chan struct{}),
		states:   make(map[string]*sessionState),
	}
}

func (m *Monitor) Start() {
	m.wg.Add(1)
	go m.run()
}

func (m *Monitor) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

func (m *Monitor) run() {
	defer m.wg.Done()

	pollInterval := time.Duration(m.cfg.Monitor.PollIntervalMs) * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkSessions()
		}
	}
}

func (m *Monitor) checkSessions() {
	sessions := m.store.GetAll()

	for _, sess := range sessions {
		if sess.Status == session.StatusStopped {
			continue
		}

		exists, err := tmux.SessionExists(sess.TmuxSession)
		if err != nil || !exists {
			sess.Status = session.StatusStopped
			m.store.Update(sess)
			continue
		}

		output, err := tmux.CapturePane(sess.TmuxSession, 100)
		if err != nil {
			continue
		}

		m.mu.Lock()
		state, exists := m.states[sess.ID]
		if !exists {
			state = &sessionState{
				lastOutput: output,
				lastChange: time.Now(),
				notified:   false,
				wasActive:  true,
			}
			m.states[sess.ID] = state
			m.mu.Unlock()
			continue
		}

		outputChanged := output != state.lastOutput
		now := time.Now()

		if outputChanged {
			// Output is changing - Claude is active
			state.lastOutput = output
			state.lastChange = now
			state.notified = false
			state.wasActive = true
			sess.UpdateActivity()
			sess.Status = session.StatusRunning
			sess.NeedsInput = false
			m.store.Update(sess)
		} else {
			// Output hasn't changed
			idleTime := now.Sub(state.lastChange)
			idleThreshold := time.Duration(m.cfg.Monitor.IdleThresholdS) * time.Second

			if idleTime > idleThreshold && state.wasActive && !state.notified {
				// Claude has stopped - notify user
				state.notified = true
				state.wasActive = false
				sess.Status = session.StatusWaitingInput
				sess.NeedsInput = true
				m.store.Update(sess)

				m.notify(sess)
			}
		}
		m.mu.Unlock()
	}
}

func (m *Monitor) notify(sess *session.Session) {
	// Skip notification if user had recent keyboard input (within idle threshold)
	// This means user is actively typing/thinking
	if tmux.HasRecentInput(sess.TmuxSession, m.cfg.Monitor.IdleThresholdS) {
		return
	}

	// Check if user is actively viewing this session
	// Skip notification if: attached AND terminal window is focused
	if tmux.IsSessionAttached(sess.TmuxSession) {
		tty := tmux.GetAttachedClientTTY(sess.TmuxSession)
		if tty != "" && notify.IsTerminalFocused(tty) {
			// User is looking at this session - no notification needed
			return
		}
	}

	title := "gclaude: " + sess.Branch
	message := "Claude has stopped - waiting for input or finished"

	if m.cfg.Notification.Desktop {
		notify.Desktop(title, message)
	}

	if m.cfg.Notification.Sound {
		notify.Sound(m.cfg.Notification.SoundFile)
	}
}
