package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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
	return cmd.Run()
}

func KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

func AttachSession(name string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = nil
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
