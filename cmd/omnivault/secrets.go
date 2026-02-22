package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/agentplexus/omnivault/internal/client"
	"golang.org/x/term"
)

func cmdGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: omnivault get <path>")
	}

	path := args[0]
	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		return fmt.Errorf("daemon is not running, start it with: omnivault daemon start")
	}

	secret, err := c.GetSecret(ctx, path)
	if err != nil {
		return err
	}

	// Print value
	if secret.Value != "" {
		fmt.Println(secret.Value)
	}

	// Print fields if present
	if len(secret.Fields) > 0 {
		for k, v := range secret.Fields {
			fmt.Printf("%s: %s\n", k, v)
		}
	}

	return nil
}

func cmdSet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: omnivault set <path> [value]")
	}

	path := args[0]
	var value string

	if len(args) >= 2 {
		value = args[1]
	} else {
		// Prompt for value
		fmt.Print("Enter secret value: ")
		var err error
		fd := int(os.Stdin.Fd()) //nolint:gosec // G115: Fd() returns small values, overflow not possible
		if term.IsTerminal(fd) {
			// Read without echo for sensitive data
			bytes, err := term.ReadPassword(fd)
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read value: %w", err)
			}
			value = string(bytes)
		} else {
			reader := bufio.NewReader(os.Stdin)
			value, err = reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read value: %w", err)
			}
			value = strings.TrimSpace(value)
		}
	}

	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		return fmt.Errorf("daemon is not running, start it with: omnivault daemon start")
	}

	if err := c.SetSecret(ctx, path, value, nil, nil); err != nil {
		return err
	}

	fmt.Printf("Secret '%s' saved\n", path)
	return nil
}

func cmdList(args []string) error {
	prefix := ""
	if len(args) >= 1 {
		prefix = args[0]
	}

	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		return fmt.Errorf("daemon is not running, start it with: omnivault daemon start")
	}

	resp, err := c.ListSecrets(ctx, prefix)
	if err != nil {
		return err
	}

	if resp.Count == 0 {
		fmt.Println("No secrets found")
		return nil
	}

	for _, item := range resp.Secrets {
		typeIndicator := ""
		if item.HasValue && item.HasFields {
			typeIndicator = " (value+fields)"
		} else if item.HasFields {
			typeIndicator = " (fields)"
		}

		tagStr := ""
		if len(item.Tags) > 0 {
			tagStr = fmt.Sprintf(" [%s]", strings.Join(item.Tags, ", "))
		}

		fmt.Printf("%s%s%s\n", item.Path, typeIndicator, tagStr)
	}

	fmt.Printf("\n%d secret(s)\n", resp.Count)
	return nil
}

func cmdDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: omnivault delete <path>")
	}

	path := args[0]
	c := client.New()
	ctx := context.Background()

	if !c.IsDaemonRunning() {
		return fmt.Errorf("daemon is not running, start it with: omnivault daemon start")
	}

	// Confirm deletion
	fmt.Printf("Delete secret '%s'? [y/N]: ", path)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		fmt.Println("Cancelled")
		return nil
	}

	if err := c.DeleteSecret(ctx, path); err != nil {
		return err
	}

	fmt.Printf("Secret '%s' deleted\n", path)
	return nil
}
