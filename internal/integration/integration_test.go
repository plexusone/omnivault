// Package integration provides integration tests for OmniVault.
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/plexusone/omnivault/internal/client"
	"github.com/plexusone/omnivault/internal/config"
	"github.com/plexusone/omnivault/internal/daemon"
)

// testPortCounter is used to allocate unique ports for Windows tests.
var testPortCounter uint32 = 19840

// testEnv holds the test environment configuration.
type testEnv struct {
	t         *testing.T
	tempDir   string
	server    *daemon.Server
	client    *client.Client
	ctx       context.Context
	cancel    context.CancelFunc
	serverErr chan error
}

// setupTestEnv creates a new test environment with a temporary directory.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "omnivault-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Allocate unique port for Windows tests
	port := atomic.AddUint32(&testPortCounter, 1)
	tcpAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Override paths to use temp directory
	paths := &config.Paths{
		ConfigDir:  tempDir,
		VaultFile:  filepath.Join(tempDir, "vault.enc"),
		MetaFile:   filepath.Join(tempDir, "vault.meta"),
		SocketPath: filepath.Join(tempDir, "omnivaultd.sock"),
		TCPAddr:    tcpAddr,
		PIDFile:    filepath.Join(tempDir, "omnivaultd.pid"),
		LogFile:    filepath.Join(tempDir, "omnivaultd.log"),
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	env := &testEnv{
		t:         t,
		tempDir:   tempDir,
		ctx:       ctx,
		cancel:    cancel,
		serverErr: make(chan error, 1),
	}

	// Create and start server with custom paths
	env.server = newTestServer(paths)

	go func() {
		env.serverErr <- env.server.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create client with custom paths
	env.client = newTestClientWithPaths(paths.SocketPath, paths.TCPAddr)

	return env
}

// cleanup tears down the test environment.
func (e *testEnv) cleanup() {
	e.t.Helper()

	// Cancel context to stop server
	e.cancel()

	// Wait for server to stop
	select {
	case <-e.serverErr:
	case <-time.After(2 * time.Second):
		e.t.Log("Server did not stop gracefully")
	}

	// Remove temp directory
	if err := os.RemoveAll(e.tempDir); err != nil {
		e.t.Logf("Failed to remove temp dir: %v", err)
	}
}

// newTestServer creates a server with custom paths for testing.
func newTestServer(paths *config.Paths) *daemon.Server {
	return daemon.NewServerWithPaths(daemon.ServerConfig{
		AutoLockDuration: 5 * time.Minute,
	}, paths)
}

// newTestClientWithPaths creates a client with custom paths for testing.
func newTestClientWithPaths(socketPath, tcpAddr string) *client.Client {
	return client.NewWithPaths(socketPath, tcpAddr)
}

// TestDaemonLifecycle tests basic daemon operations.
func TestDaemonLifecycle(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	ctx := context.Background()

	// Test 1: Check daemon is running
	t.Run("DaemonRunning", func(t *testing.T) {
		status, err := env.client.GetStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if !status.Running {
			t.Error("Expected daemon to be running")
		}

		if status.VaultExists {
			t.Error("Expected vault to not exist initially")
		}
	})

	// Test 2: Initialize vault
	t.Run("InitializeVault", func(t *testing.T) {
		err := env.client.Init(ctx, "testpassword123")
		if err != nil {
			t.Fatalf("Failed to initialize vault: %v", err)
		}

		status, err := env.client.GetStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if !status.VaultExists {
			t.Error("Expected vault to exist after init")
		}

		if status.Locked {
			t.Error("Expected vault to be unlocked after init")
		}
	})

	// Test 3: Lock vault
	t.Run("LockVault", func(t *testing.T) {
		err := env.client.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to lock vault: %v", err)
		}

		status, err := env.client.GetStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if !status.Locked {
			t.Error("Expected vault to be locked")
		}
	})

	// Test 4: Unlock vault
	t.Run("UnlockVault", func(t *testing.T) {
		err := env.client.Unlock(ctx, "testpassword123")
		if err != nil {
			t.Fatalf("Failed to unlock vault: %v", err)
		}

		status, err := env.client.GetStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if status.Locked {
			t.Error("Expected vault to be unlocked")
		}
	})

	// Test 5: Invalid password
	t.Run("InvalidPassword", func(t *testing.T) {
		// Lock first
		_ = env.client.Lock(ctx)

		err := env.client.Unlock(ctx, "wrongpassword")
		if err == nil {
			t.Error("Expected error for invalid password")
		}

		if de, ok := err.(*client.DaemonError); ok {
			if !de.IsInvalidPassword() {
				t.Errorf("Expected invalid password error, got: %v", err)
			}
		}
	})
}

// TestSecretCRUD tests secret create, read, update, delete operations.
func TestSecretCRUD(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	ctx := context.Background()

	// Initialize and unlock vault
	if err := env.client.Init(ctx, "testpassword123"); err != nil {
		t.Fatalf("Failed to init vault: %v", err)
	}

	// Test 1: Set secret
	t.Run("SetSecret", func(t *testing.T) {
		err := env.client.SetSecret(ctx, "database/password", "secret123", nil, nil)
		if err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}
	})

	// Test 2: Get secret
	t.Run("GetSecret", func(t *testing.T) {
		secret, err := env.client.GetSecret(ctx, "database/password")
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if secret.Value != "secret123" {
			t.Errorf("Expected value 'secret123', got '%s'", secret.Value)
		}

		if secret.Path != "database/password" {
			t.Errorf("Expected path 'database/password', got '%s'", secret.Path)
		}
	})

	// Test 3: Update secret
	t.Run("UpdateSecret", func(t *testing.T) {
		err := env.client.SetSecret(ctx, "database/password", "newsecret456", nil, nil)
		if err != nil {
			t.Fatalf("Failed to update secret: %v", err)
		}

		secret, err := env.client.GetSecret(ctx, "database/password")
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if secret.Value != "newsecret456" {
			t.Errorf("Expected value 'newsecret456', got '%s'", secret.Value)
		}
	})

	// Test 4: List secrets
	t.Run("ListSecrets", func(t *testing.T) {
		// Add more secrets
		_ = env.client.SetSecret(ctx, "database/username", "admin", nil, nil)
		_ = env.client.SetSecret(ctx, "api/key", "apikey123", nil, nil)

		list, err := env.client.ListSecrets(ctx, "")
		if err != nil {
			t.Fatalf("Failed to list secrets: %v", err)
		}

		if list.Count != 3 {
			t.Errorf("Expected 3 secrets, got %d", list.Count)
		}
	})

	// Test 5: List with prefix
	t.Run("ListSecretsWithPrefix", func(t *testing.T) {
		list, err := env.client.ListSecrets(ctx, "database/")
		if err != nil {
			t.Fatalf("Failed to list secrets: %v", err)
		}

		if list.Count != 2 {
			t.Errorf("Expected 2 secrets with prefix 'database/', got %d", list.Count)
		}
	})

	// Test 6: Delete secret
	t.Run("DeleteSecret", func(t *testing.T) {
		err := env.client.DeleteSecret(ctx, "api/key")
		if err != nil {
			t.Fatalf("Failed to delete secret: %v", err)
		}

		_, err = env.client.GetSecret(ctx, "api/key")
		if err == nil {
			t.Error("Expected error getting deleted secret")
		}
	})

	// Test 7: Get non-existent secret
	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := env.client.GetSecret(ctx, "nonexistent/path")
		if err == nil {
			t.Error("Expected error for non-existent secret")
		}

		if de, ok := err.(*client.DaemonError); ok {
			if !de.IsNotFound() {
				t.Errorf("Expected not found error, got: %v", err)
			}
		}
	})
}

// TestSecretWithFields tests secrets with multiple fields.
func TestSecretWithFields(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	ctx := context.Background()

	if err := env.client.Init(ctx, "testpassword123"); err != nil {
		t.Fatalf("Failed to init vault: %v", err)
	}

	// Test setting secret with fields
	t.Run("SetSecretWithFields", func(t *testing.T) {
		fields := map[string]string{
			"username": "admin",
			"password": "secret123",
			"host":     "localhost",
			"port":     "5432",
		}

		err := env.client.SetSecret(ctx, "postgres/prod", "", fields, nil)
		if err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		secret, err := env.client.GetSecret(ctx, "postgres/prod")
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if secret.Fields["username"] != "admin" {
			t.Errorf("Expected username 'admin', got '%s'", secret.Fields["username"])
		}

		if secret.Fields["password"] != "secret123" {
			t.Errorf("Expected password 'secret123', got '%s'", secret.Fields["password"])
		}
	})
}

// TestSecretWithTags tests secrets with tags.
func TestSecretWithTags(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	ctx := context.Background()

	if err := env.client.Init(ctx, "testpassword123"); err != nil {
		t.Fatalf("Failed to init vault: %v", err)
	}

	t.Run("SetSecretWithTags", func(t *testing.T) {
		tags := map[string]string{
			"env":     "production",
			"service": "api",
		}

		err := env.client.SetSecret(ctx, "api/token", "token123", nil, tags)
		if err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		secret, err := env.client.GetSecret(ctx, "api/token")
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if secret.Tags["env"] != "production" {
			t.Errorf("Expected tag env='production', got '%s'", secret.Tags["env"])
		}
	})
}

// TestVaultLocked tests operations when vault is locked.
func TestVaultLocked(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	ctx := context.Background()

	// Initialize and lock
	if err := env.client.Init(ctx, "testpassword123"); err != nil {
		t.Fatalf("Failed to init vault: %v", err)
	}
	if err := env.client.Lock(ctx); err != nil {
		t.Fatalf("Failed to lock vault: %v", err)
	}

	// Test operations fail when locked
	t.Run("SetWhenLocked", func(t *testing.T) {
		err := env.client.SetSecret(ctx, "test/secret", "value", nil, nil)
		if err == nil {
			t.Error("Expected error when setting secret on locked vault")
		}

		if de, ok := err.(*client.DaemonError); ok {
			if !de.IsVaultLocked() {
				t.Errorf("Expected vault locked error, got: %v", err)
			}
		}
	})

	t.Run("GetWhenLocked", func(t *testing.T) {
		_, err := env.client.GetSecret(ctx, "test/secret")
		if err == nil {
			t.Error("Expected error when getting secret from locked vault")
		}
	})

	t.Run("ListWhenLocked", func(t *testing.T) {
		_, err := env.client.ListSecrets(ctx, "")
		if err == nil {
			t.Error("Expected error when listing secrets from locked vault")
		}
	})
}

// TestPasswordValidation tests password requirements.
func TestPasswordValidation(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	ctx := context.Background()

	t.Run("ShortPassword", func(t *testing.T) {
		err := env.client.Init(ctx, "short")
		if err == nil {
			t.Error("Expected error for password shorter than 8 characters")
		}
	})
}

// TestDuplicateInit tests initializing an existing vault.
func TestDuplicateInit(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	ctx := context.Background()

	// First init should succeed
	if err := env.client.Init(ctx, "testpassword123"); err != nil {
		t.Fatalf("Failed to init vault: %v", err)
	}

	// Second init should fail
	err := env.client.Init(ctx, "anotherpassword")
	if err == nil {
		t.Error("Expected error for duplicate init")
	}
}
