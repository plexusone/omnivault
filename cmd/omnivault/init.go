package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/plexusone/omnivault/internal/client"
	"golang.org/x/term"
)

func cmdInit(_ []string) error {
	c := client.New()
	ctx := context.Background()

	// Check if daemon is running
	if !c.IsDaemonRunning() {
		return fmt.Errorf("daemon is not running, start it with: omnivault daemon start")
	}

	// Check if vault already exists
	status, err := c.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.VaultExists {
		return fmt.Errorf("vault already exists")
	}

	// Prompt for password
	fmt.Print("Enter master password (min 8 chars): ")
	password, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	fmt.Print("Confirm master password: ")
	confirm, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	// Initialize vault
	if err := c.Init(ctx, password); err != nil {
		return fmt.Errorf("failed to initialize vault: %w", err)
	}

	fmt.Println("Vault initialized successfully!")
	fmt.Println("Your vault is now unlocked and ready to use.")
	return nil
}

func cmdUnlock(_ []string) error {
	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		return fmt.Errorf("daemon is not running, start it with: omnivault daemon start")
	}

	status, err := c.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if !status.VaultExists {
		return fmt.Errorf("vault does not exist, run: omnivault init")
	}

	if !status.Locked {
		fmt.Println("Vault is already unlocked")
		return nil
	}

	fmt.Print("Enter master password: ")
	password, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if err := c.Unlock(ctx, password); err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	fmt.Println("Vault unlocked successfully!")
	return nil
}

func cmdLock(_ []string) error {
	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		return fmt.Errorf("daemon is not running")
	}

	if err := c.Lock(ctx); err != nil {
		return fmt.Errorf("failed to lock: %w", err)
	}

	fmt.Println("Vault locked")
	return nil
}

func cmdStatus(_ []string) error {
	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		fmt.Println("Daemon: not running")
		return nil
	}

	status, err := c.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Println("Daemon: running")
	fmt.Printf("Uptime: %s\n", status.Uptime)

	if !status.VaultExists {
		fmt.Println("Vault: not initialized")
		return nil
	}

	if status.Locked {
		fmt.Println("Vault: locked")
	} else {
		fmt.Println("Vault: unlocked")
		fmt.Printf("Secrets: %d\n", status.SecretCount)
		if !status.UnlockedAt.IsZero() {
			fmt.Printf("Unlocked at: %s\n", status.UnlockedAt.Format("2006-01-02 15:04:05"))
		}
	}

	return nil
}

// readPassword reads a password from the terminal without echo.
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd()) //nolint:gosec // G115: Fd() returns small values, overflow not possible

	// Try to read without echo
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		fmt.Println() // Print newline after password
		if err != nil {
			return "", err
		}
		return string(password), nil
	}

	// Fallback for non-terminal (e.g., piped input)
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(password), nil
}
