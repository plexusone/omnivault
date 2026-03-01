package omnivault

import (
	"errors"

	"github.com/plexusone/omnivault/vault"
)

// Re-export common errors from the vault package for convenience.
var (
	ErrSecretNotFound       = vault.ErrSecretNotFound
	ErrAccessDenied         = vault.ErrAccessDenied
	ErrInvalidPath          = vault.ErrInvalidPath
	ErrReadOnly             = vault.ErrReadOnly
	ErrNotSupported         = vault.ErrNotSupported
	ErrConnectionFailed     = vault.ErrConnectionFailed
	ErrAuthenticationFailed = vault.ErrAuthenticationFailed
	ErrVersionNotFound      = vault.ErrVersionNotFound
	ErrAlreadyExists        = vault.ErrAlreadyExists
	ErrClosed               = vault.ErrClosed
)

// Client-specific errors.
var (
	// ErrNoProvider is returned when no provider is configured.
	ErrNoProvider = errors.New("no provider configured")

	// ErrUnknownScheme is returned when a secret reference has an unknown scheme.
	ErrUnknownScheme = errors.New("unknown scheme")

	// ErrInvalidSecretRef is returned when a secret reference is malformed.
	ErrInvalidSecretRef = errors.New("invalid secret reference")

	// ErrProviderNotRegistered is returned when a scheme has no registered provider.
	ErrProviderNotRegistered = errors.New("provider not registered for scheme")
)
