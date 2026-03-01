// Package env provides a vault implementation that reads secrets from environment variables.
//
// Usage:
//
//	v := env.New()
//	secret, err := v.Get(ctx, "API_KEY")  // reads os.Getenv("API_KEY")
//
// This provider is read-only by default. Writing to environment variables
// is possible but only affects the current process.
package env

import (
	"context"
	"os"
	"strings"

	"github.com/plexusone/omnivault/vault"
)

// Config holds configuration for the environment variable provider.
type Config struct {
	// Prefix is an optional prefix to add to all variable names.
	// For example, if Prefix is "MYAPP_", Get("API_KEY") will read "MYAPP_API_KEY".
	Prefix string

	// AllowWrite enables writing to environment variables.
	// Note: This only affects the current process.
	AllowWrite bool
}

// Provider implements vault.Vault for environment variables.
type Provider struct {
	config Config
}

// New creates a new environment variable provider.
func New() *Provider {
	return &Provider{}
}

// NewWithConfig creates a new environment variable provider with configuration.
func NewWithConfig(config Config) *Provider {
	return &Provider{config: config}
}

// Get retrieves an environment variable value.
func (p *Provider) Get(ctx context.Context, path string) (*vault.Secret, error) {
	name := p.config.Prefix + path
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil, vault.NewVaultError("Get", path, p.Name(), vault.ErrSecretNotFound)
	}
	return &vault.Secret{
		Value: value,
		Metadata: vault.Metadata{
			Provider: p.Name(),
			Path:     path,
		},
	}, nil
}

// Set sets an environment variable.
func (p *Provider) Set(ctx context.Context, path string, secret *vault.Secret) error {
	if !p.config.AllowWrite {
		return vault.NewVaultError("Set", path, p.Name(), vault.ErrReadOnly)
	}
	name := p.config.Prefix + path
	return os.Setenv(name, secret.String())
}

// Delete unsets an environment variable.
func (p *Provider) Delete(ctx context.Context, path string) error {
	if !p.config.AllowWrite {
		return vault.NewVaultError("Delete", path, p.Name(), vault.ErrReadOnly)
	}
	name := p.config.Prefix + path
	return os.Unsetenv(name)
}

// Exists checks if an environment variable is set.
func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
	name := p.config.Prefix + path
	_, ok := os.LookupEnv(name)
	return ok, nil
}

// List returns all environment variable names matching the prefix.
func (p *Provider) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := p.config.Prefix + prefix
	var results []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) >= 1 {
			name := parts[0]
			if strings.HasPrefix(name, fullPrefix) {
				// Remove the config prefix from the result
				result := strings.TrimPrefix(name, p.config.Prefix)
				results = append(results, result)
			}
		}
	}
	return results, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "env"
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() vault.Capabilities {
	return vault.Capabilities{
		Read:   true,
		Write:  p.config.AllowWrite,
		Delete: p.config.AllowWrite,
		List:   true,
	}
}

// Close is a no-op for environment variables.
func (p *Provider) Close() error {
	return nil
}

// Ensure Provider implements vault.Vault.
var _ vault.Vault = (*Provider)(nil)
