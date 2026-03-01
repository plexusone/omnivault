---
marp: true
theme: agentplexus
paginate: true
---

<!-- _class: lead -->

# OmniVault

## Unified Secret Management for Go

A single interface for all your secret storage backends

---

# The Problem

Managing secrets across different environments is complex:

- **Development**: Environment variables, local files
- **Staging**: OS keychain, cloud secrets
- **Production**: AWS Secrets Manager, HashiCorp Vault

Each backend has its own API, authentication, and patterns.

---

# The Solution: OmniVault

**One interface, many backends**

```go
// Same code works everywhere
secret, err := client.Get(ctx, "database/password")
```

- Unified API across all providers
- URI-based secret references
- Zero external dependencies in core
- Extensible plugin architecture

---

# Architecture Overview

```
                    ┌─────────────────────────────────────┐
                    │           Your Application          │
                    └───────────────┬─────────────────────┘
                                    │
                    ┌───────────────▼─────────────────────┐
                    │     OmniVault Client / Resolver     │
                    └───────────────┬─────────────────────┘
                                    │
        ┌───────────┬───────────────┼───────────────┬───────────┐
        ▼           ▼               ▼               ▼           ▼
    ┌───────┐   ┌───────┐   ┌─────────────┐   ┌─────────┐   ┌───────┐
    │  Env  │   │ File  │   │ AWS Secrets │   │ Keyring │   │ Custom│
    └───────┘   └───────┘   └─────────────┘   └─────────┘   └───────┘
```

---

# Core Interface: `vault.Vault`

```go
type Vault interface {
    Get(ctx context.Context, path string) (*Secret, error)
    Set(ctx context.Context, path string, secret *Secret) error
    Delete(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) (bool, error)
    List(ctx context.Context, prefix string) ([]string, error)
    Name() string
    Capabilities() Capabilities
    Close() error
}
```

All providers implement this interface.

---

# The Secret Type

```go
type Secret struct {
    Value      string            // Primary secret value
    ValueBytes []byte            // Binary data (takes precedence)
    Fields     map[string]string // Multi-field secrets
    Metadata   Metadata          // Timestamps, tags, version info
}

// Usage
secret.String()              // Get primary value
secret.GetField("password")  // Get specific field
secret.Bytes()               // Get as bytes
```

---

# Capability Discovery

Providers declare what they support:

```go
type Capabilities struct {
    Read       bool  // Can retrieve secrets
    Write      bool  // Can store secrets
    Delete     bool  // Can remove secrets
    List       bool  // Can enumerate secrets
    Versioning bool  // Supports version history
    Rotation   bool  // Supports secret rotation
    Binary     bool  // Supports binary data
    MultiField bool  // Supports structured secrets
}
```

---

# Built-in Providers

| Provider | Scheme | Use Case |
|----------|--------|----------|
| **Environment** | `env://` | Local development, CI/CD |
| **File** | `file://` | Configuration files |
| **Memory** | `memory://` | Testing, caching |

```go
client, _ := omnivault.NewClient(omnivault.Config{
    Provider: omnivault.ProviderEnv,
})
```

Zero external dependencies!

---

# Official Provider Modules

### omnivault-aws

| Provider | Scheme | Description |
|----------|--------|-------------|
| Secrets Manager | `aws-sm://` | Credentials, API keys, rotation |
| Parameter Store | `aws-ssm://` | Config, feature flags, hierarchical |

### omnivault-keyring

| Platform | Backend |
|----------|---------|
| macOS | Keychain |
| Windows | Credential Manager |
| Linux | Secret Service (GNOME/KDE) |

---

# URI-Based Resolution

Reference secrets declaratively:

```
scheme://path[#field]
```

**Examples:**
```
env://API_KEY                    # Environment variable
file:///etc/secrets/db.json      # File
aws-sm://prod/database#password  # AWS Secrets Manager field
keyring://myapp/token            # OS Keyring
```

---

# Multi-Provider Resolver

```go
resolver := omnivault.NewResolver()

// Register multiple providers
resolver.Register("env", envProvider)
resolver.Register("aws-sm", awsProvider)
resolver.Register("keyring", keyringProvider)

// Resolve by URI scheme
apiKey, _ := resolver.Resolve(ctx, "env://API_KEY")
dbPass, _ := resolver.Resolve(ctx, "aws-sm://prod/db#password")
token, _ := resolver.Resolve(ctx, "keyring://myapp/token")
```

---

# Basic Usage Example

```go
package main

import (
    "context"
    "github.com/plexusone/omnivault"
)

func main() {
    ctx := context.Background()

    client, _ := omnivault.NewClient(omnivault.Config{
        Provider: omnivault.ProviderEnv,
    })
    defer client.Close()

    // Get secret
    value, _ := client.GetValue(ctx, "DATABASE_URL")

    // Set secret
    client.SetValue(ctx, "API_KEY", "secret-value")
}
```

---

# AWS Provider Example

```go
import (
    "github.com/plexusone/omnivault"
    aws "github.com/plexusone/omnivault-aws"
)

// Create AWS Secrets Manager provider
provider, _ := aws.NewSecretsManager(aws.Config{
    Region: "us-east-1",
})

client, _ := omnivault.NewClient(omnivault.Config{
    CustomVault: provider,
})

// Get structured secret (JSON parsed automatically)
secret, _ := client.Get(ctx, "prod/database")
username := secret.GetField("username")
password := secret.GetField("password")
```

---

# Keyring Provider Example

```go
import (
    "github.com/plexusone/omnivault"
    "github.com/plexusone/omnivault-keyring"
)

// Create OS keyring provider
provider := keyring.New(keyring.Config{
    ServiceName: "myapp",
    JSONFormat:  true,  // Enable multi-field support
})

client, _ := omnivault.NewClient(omnivault.Config{
    CustomVault: provider,
})

// Store in OS keychain/credential manager
client.SetValue(ctx, "api-token", "secret-token")
```

---

# Environment-Based Switching

```go
func getSecretProvider() vault.Vault {
    if os.Getenv("AWS_EXECUTION_ENV") != "" {
        // Running on AWS (EKS, Lambda, EC2)
        provider, _ := aws.NewSecretsManager(aws.Config{
            Region: "us-east-1",
        })
        return provider
    }

    // Local development - use OS keyring
    return keyring.New(keyring.Config{
        ServiceName: "myapp",
    })
}
```

Same code, different backends!

---

# Creating Custom Providers

```go
package myprovider

import "github.com/plexusone/omnivault/vault"

type Provider struct { /* fields */ }

func (p *Provider) Get(ctx context.Context, path string) (*vault.Secret, error) {
    // Your implementation
}

func (p *Provider) Set(ctx context.Context, path string, s *vault.Secret) error {
    // Your implementation
}

// Implement remaining interface methods...
```

Only import `vault` package - no transitive dependencies!

---

# Package Structure

```
omnivault/
├── vault/              # Core interfaces (import for custom providers)
│   ├── interface.go    # Vault, ExtendedVault, BatchVault
│   ├── types.go        # Secret, Metadata, Capabilities
│   └── errors.go       # Standard error types
├── providers/          # Built-in providers
│   ├── env/            # Environment variables
│   ├── file/           # File storage
│   └── memory/         # In-memory
├── client.go           # High-level client
└── resolver.go         # Multi-provider routing
```

---

# Error Handling

Consistent errors across all providers:

```go
import "github.com/plexusone/omnivault/vault"

secret, err := client.Get(ctx, "path")
if errors.Is(err, vault.ErrSecretNotFound) {
    // Handle missing secret
}
if errors.Is(err, vault.ErrAccessDenied) {
    // Handle permission error
}
```

Structured `VaultError` provides context (operation, path, provider).

---

# Extended Features

### Versioning (ExtendedVault)
```go
secret, _ := provider.GetVersion(ctx, "path", "v1")
versions, _ := provider.ListVersions(ctx, "path")
```

### Batch Operations (BatchVault)
```go
secrets, _ := provider.GetBatch(ctx, []string{"a", "b", "c"})
```

### Secret Rotation
```go
provider.Rotate(ctx, "path")  // Trigger rotation
```

---

# Design Principles

1. **Zero Dependencies** - Core library uses only standard library
2. **Plugin Architecture** - External providers as separate modules
3. **Interface-First** - Clean contracts enable interoperability
4. **Capability Discovery** - Adapt behavior based on provider features
5. **URI-Based Config** - Declarative secret references
6. **Thread Safety** - Safe concurrent access

---

<!-- _class: section-divider -->

# Summary

### OmniVault provides:

**Unified API** for all secret backends
**Built-in providers** for env, file, memory
**Official modules** for AWS and OS keyrings
**URI resolution** for declarative configuration
**Extensibility** through plugin architecture
**Zero dependencies** in core library

---

# Get Started

```bash
# Install core library
go get github.com/plexusone/omnivault

# Install official providers (as needed)
go get github.com/plexusone/omnivault-aws
go get github.com/plexusone/omnivault-keyring
```

**GitHub:** github.com/plexusone/omnivault

---

<!-- _class: lead -->

# Questions?

### Resources

Documentation: pkg.go.dev/github.com/plexusone/omnivault
Source: github.com/plexusone/omnivault
AWS Provider: github.com/plexusone/omnivault-aws
Keyring Provider: github.com/plexusone/omnivault-keyring
