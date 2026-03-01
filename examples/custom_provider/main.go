// Example: Custom Provider Implementation
//
// This example demonstrates how to create a custom vault provider
// that can be used with omnivault. External providers can be developed
// as separate Go modules and injected via Config.CustomVault.
//
// To create your own provider:
// 1. Implement the vault.Vault interface
// 2. Create a constructor function that returns vault.Vault
// 3. Inject via omnivault.Config{CustomVault: yourProvider}
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/plexusone/omnivault"
	"github.com/plexusone/omnivault/vault"
)

// CustomProvider is an example custom vault provider.
// In a real implementation, this would connect to your secret store.
type CustomProvider struct {
	name    string
	apiKey  string
	mu      sync.RWMutex
	secrets map[string]*vault.Secret
}

// NewCustomProvider creates a new custom provider.
// This is the constructor that external provider packages would export.
func NewCustomProvider(name, apiKey string) vault.Vault {
	return &CustomProvider{
		name:    name,
		apiKey:  apiKey,
		secrets: make(map[string]*vault.Secret),
	}
}

// Get retrieves a secret from the custom provider.
func (p *CustomProvider) Get(ctx context.Context, path string) (*vault.Secret, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// In a real implementation, this would make an API call
	secret, ok := p.secrets[path]
	if !ok {
		return nil, vault.NewVaultError("Get", path, p.Name(), vault.ErrSecretNotFound)
	}
	return secret, nil
}

// Set stores a secret in the custom provider.
func (p *CustomProvider) Set(ctx context.Context, path string, secret *vault.Secret) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// In a real implementation, this would make an API call
	p.secrets[path] = secret
	return nil
}

// Delete removes a secret from the custom provider.
func (p *CustomProvider) Delete(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.secrets, path)
	return nil
}

// Exists checks if a secret exists.
func (p *CustomProvider) Exists(ctx context.Context, path string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, ok := p.secrets[path]
	return ok, nil
}

// List returns all secrets matching the prefix.
func (p *CustomProvider) List(ctx context.Context, prefix string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var results []string
	for path := range p.secrets {
		if strings.HasPrefix(path, prefix) {
			results = append(results, path)
		}
	}
	return results, nil
}

// Name returns the provider name.
func (p *CustomProvider) Name() string {
	return p.name
}

// Capabilities returns what this provider supports.
func (p *CustomProvider) Capabilities() vault.Capabilities {
	return vault.Capabilities{
		Read:       true,
		Write:      true,
		Delete:     true,
		List:       true,
		MultiField: true,
	}
}

// Close releases resources.
func (p *CustomProvider) Close() error {
	return nil
}

// Ensure CustomProvider implements vault.Vault
var _ vault.Vault = (*CustomProvider)(nil)

func main() {
	ctx := context.Background()

	// Create custom provider
	customVault := NewCustomProvider("my-custom-vault", "api-key-12345")

	// Use with omnivault client
	client, err := omnivault.NewClient(omnivault.Config{
		CustomVault: customVault,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Store a secret
	err = client.Set(ctx, "database/password", &vault.Secret{
		Value: "super-secret-password",
		Fields: map[string]string{
			"username": "admin",
			"host":     "db.example.com",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the secret
	secret, err := client.Get(ctx, "database/password")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Provider: %s\n", client.Name())
	fmt.Printf("Password: %s\n", secret.Value)
	fmt.Printf("Username: %s\n", secret.Fields["username"])
	fmt.Printf("Host: %s\n", secret.Fields["host"])

	// Use with resolver for multi-provider support
	resolver := omnivault.NewResolver()
	resolver.Register("custom", customVault)
	resolver.Register("env", func() vault.Vault {
		c, _ := omnivault.NewClient(omnivault.Config{Provider: omnivault.ProviderEnv})
		return c.Vault()
	}())

	// Resolve from custom provider
	value, err := resolver.Resolve(ctx, "custom://database/password")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nResolved via URI: %s\n", value)
}
