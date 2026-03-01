package omnivault

import "github.com/plexusone/omnivault/vault"

// Re-export types from the vault package for convenience.
// Users can import just "omnivault" instead of "omnivault/vault".

// Vault is the primary interface for secret storage providers.
type Vault = vault.Vault

// ExtendedVault provides additional features beyond the basic Vault interface.
type ExtendedVault = vault.ExtendedVault

// BatchVault provides batch operations for providers that support them.
type BatchVault = vault.BatchVault

// Secret represents a stored secret with its value and metadata.
type Secret = vault.Secret

// Metadata contains additional information about a secret.
type Metadata = vault.Metadata

// Timestamp wraps time.Time to provide custom JSON marshaling.
type Timestamp = vault.Timestamp

// SecretRef is a URI-style reference to a secret.
type SecretRef = vault.SecretRef

// Capabilities indicates what features a provider supports.
type Capabilities = vault.Capabilities

// Version represents a version of a secret.
type Version = vault.Version

// VaultError is a structured error with additional context.
type VaultError = vault.VaultError

// NewSecret creates a new Secret with the given value.
func NewSecret(value string) *Secret {
	return &Secret{Value: value}
}

// NewSecretWithFields creates a new Secret with the given fields.
func NewSecretWithFields(fields map[string]string) *Secret {
	return &Secret{Fields: fields}
}

// NewSecretBytes creates a new Secret with binary data.
func NewSecretBytes(data []byte) *Secret {
	return &Secret{ValueBytes: data}
}

// Now returns a Timestamp for the current time.
func Now() *Timestamp {
	return vault.Now()
}

// NewTimestamp creates a Timestamp from a time.Time.
var NewTimestamp = vault.NewTimestamp
