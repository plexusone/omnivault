# Client API

The OmniVault client provides a high-level API for secret management.

## Creating a Client

```go
// With a built-in provider
client, err := omnivault.NewClient(omnivault.Config{
    Provider: omnivault.ProviderEnv,  // or ProviderFile, ProviderMemory
})

// With a custom provider
client, err := omnivault.NewClient(omnivault.Config{
    CustomVault: myCustomProvider,
})
```

!!! warning "Always Close"
    Always call `client.Close()` when done to release resources.

## Basic Operations

### Get

Retrieve a secret by path:

```go
secret, err := client.Get(ctx, "path/to/secret")
if err != nil {
    if err == vault.ErrSecretNotFound {
        // Secret doesn't exist
    }
    // Handle other errors
}
```

### Set

Store a secret:

```go
secret := &omnivault.Secret{
    Value: "my-secret-value",
}
err := client.Set(ctx, "path/to/secret", secret)
```

### Delete

Remove a secret:

```go
err := client.Delete(ctx, "path/to/secret")
```

### Exists

Check if a secret exists:

```go
exists, err := client.Exists(ctx, "path/to/secret")
```

### List

List secrets by prefix:

```go
paths, err := client.List(ctx, "database/")
// Returns: ["database/password", "database/username", ...]
```

## Convenience Methods

### GetValue

Get just the string value:

```go
value, err := client.GetValue(ctx, "API_KEY")
```

### GetField

Get a specific field from a multi-field secret:

```go
password, err := client.GetField(ctx, "database/credentials", "password")
```

### SetValue

Set a simple string value:

```go
err := client.SetValue(ctx, "API_KEY", "sk-12345")
```

## Must Variants

Variants that panic on error (useful for initialization):

```go
// Panics if secret not found or error occurs
secret := client.MustGet(ctx, "REQUIRED_SECRET")
value := client.MustGetValue(ctx, "REQUIRED_VALUE")
```

!!! danger "Use with Caution"
    Must variants should only be used during application startup or in contexts where a missing secret is truly fatal.

## Secret Structure

```go
type Secret struct {
    // Primary value
    Value string

    // Binary value (alternative to Value)
    ValueBytes []byte

    // Additional fields (for multi-field secrets)
    Fields map[string]string

    // Metadata
    Metadata Metadata
}

type Metadata struct {
    // Custom tags
    Tags map[string]string

    // Timestamps (set automatically)
    CreatedAt  *time.Time
    ModifiedAt *time.Time
}
```

### Accessing Values

```go
secret, _ := client.Get(ctx, "path")

// Primary value as string
value := secret.String()

// Primary value as bytes
bytes := secret.Bytes()

// Specific field
field := secret.GetField("username")

// Check if field exists
if val, ok := secret.Fields["password"]; ok {
    // Use val
}
```

## Error Handling

```go
secret, err := client.Get(ctx, "path")
if err != nil {
    switch err {
    case vault.ErrSecretNotFound:
        // Secret doesn't exist
    case vault.ErrAccessDenied:
        // Permission denied
    default:
        // Other error
    }
}
```

## Provider Capabilities

Check what operations a provider supports:

```go
caps := client.Capabilities()

if caps.Write {
    // Provider supports writing secrets
}

if caps.List {
    // Provider supports listing secrets
}
```

```go
type Capabilities struct {
    Read       bool  // Can read secrets
    Write      bool  // Can write secrets
    Delete     bool  // Can delete secrets
    List       bool  // Can list secrets
    Binary     bool  // Supports binary values
    MultiField bool  // Supports multiple fields per secret
}
```
