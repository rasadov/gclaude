package notify

import (
	"os/exec"
)

func Desktop(title, message string) error {
	cmd := exec.Command("notify-send", "-a", "gclaude", "-u", "normal", title, message)
	return cmd.Run()
}

func DesktopUrgent(title, message string) error {
	cmd := exec.Command("notify-send", "-a", "gclaude", "-u", "critical", title, message)
	return cmd.Run()
}
