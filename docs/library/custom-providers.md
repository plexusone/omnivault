# Custom Providers

Create your own secret provider by implementing the `vault.Vault` interface.

## Interface

```go
package vault

type Vault interface {
    // Get retrieves a secret by path
    Get(ctx context.Context, path string) (*Secret, error)

    // Set stores a secret at the given path
    Set(ctx context.Context, path string, secret *Secret) error

    // Delete removes a secret
    Delete(ctx context.Context, path string) error

    // Exists checks if a secret exists
    Exists(ctx context.Context, path string) (bool, error)

    // List returns paths matching the prefix
    List(ctx context.Context, prefix string) ([]string, error)

    // Name returns the provider name
    Name() string

    // Capabilities returns supported operations
    Capabilities() Capabilities

    // Close releases resources
    Close() error
}
```

## Creating a Provider Module

### 1. Create a new Go module

```bash
mkdir omnivault-myprovider
cd omnivault-myprovider
go mod init github.com/yourorg/omnivault-myprovider
```

### 2. Import only the vault package

```go
// Only import the interface package - no dependency on full omnivault
go get github.com/agentplexus/omnivault/vault
```

### 3. Implement the interface

```go
package myprovider

import (
    "context"
    "github.com/agentplexus/omnivault/vault"
)

type Provider struct {
    client *MyBackendClient
}

type Config struct {
    APIKey   string
    Endpoint string
}

func New(cfg Config) (vault.Vault, error) {
    client, err := newBackendClient(cfg.APIKey, cfg.Endpoint)
    if err != nil {
        return nil, err
    }
    return &Provider{client: client}, nil
}

func (p *Provider) Get(ctx context.Context, path string) (*vault.Secret, error) {
    data, err := p.client.Fetch(ctx, path)
    if err != nil {
        if isNotFound(err) {
            return nil, vault.ErrSecretNotFound
        }
        return nil, err
    }

    return &vault.Secret{
        Value: data.Value,
        Fields: data.Fields,
        Metadata: vault.Metadata{
            CreatedAt: data.CreatedAt,
        },
    }, nil
}

func (p *Provider) Set(ctx context.Context, path string, secret *vault.Secret) error {
    return p.client.Store(ctx, path, secret.Value, secret.Fields)
}

func (p *Provider) Delete(ctx context.Context, path string) error {
    return p.client.Remove(ctx, path)
}

func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
    _, err := p.client.Fetch(ctx, path)
    if err != nil {
        if isNotFound(err) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (p *Provider) List(ctx context.Context, prefix string) ([]string, error) {
    return p.client.ListKeys(ctx, prefix)
}

func (p *Provider) Name() string {
    return "myprovider"
}

func (p *Provider) Capabilities() vault.Capabilities {
    return vault.Capabilities{
        Read:       true,
        Write:      true,
        Delete:     true,
        List:       true,
        Binary:     false,
        MultiField: true,
    }
}

func (p *Provider) Close() error {
    return p.client.Close()
}
```

## Using Your Provider

### With Client

```go
import (
    "github.com/agentplexus/omnivault"
    "github.com/yourorg/omnivault-myprovider"
)

provider, _ := myprovider.New(myprovider.Config{
    APIKey: "...",
})

client, _ := omnivault.NewClient(omnivault.Config{
    CustomVault: provider,
})

secret, _ := client.Get(ctx, "my/secret")
```

### With Resolver

```go
resolver := omnivault.NewResolver()
resolver.Register("myprovider", provider)

value, _ := resolver.Resolve(ctx, "myprovider://my/secret")
```

## Best Practices

### Error Handling

Return standard errors when applicable:

```go
import "github.com/agentplexus/omnivault/vault"

// Use standard errors
if notFound {
    return nil, vault.ErrSecretNotFound
}
if accessDenied {
    return nil, vault.ErrAccessDenied
}

// Wrap backend errors
return nil, fmt.Errorf("backend error: %w", err)
```

### Context Handling

Respect context cancellation:

```go
func (p *Provider) Get(ctx context.Context, path string) (*vault.Secret, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Proceed with operation...
}
```

### Resource Management

Implement proper cleanup:

```go
func (p *Provider) Close() error {
    if p.client != nil {
        return p.client.Close()
    }
    return nil
}
```

### Capability Declaration

Be accurate about what your provider supports:

```go
func (p *Provider) Capabilities() vault.Capabilities {
    return vault.Capabilities{
        Read:       true,   // Always true for a useful provider
        Write:      false,  // Set false if read-only
        Delete:     false,  // Set false if deletion not supported
        List:       true,   // Set false if listing not supported
        Binary:     true,   // Set true if ValueBytes is supported
        MultiField: true,   // Set true if Fields map is supported
    }
}
```

## Module Architecture

```
omnivault-myprovider/
├── go.mod
├── go.sum
├── provider.go      # Main implementation
├── provider_test.go # Tests
├── config.go        # Configuration types
├── client.go        # Backend client (internal)
└── README.md        # Usage documentation
```

This architecture ensures:

- External providers don't bloat the core library with dependencies
- Providers can be versioned independently
- Users only install the providers they need
