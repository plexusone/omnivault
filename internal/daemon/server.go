package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/plexusone/omnivault/internal/config"
	"github.com/plexusone/omnivault/internal/store"
	"github.com/plexusone/omnivault/vault"
)

// Server is the OmniVault daemon server.
type Server struct {
	mu        sync.RWMutex
	store     *store.EncryptedStore
	paths     *config.Paths
	listener  net.Listener
	server    *http.Server
	logger    *slog.Logger
	startTime time.Time

	// Auto-lock settings
	autoLockDuration time.Duration
	autoLockTimer    *time.Timer
}

// ServerConfig contains server configuration.
type ServerConfig struct {
	Logger           *slog.Logger
	AutoLockDuration time.Duration
}

// NewServer creates a new daemon server.
func NewServer(cfg ServerConfig) *Server {
	return NewServerWithPaths(cfg, config.GetPaths())
}

// NewServerWithPaths creates a new daemon server with custom paths (for testing).
func NewServerWithPaths(cfg ServerConfig, paths *config.Paths) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	autoLock := cfg.AutoLockDuration
	if autoLock == 0 {
		autoLock = 15 * time.Minute // Default auto-lock
	}

	return &Server{
		store:            store.NewEncryptedStore(paths.VaultFile, paths.MetaFile),
		paths:            paths,
		logger:           logger,
		autoLockDuration: autoLock,
	}
}

// Run starts the daemon server.
func (s *Server) Run(ctx context.Context) error {
	// Ensure config directory exists
	if err := s.paths.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Cleanup any existing socket
	_ = s.paths.CleanupSocket()

	// Create listener
	listener, err := s.createListener()
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Create HTTP server with routes
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	s.startTime = time.Now()

	// Write PID file
	if err := s.writePIDFile(); err != nil {
		s.logger.Warn("failed to write PID file", "error", err)
	}

	s.logger.Info("daemon started", "socket", s.paths.SocketPath)

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("context cancelled, shutting down")
	case sig := <-sigCh:
		s.logger.Info("received signal, shutting down", "signal", sig)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	return s.Shutdown()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	s.logger.Info("shutting down daemon")

	// Stop auto-lock timer
	if s.autoLockTimer != nil {
		s.autoLockTimer.Stop()
	}

	// Lock the vault
	if err := s.store.Lock(); err != nil {
		s.logger.Warn("failed to lock vault on shutdown", "error", err)
	}

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Warn("failed to shutdown server", "error", err)
		}
	}

	// Cleanup socket and PID file
	_ = s.paths.CleanupSocket()
	_ = os.Remove(s.paths.PIDFile)

	return nil
}

// createListener creates the appropriate listener for the platform.
func (s *Server) createListener() (net.Listener, error) {
	if runtime.GOOS == "windows" {
		// Windows uses TCP on localhost
		return net.Listen("tcp", s.paths.TCPAddr)
	}

	return net.Listen("unix", s.paths.SocketPath)
}

// registerRoutes registers HTTP routes.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/init", s.handleInit)
	mux.HandleFunc("/unlock", s.handleUnlock)
	mux.HandleFunc("/lock", s.handleLock)
	mux.HandleFunc("/secrets", s.handleSecrets)
	mux.HandleFunc("/secret/", s.handleSecret)
	mux.HandleFunc("/stop", s.handleStop)
}

// handleStatus returns the daemon status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	status := StatusResponse{
		Running:     true,
		Locked:      s.store.IsLocked(),
		VaultExists: s.store.VaultExists(),
		SecretCount: s.store.SecretCount(),
		Uptime:      time.Since(s.startTime).Round(time.Second).String(),
	}

	if !s.store.IsLocked() {
		status.UnlockedAt = s.store.UnlockTime()
	}

	s.writeJSON(w, http.StatusOK, status)
}

// handleInit initializes a new vault.
func (s *Server) handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	var req InitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body", ErrCodeInvalidRequest)
		return
	}

	if len(req.Password) < 8 {
		s.writeError(w, http.StatusBadRequest, "password must be at least 8 characters", ErrCodeInvalidRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.store.VaultExists() {
		s.writeError(w, http.StatusConflict, "vault already exists", ErrCodeAlreadyExists)
		return
	}

	if err := s.store.Initialize(req.Password); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	s.resetAutoLock()
	s.writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "vault initialized"})
}

// handleUnlock unlocks the vault.
func (s *Server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	var req UnlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body", ErrCodeInvalidRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.store.VaultExists() {
		s.writeError(w, http.StatusNotFound, "vault does not exist, run init first", ErrCodeVaultNotFound)
		return
	}

	if err := s.store.Unlock(req.Password); err != nil {
		if strings.Contains(err.Error(), "invalid password") {
			s.writeError(w, http.StatusUnauthorized, "invalid password", ErrCodeInvalidPassword)
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		}
		return
	}

	s.resetAutoLock()
	s.writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "vault unlocked"})
}

// handleLock locks the vault.
func (s *Server) handleLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.autoLockTimer != nil {
		s.autoLockTimer.Stop()
	}

	if err := s.store.Lock(); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	s.writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "vault locked"})
}

// handleSecrets handles listing secrets.
func (s *Server) handleSecrets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.store.IsLocked() {
		s.writeError(w, http.StatusForbidden, "vault is locked", ErrCodeVaultLocked)
		return
	}

	prefix := r.URL.Query().Get("prefix")
	paths, err := s.store.List(r.Context(), prefix)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	// Build list response with metadata
	items := make([]SecretListItem, 0, len(paths))
	for _, path := range paths {
		secret, err := s.store.Get(r.Context(), path)
		if err != nil {
			continue
		}

		var tags []string
		if secret.Metadata.Tags != nil {
			for k := range secret.Metadata.Tags {
				tags = append(tags, k)
			}
		}

		item := SecretListItem{
			Path:      path,
			HasValue:  secret.Value != "" || len(secret.ValueBytes) > 0,
			HasFields: len(secret.Fields) > 0,
			Tags:      tags,
		}
		if secret.Metadata.ModifiedAt != nil {
			item.UpdatedAt = secret.Metadata.ModifiedAt.Time
		}

		items = append(items, item)
	}

	s.resetAutoLock()
	s.writeJSON(w, http.StatusOK, ListResponse{Secrets: items, Count: len(items)})
}

// handleSecret handles single secret operations.
func (s *Server) handleSecret(w http.ResponseWriter, r *http.Request) {
	// Extract path from URL
	path := strings.TrimPrefix(r.URL.Path, "/secret/")
	if path == "" {
		s.writeError(w, http.StatusBadRequest, "path is required", ErrCodeInvalidRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.store.IsLocked() {
		s.writeError(w, http.StatusForbidden, "vault is locked", ErrCodeVaultLocked)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getSecret(w, r, path)
	case http.MethodPut:
		s.setSecret(w, r, path)
	case http.MethodDelete:
		s.deleteSecret(w, r, path)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
	}
}

func (s *Server) getSecret(w http.ResponseWriter, r *http.Request, path string) {
	secret, err := s.store.Get(r.Context(), path)
	if err != nil {
		if err == vault.ErrSecretNotFound {
			s.writeError(w, http.StatusNotFound, "secret not found", ErrCodeSecretNotFound)
		} else {
			s.writeError(w, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		}
		return
	}

	resp := SecretResponse{
		Path:   path,
		Value:  secret.String(),
		Fields: secret.Fields,
	}
	if secret.Metadata.Tags != nil {
		resp.Tags = secret.Metadata.Tags
	}
	if secret.Metadata.CreatedAt != nil {
		resp.CreatedAt = secret.Metadata.CreatedAt.Time
	}
	if secret.Metadata.ModifiedAt != nil {
		resp.UpdatedAt = secret.Metadata.ModifiedAt.Time
	}

	s.resetAutoLock()
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) setSecret(w http.ResponseWriter, r *http.Request, path string) {
	var req SetSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body", ErrCodeInvalidRequest)
		return
	}

	secret := &vault.Secret{
		Value:  req.Value,
		Fields: req.Fields,
		Metadata: vault.Metadata{
			Tags: req.Tags,
		},
	}

	if err := s.store.Set(r.Context(), path, secret); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	s.resetAutoLock()
	s.writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "secret saved"})
}

func (s *Server) deleteSecret(w http.ResponseWriter, r *http.Request, path string) {
	if err := s.store.Delete(r.Context(), path); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	s.resetAutoLock()
	s.writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "secret deleted"})
}

// handleStop stops the daemon.
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	s.writeJSON(w, http.StatusOK, SuccessResponse{Success: true, Message: "daemon stopping"})

	// Shutdown in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := s.Shutdown(); err != nil {
			s.logger.Error("shutdown error", "error", err)
		}
		os.Exit(0)
	}()
}

// resetAutoLock resets the auto-lock timer.
func (s *Server) resetAutoLock() {
	if s.autoLockTimer != nil {
		s.autoLockTimer.Stop()
	}

	s.autoLockTimer = time.AfterFunc(s.autoLockDuration, func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if err := s.store.Lock(); err != nil {
			s.logger.Warn("auto-lock failed", "error", err)
		} else {
			s.logger.Info("vault auto-locked due to inactivity")
		}
	})
}

// writePIDFile writes the daemon PID to a file.
func (s *Server) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(s.paths.PIDFile, []byte(fmt.Sprintf("%d", pid)), 0600)
}

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response.
func (s *Server) writeError(w http.ResponseWriter, status int, message, code string) {
	s.writeJSON(w, status, ErrorResponse{Error: message, Code: code})
}
