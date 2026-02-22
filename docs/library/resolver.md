# URI Resolver

The resolver enables accessing secrets from multiple providers using URI syntax.

## URI Format

```
scheme://path[#field]
```

| Component | Description | Example |
|-----------|-------------|---------|
| `scheme` | Provider identifier | `env`, `aws-sm`, `keyring` |
| `path` | Secret path | `API_KEY`, `prod/database` |
| `field` | Optional field name | `#password` |

### Examples

```
env://API_KEY                    # Environment variable
file:///path/to/secret           # File
memory://database/password       # In-memory
aws-sm://prod/database#password  # AWS Secrets Manager with field
keyring://myapp/token            # OS keyring
```

## Creating a Resolver

```go
resolver := omnivault.NewResolver()

// Register providers with scheme names
resolver.Register("env", env.New())
resolver.Register("memory", memory.New())
resolver.Register("aws-sm", awsProvider)
```

## Resolving Secrets

### Resolve

Get the string value of a secret:

```go
value, err := resolver.Resolve(ctx, "env://API_KEY")
```

### ResolveSecret

Get the full secret object:

```go
secret, err := resolver.ResolveSecret(ctx, "aws-sm://prod/database")

// Access fields
username := secret.GetField("username")
password := secret.GetField("password")
```

### ResolveString

Resolve if it's a URI, otherwise return as-is:

```go
// If input is a URI, resolves it
// If input is a plain string, returns it unchanged
value, err := resolver.ResolveString(ctx, maybeSecretRef)
```

This is useful when a value might be either a literal or a secret reference:

```go
config := map[string]string{
    "api_key":  "env://API_KEY",      // Will be resolved
    "timeout":  "30s",                 // Returned as-is
}

for key, val := range config {
    resolved, _ := resolver.ResolveString(ctx, val)
    // Use resolved value
}
```

### ResolveMap

Resolve all values in a map:

```go
config := map[string]string{
    "database_url": "aws-sm://prod/database#url",
    "api_key":      "env://API_KEY",
    "timeout":      "30s",
}

resolved, err := resolver.ResolveMap(ctx, config)
// resolved["database_url"] = actual database URL
// resolved["api_key"] = actual API key
// resolved["timeout"] = "30s" (unchanged)
```

## Provider Registration

### Static Registration

Register providers at startup:

```go
resolver := omnivault.NewResolver()
resolver.Register("env", env.New())
resolver.Register("aws-sm", awsProvider)
```

### Dynamic Registration

Add providers at runtime:

```go
// Add a new provider
resolver.Register("vault", hashicorpVaultProvider)

// Check if a scheme is registered
if resolver.HasScheme("vault") {
    // Provider is available
}
```

## Error Handling

```go
value, err := resolver.Resolve(ctx, "unknown://secret")
if err != nil {
    // Could be:
    // - Unknown scheme
    // - Secret not found
    // - Provider error
}
```

## Use Cases

### Configuration Loading

```go
type Config struct {
    DatabaseURL string `json:"database_url"`
    APIKey      string `json:"api_key"`
    Debug       bool   `json:"debug"`
}

func LoadConfig(resolver *omnivault.Resolver) (*Config, error) {
    // Load raw config (might contain secret references)
    raw := loadRawConfig()

    // Resolve secrets
    dbURL, err := resolver.ResolveString(ctx, raw.DatabaseURL)
    if err != nil {
        return nil, err
    }

    apiKey, err := resolver.ResolveString(ctx, raw.APIKey)
    if err != nil {
        return nil, err
    }

    return &Config{
        DatabaseURL: dbURL,
        APIKey:      apiKey,
        Debug:       raw.Debug,
    }, nil
}
```

### Environment-Based Provider Selection

```go
func setupResolver(env string) *omnivault.Resolver {
    resolver := omnivault.NewResolver()

    switch env {
    case "production":
        awsProvider, _ := aws.NewSecretsManager(aws.Config{
            Region: "us-east-1",
        })
        resolver.Register("secrets", awsProvider)

    case "development":
        resolver.Register("secrets", env.New())
    }

    return resolver
}
```
