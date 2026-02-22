# Providers

OmniVault supports multiple secret storage backends through providers.

## Built-in Providers

These providers are included in the core library with zero external dependencies.

### Environment Variables

Read secrets from environment variables:

```go
import "github.com/agentplexus/omnivault/providers/env"

provider := env.New()

// Or use with client
client, _ := omnivault.NewClient(omnivault.Config{
    Provider: omnivault.ProviderEnv,
})
```

| Capability | Supported |
|------------|-----------|
| Read | Yes |
| Write | No |
| Delete | No |
| List | No |

**URI Scheme:** `env://`

```go
resolver.Register("env", env.New())
value, _ := resolver.Resolve(ctx, "env://API_KEY")
```

### File

Read secrets from files:

```go
import "github.com/agentplexus/omnivault/providers/file"

provider := file.New(file.Config{
    BasePath: "/etc/secrets",
})

// Or use with client
client, _ := omnivault.NewClient(omnivault.Config{
    Provider: omnivault.ProviderFile,
})
```

| Capability | Supported |
|------------|-----------|
| Read | Yes |
| Write | Yes |
| Delete | Yes |
| List | Yes |

**URI Scheme:** `file://`

```go
resolver.Register("file", file.New(file.Config{BasePath: "/etc/secrets"}))
value, _ := resolver.Resolve(ctx, "file://database/password")
```

### Memory

In-memory storage, useful for testing:

```go
import "github.com/agentplexus/omnivault/providers/memory"

// Empty store
provider := memory.New()

// Pre-populated store
provider := memory.NewWithSecrets(map[string]string{
    "api_key": "sk-12345",
    "db_pass": "secret",
})
```

| Capability | Supported |
|------------|-----------|
| Read | Yes |
| Write | Yes |
| Delete | Yes |
| List | Yes |

**URI Scheme:** `memory://`

## Official Provider Modules

First-party modules maintained alongside OmniVault. Install separately to avoid dependency bloat.

### omnivault-aws

AWS Secrets Manager and Parameter Store:

```bash
go get github.com/agentplexus/omnivault-aws
```

```go
import aws "github.com/agentplexus/omnivault-aws"

// AWS Secrets Manager
smProvider, _ := aws.NewSecretsManager(aws.Config{
    Region: "us-east-1",
})

// AWS Parameter Store (SSM)
ssmProvider, _ := aws.NewParameterStore(aws.Config{
    Region: "us-east-1",
})

resolver.Register("aws-sm", smProvider)
resolver.Register("aws-ssm", ssmProvider)
```

**URI Schemes:** `aws-sm://`, `aws-ssm://`

### omnivault-keyring

OS-level keyring integration:

```bash
go get github.com/agentplexus/omnivault-keyring
```

```go
import "github.com/agentplexus/omnivault-keyring"

provider := keyring.New(keyring.Config{
    ServiceName: "myapp",
})

resolver.Register("keyring", provider)
```

**Supported backends:**

- macOS Keychain
- Windows Credential Manager
- Linux Secret Service (GNOME Keyring, KWallet)

**URI Scheme:** `keyring://`

## Community Providers

External providers can be developed as separate Go modules:

| Category | Potential Providers |
|----------|---------------------|
| **Password Managers** | 1Password, Bitwarden, LastPass, KeePass, pass/gopass |
| **Cloud** | GCP Secret Manager, Azure Key Vault, DigitalOcean |
| **Enterprise** | HashiCorp Vault, CyberArk Conjur, Akeyless, Doppler |

## Provider Capabilities

Query what a provider supports:

```go
caps := provider.Capabilities()

fmt.Printf("Read: %v\n", caps.Read)
fmt.Printf("Write: %v\n", caps.Write)
fmt.Printf("Delete: %v\n", caps.Delete)
fmt.Printf("List: %v\n", caps.List)
fmt.Printf("Binary: %v\n", caps.Binary)
fmt.Printf("MultiField: %v\n", caps.MultiField)
```

## Provider Selection Guide

| Use Case | Recommended Provider |
|----------|---------------------|
| Local development | `env`, `memory` |
| CI/CD pipelines | `env`, `aws-ssm` |
| Production (AWS) | `aws-sm` |
| Desktop apps | `keyring` |
| Testing | `memory` |
| Kubernetes | `file` (with mounted secrets) |
