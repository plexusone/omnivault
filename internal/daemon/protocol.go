// Package daemon provides the OmniVault daemon server.
package daemon

import "time"

// Request types for daemon IPC.

// UnlockRequest is the request to unlock the vault.
type UnlockRequest struct {
	Password string `json:"password"` //nolint:gosec // G117: password field is intentional for vault
}

// SetSecretRequest is the request to set a secret.
type SetSecretRequest struct {
	Value  string            `json:"value,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// ChangePasswordRequest is the request to change the master password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// InitRequest is the request to initialize a new vault.
type InitRequest struct {
	Password string `json:"password"` //nolint:gosec // G117: password field is intentional for vault
}

// Response types for daemon IPC.

// StatusResponse is the response for status requests.
type StatusResponse struct {
	Running     bool      `json:"running"`
	Locked      bool      `json:"locked"`
	VaultExists bool      `json:"vault_exists"`
	SecretCount int       `json:"secret_count"`
	UnlockedAt  time.Time `json:"unlocked_at,omitempty"`
	Uptime      string    `json:"uptime"`
}

// SecretResponse is the response for get secret requests.
type SecretResponse struct {
	Path      string            `json:"path"`
	Value     string            `json:"value,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty"`
}

// SecretListItem is an item in the secret list (metadata only).
type SecretListItem struct {
	Path      string    `json:"path"`
	HasValue  bool      `json:"has_value"`
	HasFields bool      `json:"has_fields"`
	Tags      []string  `json:"tags,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ListResponse is the response for list requests.
type ListResponse struct {
	Secrets []SecretListItem `json:"secrets"`
	Count   int              `json:"count"`
}

// ErrorResponse is the response for errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse is a generic success response.
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Error codes.
const (
	ErrCodeVaultLocked     = "VAULT_LOCKED"
	ErrCodeVaultNotFound   = "VAULT_NOT_FOUND"
	ErrCodeSecretNotFound  = "SECRET_NOT_FOUND"
	ErrCodeInvalidPassword = "INVALID_PASSWORD"
	ErrCodeInvalidRequest  = "INVALID_REQUEST"
	ErrCodeInternalError   = "INTERNAL_ERROR"
	ErrCodeAlreadyExists   = "ALREADY_EXISTS"
)
