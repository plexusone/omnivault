// Package omnivault provides a unified interface for secret management across
// multiple providers including password managers (1Password, Bitwarden),
// cloud secret managers (AWS, GCP, Azure), and enterprise vaults (HashiCorp Vault).
//
// Basic usage:
//
//	client, err := omnivault.NewClient(omnivault.Config{
//	    Provider: omnivault.ProviderEnv,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	secret, err := client.Get(ctx, "API_KEY")
//
// Using a custom provider:
//
//	customVault := myprovider.New(...)
//	client, err := omnivault.NewClient(omnivault.Config{
//	    CustomVault: customVault,
//	})
//
// Using the resolver for URI-based secret references:
//
//	resolver := omnivault.NewResolver()
//	resolver.Register("op", onepasswordVault)
//	resolver.Register("env", envVault)
//
//	value, err := resolver.Resolve(ctx, "op://Development/api/token")
package omnivault

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/plexusone/omnivault/vault"
)

// Config holds configuration for creating a new Client.
type Config struct {
	// Provider is the name of a built-in provider to use.
	// Ignored if CustomVault is set.
	Provider ProviderName

	// CustomVault allows injecting a custom vault implementation.
	// When set, this takes precedence over Provider.
	CustomVault vault.Vault

	// ProviderConfig contains provider-specific configuration.
	// The expected type depends on the provider being used.
	ProviderConfig any

	// HTTPClient is an optional HTTP client for providers that make HTTP requests.
	HTTPClient *http.Client

	// Logger is an optional structured logger.
	Logger *slog.Logger

	// Extra contains additional provider-specific options.
	Extra map[string]any
}

// Client wraps a vault provider with additional functionality.
type Client struct {
	vault  vault.Vault
	config Config
	logger *slog.Logger
}

// NewClient creates a new Client with the given configuration.
func NewClient(config Config) (*Client, error) {
	var v vault.Vault
	var err error

	// Custom vault takes precedence
	if config.CustomVault != nil {
		v = config.CustomVault
	} else {
		// Use built-in provider factory
		v, err = newProvider(config)
		if err != nil {
			return nil, err
		}
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		vault:  v,
		config: config,
		logger: logger,
	}, nil
}

// Get retrieves a secret from the vault.
func (c *Client) Get(ctx context.Context, path string) (*vault.Secret, error) {
	return c.vault.Get(ctx, path)
}

// GetValue retrieves only the value of a secret (convenience method).
func (c *Client) GetValue(ctx context.Context, path string) (string, error) {
	secret, err := c.vault.Get(ctx, path)
	if err != nil {
		return "", err
	}
	return secret.String(), nil
}

// GetField retrieves a specific field from a secret.
func (c *Client) GetField(ctx context.Context, path, field string) (string, error) {
	secret, err := c.vault.Get(ctx, path)
	if err != nil {
		return "", err
	}
	return secret.GetField(field), nil
}

// Set stores a secret in the vault.
func (c *Client) Set(ctx context.Context, path string, secret *vault.Secret) error {
	return c.vault.Set(ctx, path, secret)
}

// SetValue stores a simple string value as a secret (convenience method).
func (c *Client) SetValue(ctx context.Context, path, value string) error {
	return c.vault.Set(ctx, path, &vault.Secret{Value: value})
}

// Delete removes a secret from the vault.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.vault.Delete(ctx, path)
}

// Exists checks if a secret exists.
func (c *Client) Exists(ctx context.Context, path string) (bool, error) {
	return c.vault.Exists(ctx, path)
}

// List returns all secrets matching the given prefix.
func (c *Client) List(ctx context.Context, prefix string) ([]string, error) {
	return c.vault.List(ctx, prefix)
}

// Name returns the provider name.
func (c *Client) Name() string {
	return c.vault.Name()
}

// Capabilities returns the provider capabilities.
func (c *Client) Capabilities() vault.Capabilities {
	return c.vault.Capabilities()
}

// Vault returns the underlying vault provider.
// This can be used to access provider-specific functionality.
func (c *Client) Vault() vault.Vault {
	return c.vault
}

// Close releases any resources held by the client.
func (c *Client) Close() error {
	return c.vault.Close()
}

// MustGet retrieves a secret or panics if an error occurs.
func (c *Client) MustGet(ctx context.Context, path string) *vault.Secret {
	secret, err := c.Get(ctx, path)
	if err != nil {
		panic(err)
	}
	return secret
}

// MustGetValue retrieves a secret value or panics if an error occurs.
func (c *Client) MustGetValue(ctx context.Context, path string) string {
	value, err := c.GetValue(ctx, path)
	if err != nil {
		panic(err)
	}
	return value
}
