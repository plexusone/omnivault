# OmniVault

[![Build Status](https://github.com/agentplexus/omnivault/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/agentplexus/omnivault/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agentplexus/omnivault)](https://goreportcard.com/report/github.com/agentplexus/omnivault)
[![GoDoc](https://pkg.go.dev/badge/github.com/agentplexus/omnivault)](https://pkg.go.dev/github.com/agentplexus/omnivault)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/agentplexus/omnivault/blob/master/LICENSE)

OmniVault is a unified Go library for secret management across multiple providers. It provides a single interface for accessing secrets from password managers, cloud secret managers, enterprise vaults, and local storage.

## Features

- **Unified Interface** - Single API for all secret storage backends
- **Extensible Architecture** - Add custom providers as separate Go modules without modifying the core library
- **URI-Based Resolution** - Reference secrets using URIs like `op://vault/item/field` or `aws-sm://secret-name`
- **Built-in Providers** - Environment variables, file-based, and in-memory storage included
- **Zero External Dependencies** - Core library has no external dependencies beyond the standard library
- **CLI Tool** - Command-line interface with encrypted local storage and daemon architecture
- **Secure Local Storage** - AES-256-GCM encryption with Argon2id key derivation

## Two Ways to Use OmniVault

### As a Go Library

Import OmniVault into your Go application to access secrets from multiple providers:

```go
import "github.com/agentplexus/omnivault"

client, _ := omnivault.NewClient(omnivault.Config{
    Provider: omnivault.ProviderEnv,
})

secret, _ := client.Get(ctx, "API_KEY")
fmt.Println(secret.Value)
```

[Get started with the library →](library/quickstart.md)

### As a CLI Tool

Use the `omnivault` command-line tool for secure local secret management:

```bash
omnivault daemon start
omnivault init
omnivault set database/password
omnivault get database/password
```

[Get started with the CLI →](cli/quickstart.md)

## Quick Links

| Section | Description |
|---------|-------------|
| [Installation](installation.md) | Install the library or CLI |
| [Library Quick Start](library/quickstart.md) | Get started with the Go library |
| [CLI Quick Start](cli/quickstart.md) | Get started with the CLI |
| [Providers](library/providers.md) | Available secret providers |
| [Security](cli/security.md) | CLI security model |
| [Changelog](changelog.md) | Release history |

## License

MIT License - see [LICENSE](https://github.com/agentplexus/omnivault/blob/master/LICENSE) for details.
