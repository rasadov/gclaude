package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func SessionExists(name string) (bool, error) {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

func CreateSession(name, workDir, command string) error {
	args := []string{"new-session", "-d", "-s", name, "-c", workDir}
	if command != "" {
		args = append(args, command)
	}
	cmd := exec.Command("tmux", args...)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Enable mouse support for scrolling
	SetOption(name, "mouse", "on")

	return nil
}

func SetOption(sessionName, option, value string) error {
	cmd := exec.Command("tmux", "set-option", "-t", sessionName, option, value)
	return cmd.Run()
}

func KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

func AttachSession(name string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CapturePane(sessionName string, lines int) (string, error) {
	startLine := fmt.Sprintf("-%d", lines)
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-S", startLine)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func SendKeys(sessionName, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, keys, "Enter")
	return cmd.Run()
}

func ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return []string{}, nil
			}
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

func IsRunning() bool {
	cmd := exec.Command("tmux", "list-sessions")
	return cmd.Run() == nil
}

func GetPanePid(sessionName string) (string, error) {
	cmd := exec.Command("tmux", "display-message", "-t", sessionName, "-p", "#{pane_pid}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func IsSessionAttached(sessionName string) bool {
	cmd := exec.Command("tmux", "list-clients", "-t", sessionName, "-F", "#{client_tty}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	// If there's any output, a client is attached
	return strings.TrimSpace(out.String()) != ""
}

func GetAttachedClientTTY(sessionName string) string {
	cmd := exec.Command("tmux", "list-clients", "-t", sessionName, "-F", "#{client_tty}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

// GetClientLastActivity returns seconds since last client input activity
func GetClientLastActivity(sessionName string) int {
	// Get client_activity (Unix timestamp of last activity)
	cmd := exec.Command("tmux", "list-clients", "-t", sessionName, "-F", "#{client_activity}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return -1
	}

	activityStr := strings.TrimSpace(out.String())
	if activityStr == "" {
		return -1
	}

	// Parse the timestamp
	var activityTime int64
	if _, err := fmt.Sscanf(activityStr, "%d", &activityTime); err != nil {
		return -1
	}

	// Calculate seconds since last activity
	now := time.Now().Unix()
	return int(now - activityTime)
}

// HasRecentInput returns true if user has typed something in the last N seconds
func HasRecentInput(sessionName string, withinSeconds int) bool {
	lastActivity := GetClientLastActivity(sessionName)
	if lastActivity < 0 {
		return false
	}
	return lastActivity <= withinSeconds
}
