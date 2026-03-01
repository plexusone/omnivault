// Package client provides a client for the OmniVault daemon.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/plexusone/omnivault/internal/config"
	"github.com/plexusone/omnivault/internal/daemon"
)

// Client is a client for the OmniVault daemon.
type Client struct {
	socketPath string // Unix socket path (Unix only)
	tcpAddr    string // TCP address (Windows only)
	httpClient *http.Client
}

// New creates a new daemon client.
func New() *Client {
	paths := config.GetPaths()
	return NewWithPaths(paths.SocketPath, paths.TCPAddr)
}

// NewWithSocket creates a new daemon client with a custom socket path (for testing).
// Deprecated: Use NewWithPaths for cross-platform support.
func NewWithSocket(socketPath string) *Client {
	return NewWithPaths(socketPath, "")
}

// NewWithPaths creates a new daemon client with custom paths (for testing).
func NewWithPaths(socketPath, tcpAddr string) *Client {
	c := &Client{
		socketPath: socketPath,
		tcpAddr:    tcpAddr,
	}

	// Create HTTP client with appropriate transport
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			if runtime.GOOS == "windows" {
				return net.Dial("tcp", c.tcpAddr)
			}
			return net.Dial("unix", c.socketPath)
		},
	}

	c.httpClient = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return c
}

// IsDaemonRunning checks if the daemon is running.
func (c *Client) IsDaemonRunning() bool {
	if runtime.GOOS == "windows" {
		// Windows: try to connect via TCP
		conn, err := net.DialTimeout("tcp", c.tcpAddr, time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}

	// Unix: check socket file exists
	_, err := os.Stat(c.socketPath)
	if err != nil {
		return false
	}

	// Try to connect
	conn, err := net.DialTimeout("unix", c.socketPath, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetStatus returns the daemon status.
func (c *Client) GetStatus(ctx context.Context) (*daemon.StatusResponse, error) {
	var resp daemon.StatusResponse
	if err := c.get(ctx, "/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Init initializes a new vault.
func (c *Client) Init(ctx context.Context, password string) error {
	req := daemon.InitRequest{Password: password}
	var resp daemon.SuccessResponse
	return c.post(ctx, "/init", req, &resp)
}

// Unlock unlocks the vault.
func (c *Client) Unlock(ctx context.Context, password string) error {
	req := daemon.UnlockRequest{Password: password}
	var resp daemon.SuccessResponse
	return c.post(ctx, "/unlock", req, &resp)
}

// Lock locks the vault.
func (c *Client) Lock(ctx context.Context) error {
	var resp daemon.SuccessResponse
	return c.post(ctx, "/lock", nil, &resp)
}

// ListSecrets returns all secrets.
func (c *Client) ListSecrets(ctx context.Context, prefix string) (*daemon.ListResponse, error) {
	path := "/secrets"
	if prefix != "" {
		path += "?prefix=" + prefix
	}

	var resp daemon.ListResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSecret retrieves a secret.
func (c *Client) GetSecret(ctx context.Context, path string) (*daemon.SecretResponse, error) {
	var resp daemon.SecretResponse
	if err := c.get(ctx, "/secret/"+path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetSecret stores a secret.
func (c *Client) SetSecret(ctx context.Context, path, value string, fields, tags map[string]string) error {
	req := daemon.SetSecretRequest{
		Value:  value,
		Fields: fields,
		Tags:   tags,
	}
	var resp daemon.SuccessResponse
	return c.request(ctx, http.MethodPut, "/secret/"+path, req, &resp)
}

// DeleteSecret removes a secret.
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	var resp daemon.SuccessResponse
	return c.request(ctx, http.MethodDelete, "/secret/"+path, nil, &resp)
}

// Stop stops the daemon.
func (c *Client) Stop(ctx context.Context) error {
	var resp daemon.SuccessResponse
	return c.post(ctx, "/stop", nil, &resp)
}

// get performs a GET request.
func (c *Client) get(ctx context.Context, path string, result any) error {
	return c.request(ctx, http.MethodGet, path, nil, result)
}

// post performs a POST request.
func (c *Client) post(ctx context.Context, path string, body, result any) error {
	return c.request(ctx, http.MethodPost, path, body, result)
}

// request performs an HTTP request.
func (c *Client) request(ctx context.Context, method, path string, body, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Use "http://localhost" as the host; the transport will use the socket
	req, err := http.NewRequestWithContext(ctx, method, "http://localhost"+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req) //nolint:gosec // G704: request goes to localhost via unix socket
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	if resp.StatusCode >= 400 {
		var errResp daemon.ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
			return &DaemonError{
				StatusCode: resp.StatusCode,
				Code:       errResp.Code,
				Message:    errResp.Error,
			}
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// DaemonError represents an error from the daemon.
type DaemonError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *DaemonError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// IsVaultLocked returns true if the error indicates the vault is locked.
func (e *DaemonError) IsVaultLocked() bool {
	return e.Code == daemon.ErrCodeVaultLocked
}

// IsNotFound returns true if the error indicates not found.
func (e *DaemonError) IsNotFound() bool {
	return e.Code == daemon.ErrCodeSecretNotFound || e.Code == daemon.ErrCodeVaultNotFound
}

// IsInvalidPassword returns true if the error indicates invalid password.
func (e *DaemonError) IsInvalidPassword() bool {
	return e.Code == daemon.ErrCodeInvalidPassword
}
