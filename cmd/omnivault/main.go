// Package main provides the omnivault CLI.
package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "init":
		err = cmdInit(args)
	case "unlock":
		err = cmdUnlock(args)
	case "lock":
		err = cmdLock(args)
	case "status":
		err = cmdStatus(args)
	case "get":
		err = cmdGet(args)
	case "set":
		err = cmdSet(args)
	case "list", "ls":
		err = cmdList(args)
	case "delete", "rm":
		err = cmdDelete(args)
	case "daemon":
		err = cmdDaemon(args)
	case "version":
		fmt.Printf("omnivault version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd) //nolint:gosec // G705: stderr output, not HTML
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`omnivault - Secure local secret management

Usage:
  omnivault <command> [arguments]

Vault Commands:
  init              Initialize a new vault with a master password
  unlock            Unlock the vault
  lock              Lock the vault
  status            Show vault and daemon status

Secret Commands:
  get <path>        Get a secret value
  set <path> [val]  Set a secret (prompts for value if not provided)
  list [prefix]     List secrets
  delete <path>     Delete a secret

Daemon Commands:
  daemon start      Start the daemon in background
  daemon stop       Stop the daemon
  daemon status     Show daemon status
  daemon run        Run daemon in foreground (for debugging)

Other Commands:
  version           Show version
  help              Show this help

Examples:
  omnivault init
  omnivault set database/password
  omnivault get database/password
  omnivault list database/
  omnivault delete database/password`)
}
