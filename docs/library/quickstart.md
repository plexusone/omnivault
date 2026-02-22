# Library Quick Start

This guide shows you how to use OmniVault as a Go library to access secrets from various providers.

## Basic Usage

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

## Multi-Provider Resolution

Use the resolver to access secrets from multiple providers using URIs:

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

## Using External Providers

Install and use official provider modules:

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

## Working with Secrets

Secrets can contain a primary value, multiple fields, and metadata:

```go
// Create a secret with fields
secret := &omnivault.Secret{
    Value: "primary-value",
    Fields: map[string]string{
        "username": "admin",
        "password": "secret123",
    },
    Metadata: omnivault.Metadata{
        Tags: map[string]string{
            "env": "production",
        },
    },
}

// Store the secret
err := client.Set(ctx, "database/credentials", secret)

// Retrieve and access values
retrieved, _ := client.Get(ctx, "database/credentials")
fmt.Println(retrieved.String())              // Primary value
fmt.Println(retrieved.GetField("username"))  // Specific field
fmt.Println(retrieved.Bytes())               // As bytes
```

## Next Steps

- [Client API](client.md) - Full client API reference
- [URI Resolver](resolver.md) - URI-based secret resolution
- [Providers](providers.md) - Available providers
- [Custom Providers](custom-providers.md) - Create your own provider
