package monitor

import (
	"strings"
	"sync"
	"time"

	"github.com/bb/gclaude/internal/config"
	"github.com/bb/gclaude/internal/notify"
	"github.com/bb/gclaude/internal/session"
	"github.com/bb/gclaude/internal/tmux"
)

type Monitor struct {
	store         *session.Store
	cfg           *config.Config
	stopChan      chan struct{}
	wg            sync.WaitGroup
	lastOutputs   map[string]string
	lastNotify    map[string]time.Time
	mu            sync.Mutex
}

func New(store *session.Store, cfg *config.Config) *Monitor {
	return &Monitor{
		store:       store,
		cfg:         cfg,
		stopChan:    make(chan struct{}),
		lastOutputs: make(map[string]string),
		lastNotify:  make(map[string]time.Time),
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

		output, err := tmux.CapturePane(sess.TmuxSession, 50)
		if err != nil {
			continue
		}

		m.mu.Lock()
		lastOutput := m.lastOutputs[sess.ID]
		m.lastOutputs[sess.ID] = output
		m.mu.Unlock()

		outputChanged := output != lastOutput
		if outputChanged {
			sess.UpdateActivity()
		}

		lastLines := getLastLines(output, 5)
		needsInput := MatchesInputPattern(lastLines)

		idleThreshold := time.Duration(m.cfg.Monitor.IdleThresholdS) * time.Second
		isIdle := time.Since(sess.LastActivity) > idleThreshold

		if needsInput && isIdle && !sess.NeedsInput {
			sess.SetNeedsInput(true)
			m.store.Update(sess)
			m.maybeNotify(sess)
		} else if !needsInput && sess.NeedsInput {
			sess.SetNeedsInput(false)
			m.store.Update(sess)
		} else if outputChanged {
			m.store.Update(sess)
		}
	}
}

func (m *Monitor) maybeNotify(sess *session.Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	debounce := time.Duration(m.cfg.Monitor.DebounceSecs) * time.Second
	lastTime, exists := m.lastNotify[sess.ID]
	if exists && time.Since(lastTime) < debounce {
		return
	}

	m.lastNotify[sess.ID] = time.Now()

	if m.cfg.Notification.Desktop {
		notify.Desktop("gclaude: Input Required",
			"Branch '"+sess.Branch+"' is waiting for input")
	}

	if m.cfg.Notification.Sound {
		notify.Sound(m.cfg.Notification.SoundFile)
	}
}

func getLastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
