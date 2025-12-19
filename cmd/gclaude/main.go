package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/bb/gclaude/internal/config"
	"github.com/bb/gclaude/internal/monitor"
	"github.com/bb/gclaude/internal/session"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "gclaude",
	Short: "Multi-branch Claude Code session manager",
	Long: `gclaude simplifies working with Claude Code across multiple git branches.

Features:
  - Automatic git worktree creation for parallel branch work
  - Session management via tmux (persistent sessions)
  - Desktop notifications when Claude needs input
  - Sound alerts for attention`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(configCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gclaude %s\n", version)
	},
}

var (
	startNoWorktree bool
	startPrompt     string
)

var startCmd = &cobra.Command{
	Use:   "start <branch>",
	Short: "Start a new Claude session on a branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		mgr := session.NewManager()
		sess, err := mgr.Start(branch, cwd, !startNoWorktree)
		if err != nil {
			return err
		}

		fmt.Printf("Started session for branch '%s'\n", sess.Branch)
		fmt.Printf("  Worktree: %s\n", sess.WorktreePath)
		fmt.Printf("  tmux: %s\n", sess.TmuxSession)

		cfg, _ := config.Load()
		mon := monitor.New(mgr.GetStore(), cfg)
		mon.Start()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		fmt.Println("\nMonitoring for input prompts. Press Ctrl+C to stop monitoring (session continues).")
		fmt.Printf("Use 'gclaude attach %s' to attach to the session.\n", branch)

		<-sigChan
		mon.Stop()
		fmt.Println("\nMonitoring stopped. Session continues in background.")

		return nil
	},
}

func init() {
	startCmd.Flags().BoolVar(&startNoWorktree, "no-worktree", false, "Don't create a worktree, use current directory")
	startCmd.Flags().StringVarP(&startPrompt, "prompt", "p", "", "Initial prompt to send to Claude")
}

var (
	stopAll            bool
	stopRemoveWorktree bool
)

var stopCmd = &cobra.Command{
	Use:   "stop [branch]",
	Short: "Stop a Claude session",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := session.NewManager()

		if stopAll {
			if err := mgr.StopAll(stopRemoveWorktree); err != nil {
				return err
			}
			fmt.Println("All sessions stopped")
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("branch name required (or use --all)")
		}

		if err := mgr.Stop(args[0], stopRemoveWorktree); err != nil {
			return err
		}
		fmt.Printf("Session for branch '%s' stopped\n", args[0])
		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVar(&stopAll, "all", false, "Stop all sessions")
	stopCmd.Flags().BoolVar(&stopRemoveWorktree, "remove-worktree", false, "Also remove the worktree")
}

var attachCmd = &cobra.Command{
	Use:     "attach <branch>",
	Aliases: []string{"a"},
	Short:   "Attach to a running session",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := session.NewManager()
		return mgr.Attach(args[0])
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := session.NewManager()
		sessions := mgr.List()

		if len(sessions) == 0 {
			fmt.Println("No active sessions")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "BRANCH\tSTATUS\tWORKTREE\tLAST ACTIVITY")
		fmt.Fprintln(w, strings.Repeat("-", 80))

		for _, sess := range sessions {
			status := string(sess.Status)
			if sess.NeedsInput {
				status = "⚠ " + status
			}

			lastActivity := sess.LastActivity.Format(time.RFC3339)
			if time.Since(sess.LastActivity) < time.Hour {
				lastActivity = time.Since(sess.LastActivity).Round(time.Second).String() + " ago"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				sess.Branch,
				status,
				truncatePath(sess.WorktreePath, 40),
				lastActivity,
			)
		}

		w.Flush()
		return nil
	},
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove stale sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := session.NewManager()
		removed, err := mgr.Cleanup()
		if err != nil {
			return err
		}

		if removed == 0 {
			fmt.Println("No stale sessions found")
		} else {
			fmt.Printf("Removed %d stale session(s)\n", removed)
		}
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Printf("notification.desktop: %v\n", cfg.Notification.Desktop)
		fmt.Printf("notification.sound: %v\n", cfg.Notification.Sound)
		fmt.Printf("notification.sound_file: %s\n", cfg.Notification.SoundFile)
		fmt.Printf("monitor.poll_interval_ms: %d\n", cfg.Monitor.PollIntervalMs)
		fmt.Printf("monitor.idle_threshold_s: %d\n", cfg.Monitor.IdleThresholdS)
		fmt.Printf("monitor.debounce_secs: %d\n", cfg.Monitor.DebounceSecs)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		switch key {
		case "notification.desktop":
			cfg.Notification.Desktop = value == "true"
		case "notification.sound":
			cfg.Notification.Sound = value == "true"
		case "notification.sound_file":
			cfg.Notification.SoundFile = value
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}
