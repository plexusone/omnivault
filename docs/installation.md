# Installation

## Go Library

Add OmniVault to your Go project:

```bash
go get github.com/agentplexus/omnivault
```

### Requirements

- Go 1.22.0 or later

### Optional Provider Modules

Install official provider modules as needed:

```bash
# AWS Secrets Manager and Parameter Store
go get github.com/agentplexus/omnivault-aws

# OS Keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
go get github.com/agentplexus/omnivault-keyring
```

## CLI Tool

Install the `omnivault` command-line tool:

```bash
go install github.com/agentplexus/omnivault/cmd/omnivault@latest
```

### Verify Installation

```bash
omnivault version
```

### Requirements

- Go 1.24.0 or later (for building)
- macOS, Linux, or Windows

### Platform-Specific Notes

=== "macOS / Linux"

    The daemon communicates via Unix socket at `~/.omnivault/omnivaultd.sock`.

=== "Windows"

    Currently uses TCP on localhost. Named pipe support is planned.

## Building from Source

Clone the repository and build:

```bash
git clone https://github.com/agentplexus/omnivault.git
cd omnivault

# Build the CLI
go build -o omnivault ./cmd/omnivault

# Run tests
go test -v ./...
```
