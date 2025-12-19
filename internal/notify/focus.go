package notify

import (
	"os/exec"
	"strings"
)

// IsTerminalFocused checks if the terminal containing a tmux session is the focused window
func IsTerminalFocused(tty string) bool {
	if tty == "" {
		return false
	}

	// Get the active window PID using xdotool
	activeWinCmd := exec.Command("xdotool", "getactivewindow", "getwindowpid")
	activeOut, err := activeWinCmd.Output()
	if err != nil {
		// xdotool not available or failed - can't determine focus
		return false
	}
	activeWinPID := strings.TrimSpace(string(activeOut))
	if activeWinPID == "" {
		return false
	}

	// Find the process that owns the TTY (the terminal emulator)
	// We look for the process group leader of the TTY
	psCmd := exec.Command("ps", "-o", "pid=", "-t", strings.TrimPrefix(tty, "/dev/"))
	psOut, err := psCmd.Output()
	if err != nil {
		return false
	}

	// Get all PIDs using this TTY
	pids := strings.Fields(string(psOut))

	// For each PID, walk up to find if it matches the active window's process tree
	for _, pid := range pids {
		if isProcessInTree(pid, activeWinPID) {
			return true
		}
	}

	// Also check if the active window's process tree contains any of our TTY processes
	// by walking up from TTY processes to see if they share a common ancestor with active window
	for _, pid := range pids {
		if sharesTerminalAncestor(pid, activeWinPID) {
			return true
		}
	}

	return false
}

// isProcessInTree checks if pid is in the process tree rooted at rootPid
func isProcessInTree(pid, rootPid string) bool {
	current := pid
	for i := 0; i < 20; i++ { // Max depth
		if current == rootPid {
			return true
		}
		if current == "" || current == "1" || current == "0" {
			break
		}
		// Get parent PID
		ppidCmd := exec.Command("ps", "-o", "ppid=", "-p", current)
		ppidOut, err := ppidCmd.Output()
		if err != nil {
			break
		}
		current = strings.TrimSpace(string(ppidOut))
	}
	return false
}

// sharesTerminalAncestor checks if two PIDs share a common terminal emulator ancestor
func sharesTerminalAncestor(pid1, pid2 string) bool {
	// Get ancestors of pid1
	ancestors1 := getAncestors(pid1)
	ancestors2 := getAncestors(pid2)

	// Check for common ancestor (excluding init/systemd)
	for _, a1 := range ancestors1 {
		for _, a2 := range ancestors2 {
			if a1 == a2 && a1 != "1" && a1 != "0" {
				return true
			}
		}
	}
	return false
}

func getAncestors(pid string) []string {
	var ancestors []string
	current := pid
	for i := 0; i < 20; i++ {
		if current == "" || current == "1" || current == "0" {
			break
		}
		ancestors = append(ancestors, current)
		ppidCmd := exec.Command("ps", "-o", "ppid=", "-p", current)
		ppidOut, err := ppidCmd.Output()
		if err != nil {
			break
		}
		current = strings.TrimSpace(string(ppidOut))
	}
	return ancestors
}
