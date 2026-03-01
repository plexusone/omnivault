package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/grokify/oscompat/process"

	"github.com/plexusone/omnivault/internal/client"
	"github.com/plexusone/omnivault/internal/config"
	"github.com/plexusone/omnivault/internal/daemon"
)

func cmdDaemon(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: omnivault daemon <start|stop|status|run>")
	}

	subcmd := args[0]

	switch subcmd {
	case "start":
		return daemonStart()
	case "stop":
		return daemonStop()
	case "status":
		return daemonStatus()
	case "run":
		return daemonRun()
	default:
		return fmt.Errorf("unknown daemon command: %s", subcmd)
	}
}

func daemonStart() error {
	c := client.New()

	if c.IsDaemonRunning() {
		fmt.Println("Daemon is already running")
		return nil
	}

	// Start daemon in background
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command(exe, "daemon", "run")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach from parent process (cross-platform via oscompat)
	process.SetDetached(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Printf("Daemon started (PID: %d)\n", cmd.Process.Pid)

	// Don't wait for the child process - it's intentionally detached.
	// The error from Wait() is not meaningful for a daemon we don't manage.
	go func() { _ = cmd.Wait() }()

	return nil
}

func daemonStop() error {
	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		fmt.Println("Daemon is not running")
		return nil
	}

	// Try graceful stop via API
	if err := c.Stop(ctx); err != nil {
		// If API fails, try to kill by PID
		return killDaemonByPID()
	}

	fmt.Println("Daemon stopped")
	return nil
}

func daemonStatus() error {
	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		fmt.Println("Daemon: not running")
		return nil
	}

	status, err := c.GetStatus(ctx)
	if err != nil {
		fmt.Println("Daemon: running (status unavailable)")
		return nil
	}

	fmt.Println("Daemon: running")
	fmt.Printf("Uptime: %s\n", status.Uptime)

	if status.VaultExists {
		if status.Locked {
			fmt.Println("Vault: locked")
		} else {
			fmt.Println("Vault: unlocked")
			fmt.Printf("Secrets: %d\n", status.SecretCount)
		}
	} else {
		fmt.Println("Vault: not initialized")
	}

	return nil
}

func daemonRun() error {
	// Run daemon in foreground
	fmt.Println("Starting OmniVault daemon...")

	server := daemon.NewServer(daemon.ServerConfig{})

	ctx := context.Background()
	return server.Run(ctx)
}

func killDaemonByPID() error {
	paths := config.GetPaths()

	data, err := os.ReadFile(paths.PIDFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	if err := process.Signal(pid); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	// Cleanup PID file
	if err := os.Remove(paths.PIDFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	fmt.Println("Daemon stopped")
	return nil
}
