package omnivault

import (
	"context"
	"fmt"
	"sync"

	"github.com/plexusone/omnivault/vault"
)

// Resolver handles URI-based secret resolution across multiple providers.
// It routes secret references to the appropriate provider based on the URI scheme.
type Resolver struct {
	mu        sync.RWMutex
	providers map[string]vault.Vault
}

// NewResolver creates a new Resolver.
func NewResolver() *Resolver {
	return &Resolver{
		providers: make(map[string]vault.Vault),
	}
}

// Register adds a vault provider for the given scheme.
// The scheme should match the URI scheme used in secret references
// (e.g., "op" for op://..., "env" for env://...).
func (r *Resolver) Register(scheme string, v vault.Vault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[scheme] = v
}

// Unregister removes a vault provider for the given scheme.
func (r *Resolver) Unregister(scheme string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, scheme)
}

// Get returns the vault provider for the given scheme.
func (r *Resolver) Get(scheme string) (vault.Vault, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.providers[scheme]
	return v, ok
}

// Schemes returns all registered schemes.
func (r *Resolver) Schemes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schemes := make([]string, 0, len(r.providers))
	for scheme := range r.providers {
		schemes = append(schemes, scheme)
	}
	return schemes
}

// Resolve resolves a secret reference URI and returns the secret value.
// The URI format is: scheme://path[#field]
//
// Examples:
//
//	resolver.Resolve(ctx, "op://vault/item/field")
//	resolver.Resolve(ctx, "env://API_KEY")
//	resolver.Resolve(ctx, "aws-sm://my-secret#password")
func (r *Resolver) Resolve(ctx context.Context, uri string) (string, error) {
	secret, err := r.ResolveSecret(ctx, uri)
	if err != nil {
		return "", err
	}
	return secret.String(), nil
}

// ResolveSecret resolves a secret reference URI and returns the full Secret.
func (r *Resolver) ResolveSecret(ctx context.Context, uri string) (*vault.Secret, error) {
	ref := vault.SecretRef(uri)
	scheme := ref.Scheme()
	if scheme == "" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidSecretRef, uri)
	}

	r.mu.RLock()
	v, ok := r.providers[scheme]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotRegistered, scheme)
	}

	path := ref.Path()
	secret, err := v.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	// If a fragment (field) is specified, extract just that field
	if fragment := ref.Fragment(); fragment != "" && secret != nil {
		return &vault.Secret{
			Value:    secret.GetField(fragment),
			Metadata: secret.Metadata,
		}, nil
	}

	return secret, nil
}

// MustResolve resolves a secret reference or panics if an error occurs.
func (r *Resolver) MustResolve(ctx context.Context, uri string) string {
	value, err := r.Resolve(ctx, uri)
	if err != nil {
		panic(err)
	}
	return value
}

// ResolveAll resolves multiple secret references and returns a map of URI to value.
// If any resolution fails, it returns an error.
func (r *Resolver) ResolveAll(ctx context.Context, uris []string) (map[string]string, error) {
	results := make(map[string]string, len(uris))
	for _, uri := range uris {
		value, err := r.Resolve(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", uri, err)
		}
		results[uri] = value
	}
	return results, nil
}

// Close closes all registered providers.
func (r *Resolver) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for _, v := range r.providers {
		if err := v.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// IsSecretRef checks if a string looks like a secret reference URI.
func IsSecretRef(s string) bool {
	ref := vault.SecretRef(s)
	return ref.Scheme() != "" && len(s) > len(ref.Scheme())+3
}

// ResolveString resolves a string if it's a secret reference, otherwise returns it as-is.
// This is useful for processing configuration values that may or may not be secret references.
func (r *Resolver) ResolveString(ctx context.Context, s string) (string, error) {
	if !IsSecretRef(s) {
		return s, nil
	}
	return r.Resolve(ctx, s)
}

// ResolveMap resolves all values in a map that are secret references.
// Non-reference values are passed through unchanged.
func (r *Resolver) ResolveMap(ctx context.Context, m map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(m))
	for k, v := range m {
		resolved, err := r.ResolveString(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", k, err)
		}
		result[k] = resolved
	}
	return result, nil
}
