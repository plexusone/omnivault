// Package memory provides an in-memory vault implementation.
// This is primarily useful for testing and development.
//
// Usage:
//
//	v := memory.New()
//	v.Set(ctx, "my-secret", &vault.Secret{Value: "secret-value"})
//	secret, err := v.Get(ctx, "my-secret")
package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/plexusone/omnivault/vault"
)

// Provider implements vault.Vault with in-memory storage.
type Provider struct {
	mu      sync.RWMutex
	secrets map[string]*vault.Secret
	closed  bool
}

// New creates a new in-memory provider.
func New() *Provider {
	return &Provider{
		secrets: make(map[string]*vault.Secret),
	}
}

// NewWithSecrets creates a new in-memory provider pre-populated with secrets.
func NewWithSecrets(secrets map[string]string) *Provider {
	p := New()
	for k, v := range secrets {
		p.secrets[k] = &vault.Secret{
			Value: v,
			Metadata: vault.Metadata{
				Provider:  p.Name(),
				Path:      k,
				CreatedAt: vault.Now(),
			},
		}
	}
	return p
}

// Get retrieves a secret from memory.
func (p *Provider) Get(ctx context.Context, path string) (*vault.Secret, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, vault.NewVaultError("Get", path, p.Name(), vault.ErrClosed)
	}

	secret, ok := p.secrets[path]
	if !ok {
		return nil, vault.NewVaultError("Get", path, p.Name(), vault.ErrSecretNotFound)
	}

	// Return a copy to prevent mutation
	return p.copySecret(secret), nil
}

// Set stores a secret in memory.
func (p *Provider) Set(ctx context.Context, path string, secret *vault.Secret) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return vault.NewVaultError("Set", path, p.Name(), vault.ErrClosed)
	}

	// Store a copy to prevent external mutation
	stored := p.copySecret(secret)
	if stored.Metadata.CreatedAt == nil {
		stored.Metadata.CreatedAt = vault.Now()
	}
	stored.Metadata.ModifiedAt = vault.Now()
	stored.Metadata.Provider = p.Name()
	stored.Metadata.Path = path

	p.secrets[path] = stored
	return nil
}

// Delete removes a secret from memory.
func (p *Provider) Delete(ctx context.Context, path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return vault.NewVaultError("Delete", path, p.Name(), vault.ErrClosed)
	}

	delete(p.secrets, path)
	return nil
}

// Exists checks if a secret exists in memory.
func (p *Provider) Exists(ctx context.Context, path string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return false, vault.NewVaultError("Exists", path, p.Name(), vault.ErrClosed)
	}

	_, ok := p.secrets[path]
	return ok, nil
}

// List returns all secret paths matching the prefix.
func (p *Provider) List(ctx context.Context, prefix string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, vault.NewVaultError("List", prefix, p.Name(), vault.ErrClosed)
	}

	var results []string
	for path := range p.secrets {
		if strings.HasPrefix(path, prefix) {
			results = append(results, path)
		}
	}
	return results, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "memory"
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() vault.Capabilities {
	return vault.Capabilities{
		Read:       true,
		Write:      true,
		Delete:     true,
		List:       true,
		MultiField: true,
		Binary:     true,
	}
}

// Close marks the provider as closed.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	p.secrets = nil
	return nil
}

// Clear removes all secrets from memory.
func (p *Provider) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.secrets = make(map[string]*vault.Secret)
}

// Count returns the number of secrets stored.
func (p *Provider) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.secrets)
}

// copySecret creates a deep copy of a secret.
func (p *Provider) copySecret(secret *vault.Secret) *vault.Secret {
	if secret == nil {
		return nil
	}

	copied := &vault.Secret{
		Value:    secret.Value,
		Metadata: secret.Metadata,
	}

	if len(secret.ValueBytes) > 0 {
		copied.ValueBytes = make([]byte, len(secret.ValueBytes))
		copy(copied.ValueBytes, secret.ValueBytes)
	}

	if secret.Fields != nil {
		copied.Fields = make(map[string]string, len(secret.Fields))
		for k, v := range secret.Fields {
			copied.Fields[k] = v
		}
	}

	if secret.Metadata.Tags != nil {
		copied.Metadata.Tags = make(map[string]string, len(secret.Metadata.Tags))
		for k, v := range secret.Metadata.Tags {
			copied.Metadata.Tags[k] = v
		}
	}

	if secret.Metadata.Labels != nil {
		copied.Metadata.Labels = make([]string, len(secret.Metadata.Labels))
		copy(copied.Metadata.Labels, secret.Metadata.Labels)
	}

	return copied
}

// Ensure Provider implements vault.Vault.
var _ vault.Vault = (*Provider)(nil)
