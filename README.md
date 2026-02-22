# OmniVault

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

OmniVault is a unified Go library for secret management across multiple providers. It provides a single interface for accessing secrets from password managers, cloud secret managers, enterprise vaults, and local storage.

## Features

- 🔗 **Unified Interface**: Single API for all secret storage backends
- 🧩 **Extensible Architecture**: Add custom providers as separate Go modules without modifying the core library
- 🌐 **URI-Based Resolution**: Reference secrets using URIs like `op://vault/item/field` or `aws-sm://secret-name`
- 📦 **Built-in Providers**: Environment variables, file-based, and in-memory storage included
- ⚡ **Zero External Dependencies**: Core library has no external dependencies beyond the standard library
- 💻 **CLI Tool**: Command-line interface with encrypted local storage and daemon architecture
- 🔐 **Secure Local Storage**: AES-256-GCM encryption with Argon2id key derivation

## Installation

### Go Library

```bash
go get github.com/agentplexus/omnivault
```

### CLI Tool

```bash
go install github.com/agentplexus/omnivault/cmd/omnivault@latest
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/agentplexus/omnivault"
)

func main() {
    ctx := context.Background()

    // Create client with environment variable provider
    client, err := omnivault.NewClient(omnivault.Config{
        Provider: omnivault.ProviderEnv,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get a secret
    secret, err := client.Get(ctx, "API_KEY")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("API Key:", secret.Value)
}
```

### Multi-Provider Resolution

```go
package main

import (
    "context"
    "fmt"

    "github.com/agentplexus/omnivault"
    "github.com/agentplexus/omnivault/providers/env"
    "github.com/agentplexus/omnivault/providers/memory"
)

func main() {
    ctx := context.Background()

    // Create resolver with multiple providers
    resolver := omnivault.NewResolver()
    resolver.Register("env", env.New())
    resolver.Register("memory", memory.NewWithSecrets(map[string]string{
        "database/password": "secret123",
    }))

    // Resolve secrets from different providers using URIs
    apiKey, _ := resolver.Resolve(ctx, "env://API_KEY")
    dbPass, _ := resolver.Resolve(ctx, "memory://database/password")

    fmt.Println("API Key:", apiKey)
    fmt.Println("DB Password:", dbPass)
}
```

### Using Official Provider Modules

```go
package main

import (
    "context"
    "fmt"

    "github.com/agentplexus/omnivault"
    aws "github.com/agentplexus/omnivault-aws"
    "github.com/agentplexus/omnivault-keyring"
)

func main() {
    ctx := context.Background()

    // Create providers
    awsProvider, _ := aws.NewSecretsManager(aws.Config{Region: "us-east-1"})
    keyringProvider := keyring.New(keyring.Config{ServiceName: "myapp"})

    // Multi-provider resolver
    resolver := omnivault.NewResolver()
    resolver.Register("aws-sm", awsProvider)
    resolver.Register("keyring", keyringProvider)

    // Resolve from AWS Secrets Manager
    dbCreds, _ := resolver.Resolve(ctx, "aws-sm://prod/database")

    // Resolve from OS keyring
    localToken, _ := resolver.Resolve(ctx, "keyring://dev/api-token")

    fmt.Println("DB Credentials:", dbCreds)
    fmt.Println("Local Token:", localToken)
}
```

## Supported Providers

### Built-in Providers

| Provider | Scheme | Description |
|----------|--------|-------------|
| Environment Variables | `env://` | Read from `os.Getenv()` |
| File | `file://` | File-based storage |
| Memory | `memory://` | In-memory storage (for testing) |

### Official Provider Modules

First-party provider modules maintained alongside OmniVault:

| Module | Providers | Schemes |
|--------|-----------|---------|
| [omnivault-aws](https://github.com/agentplexus/omnivault-aws) | AWS Secrets Manager, AWS Parameter Store | `aws-sm://`, `aws-ssm://` |
| [omnivault-keyring](https://github.com/agentplexus/omnivault-keyring) | macOS Keychain, Windows Credential Manager, Linux Secret Service | `keyring://` |

```bash
# Install official provider modules
go get github.com/agentplexus/omnivault-aws
go get github.com/agentplexus/omnivault-keyring
```

### Community Providers

External providers can be developed as separate Go modules and injected via `Config.CustomVault`:

| Category | Providers |
|----------|-----------|
| **Password Managers** | 1Password, Bitwarden, LastPass, KeePass, pass/gopass |
| **Cloud Secret Managers** | GCP Secret Manager, Azure Key Vault |
| **Enterprise Vaults** | HashiCorp Vault, CyberArk Conjur, Akeyless, Doppler |

## Creating Custom Providers

Custom providers can be developed as separate Go modules. Import only the `vault` package to avoid pulling in unnecessary dependencies:

```go
package myprovider

import (
    "context"
    "github.com/agentplexus/omnivault/vault"
)

type Provider struct {
    // Your provider fields
}

// New creates a new provider instance.
func New(apiKey string) vault.Vault {
    return &Provider{/* ... */}
}

// Implement vault.Vault interface
func (p *Provider) Get(ctx context.Context, path string) (*vault.Secret, error) {
    // Your implementation
}

func (p *Provider) Set(ctx context.Context, path string, secret *vault.Secret) error {
    // Your implementation
}

func (p *Provider) Delete(ctx context.Context, path string) error {
    // Your implementation
}

func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
    // Your implementation
}

func (p *Provider) List(ctx context.Context, prefix string) ([]string, error) {
    // Your implementation
}

func (p *Provider) Name() string {
    return "myprovider"
}

func (p *Provider) Capabilities() vault.Capabilities {
    return vault.Capabilities{
        Read:  true,
        Write: true,
        // ...
    }
}

func (p *Provider) Close() error {
    return nil
}
```

Then use it with OmniVault:

```go
import (
    "github.com/agentplexus/omnivault"
    "github.com/yourorg/omnivault-myprovider"
)

client, _ := omnivault.NewClient(omnivault.Config{
    CustomVault: myprovider.New("api-key"),
})
```

## URI Scheme Reference

```
scheme://path[#field]

# Examples:
env://API_KEY                    # Environment variable
file:///path/to/secret           # File
memory://database/password       # In-memory

# External providers (when installed):
op://vault/item/field            # 1Password
keychain://service/account       # macOS Keychain
aws-sm://secret-name#key         # AWS Secrets Manager
gcp-sm://project/secret          # GCP Secret Manager
azure-kv://vault/secret          # Azure Key Vault
vault://secret/path#field        # HashiCorp Vault
```

## API Reference

### Client

```go
// Create a new client
client, err := omnivault.NewClient(omnivault.Config{
    Provider:    omnivault.ProviderEnv,  // Built-in provider
    CustomVault: myVault,                 // OR custom provider
})

// Basic operations
secret, err := client.Get(ctx, "path")
err := client.Set(ctx, "path", &omnivault.Secret{Value: "secret"})
err := client.Delete(ctx, "path")
exists, err := client.Exists(ctx, "path")
paths, err := client.List(ctx, "prefix")

// Convenience methods
value, err := client.GetValue(ctx, "path")      // Returns just the value
value, err := client.GetField(ctx, "path", "field")  // Returns a specific field
err := client.SetValue(ctx, "path", "value")    // Set a simple string value

// Must variants (panic on error)
secret := client.MustGet(ctx, "path")
value := client.MustGetValue(ctx, "path")
```

### Resolver

```go
// Create resolver
resolver := omnivault.NewResolver()

// Register providers
resolver.Register("env", envVault)
resolver.Register("op", onePasswordVault)

// Resolve secrets
value, err := resolver.Resolve(ctx, "env://API_KEY")
secret, err := resolver.ResolveSecret(ctx, "op://vault/item")

// Resolve if it's a secret reference, otherwise return as-is
value, err := resolver.ResolveString(ctx, maybeSecretRef)

// Resolve all values in a map
resolved, err := resolver.ResolveMap(ctx, configMap)
```

### Secret

```go
// Create secrets
secret := &omnivault.Secret{
    Value: "my-secret-value",
    Fields: map[string]string{
        "username": "admin",
        "password": "secret",
    },
    Metadata: omnivault.Metadata{
        Tags: map[string]string{"env": "prod"},
    },
}

// Access values
value := secret.String()           // Primary value
value := secret.GetField("username") // Specific field
bytes := secret.Bytes()            // As bytes
```

## Package Structure

```
omnivault/
├── vault/              # Core interface (import this for custom providers)
│   ├── interface.go    # Vault interface definition
│   ├── types.go        # Secret, Metadata, SecretRef types
│   └── errors.go       # Standard errors
├── providers/          # Built-in providers
│   ├── env/            # Environment variables
│   ├── file/           # File-based storage
│   └── memory/         # In-memory storage
├── client.go           # Main client
├── resolver.go         # URI-based resolution
├── providers.go        # Provider factory
├── constants.go        # Provider names
├── errors.go           # Client errors
└── types.go            # Type aliases
```

## External Provider Modules

To create an external provider module:

1. Create a new Go module (e.g., `github.com/yourorg/omnivault-onepassword`)
2. Import only `github.com/agentplexus/omnivault/vault`
3. Implement the `vault.Vault` interface
4. Export a constructor function returning `vault.Vault`

This architecture ensures:
- External providers don't bloat the core library with dependencies
- Providers can be versioned independently
- Users only install the providers they need

---

## CLI Tool

The `omnivault` CLI provides secure local secret management with a daemon architecture.

### CLI Quick Start

```bash
# Start the daemon
omnivault daemon start

# Initialize a new vault with a master password
omnivault init

# Store a secret
omnivault set database/password

# Retrieve a secret
omnivault get database/password

# List all secrets
omnivault list

# Lock the vault
omnivault lock

# Unlock the vault
omnivault unlock

# Check status
omnivault status
```

### CLI Commands

#### Vault Commands

| Command | Description |
|---------|-------------|
| `omnivault init` | Initialize a new vault with a master password |
| `omnivault unlock` | Unlock the vault with master password |
| `omnivault lock` | Lock the vault |
| `omnivault status` | Show vault and daemon status |

#### Secret Commands

| Command | Description |
|---------|-------------|
| `omnivault get <path>` | Get a secret value |
| `omnivault set <path> [value]` | Set a secret (prompts for value if not provided) |
| `omnivault list [prefix]` | List secrets, optionally filtered by prefix |
| `omnivault delete <path>` | Delete a secret (with confirmation) |

#### Daemon Commands

| Command | Description |
|---------|-------------|
| `omnivault daemon start` | Start the daemon in background |
| `omnivault daemon stop` | Stop the daemon |
| `omnivault daemon status` | Show daemon status |
| `omnivault daemon run` | Run daemon in foreground (for debugging) |

### Daemon Architecture

The CLI uses a daemon (background service) architecture for secure secret access:

- **Cross-Platform IPC**: Unix socket on macOS/Linux, TCP localhost on Windows
- **Session-Based Unlock**: Vault stays unlocked until locked or timeout
- **Auto-Lock**: Configurable inactivity timeout (default: 15 minutes)
- **Graceful Shutdown**: Vault is locked on daemon shutdown

#### Platform Support

| Platform | IPC Method | Socket/Address |
|----------|------------|----------------|
| macOS | Unix Socket | `~/.omnivault/omnivaultd.sock` |
| Linux | Unix Socket | `~/.omnivault/omnivaultd.sock` |
| Windows | TCP | `127.0.0.1:19839` |

### Security Model

#### Encryption

- **Algorithm**: AES-256-GCM (authenticated encryption)
- **Key Derivation**: Argon2id (memory-hard, resistant to GPU attacks)
  - 3 iterations
  - 64 MB memory
  - 4 parallel threads
- **Salt**: Random 32 bytes per vault
- **Nonce**: Random 12 bytes per secret

#### Storage

**macOS / Linux:**

```
~/.omnivault/
├── vault.enc           # Encrypted secrets (AES-256-GCM)
├── vault.meta          # Unencrypted metadata (salt, Argon2 params)
├── omnivaultd.sock     # Unix socket (runtime)
└── omnivaultd.pid      # Daemon PID file (runtime)
```

**Windows:**

```
%LOCALAPPDATA%\OmniVault\
├── vault.enc           # Encrypted secrets (AES-256-GCM)
├── vault.meta          # Unencrypted metadata (salt, Argon2 params)
└── omnivaultd.pid      # Daemon PID file (runtime)
```

#### Master Password

- Never stored on disk
- Used only to derive the encryption key
- Minimum 8 characters enforced
- Session-based unlock with configurable timeout

## Contributing

Contributions are welcome! Please submit pull requests or create issues for bugs and feature requests.

## License

MIT License - see [LICENSE](LICENSE) for details.

 [build-status-svg]: https://github.com/agentplexus/omnivault/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/agentplexus/omnivault/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/agentplexus/omnivault/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/agentplexus/omnivault/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/agentplexus/omnivault
 [goreport-url]: https://goreportcard.com/report/github.com/agentplexus/omnivault
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/agentplexus/omnivault
 [docs-godoc-url]: https://pkg.go.dev/github.com/agentplexus/omnivault
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/agentplexus/omnivault/blob/master/LICENSE
 [used-by-svg]: https://sourcegraph.com/github.com/agentplexus/omnivault/-/badge.svg
 [used-by-url]: https://sourcegraph.com/github.com/agentplexus/omnivault?badge
