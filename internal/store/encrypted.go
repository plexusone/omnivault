package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/plexusone/omnivault/vault"
)

// VaultMeta contains unencrypted vault metadata.
type VaultMeta struct {
	Version      int          `json:"version"`
	CreatedAt    time.Time    `json:"created_at"`
	Salt         []byte       `json:"salt"`
	Argon2Params Argon2Params `json:"argon2_params"`
	Verification string       `json:"verification"` // Encrypted verification blob
}

// VaultData contains encrypted vault data.
type VaultData struct {
	Secrets map[string]string `json:"secrets"` // path -> encrypted secret JSON
}

// EncryptedStore implements vault.Vault with encrypted file storage.
type EncryptedStore struct {
	mu         sync.RWMutex
	vaultPath  string
	metaPath   string
	crypto     *Crypto
	meta       *VaultMeta
	data       *VaultData
	dirty      bool
	autoSave   bool
	unlockTime time.Time
}

// NewEncryptedStore creates a new encrypted store.
func NewEncryptedStore(vaultPath, metaPath string) *EncryptedStore {
	return &EncryptedStore{
		vaultPath: vaultPath,
		metaPath:  metaPath,
		autoSave:  true,
	}
}

// Initialize creates a new vault with the given master password.
func (s *EncryptedStore) Initialize(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.VaultExists() {
		return errors.New("vault already exists")
	}

	// Create crypto with new random salt
	crypto, err := NewCrypto(nil, DefaultArgon2Params())
	if err != nil {
		return fmt.Errorf("failed to create crypto: %w", err)
	}

	// Unlock with password to create verification blob
	crypto.Unlock(password)
	verification, err := crypto.CreateVerificationBlob()
	if err != nil {
		crypto.Lock()
		return fmt.Errorf("failed to create verification: %w", err)
	}

	// Create metadata
	s.meta = &VaultMeta{
		Version:      1,
		CreatedAt:    time.Now(),
		Salt:         crypto.Salt(),
		Argon2Params: crypto.Params(),
		Verification: verification,
	}

	// Create empty vault data
	s.data = &VaultData{
		Secrets: make(map[string]string),
	}

	s.crypto = crypto
	s.unlockTime = time.Now()

	// Save to disk
	if err := s.saveMeta(); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	if err := s.saveData(); err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	return nil
}

// VaultExists returns true if the vault exists on disk.
func (s *EncryptedStore) VaultExists() bool {
	_, err := os.Stat(s.metaPath)
	return err == nil
}

// Unlock unlocks the vault with the master password.
func (s *EncryptedStore) Unlock(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.VaultExists() {
		return errors.New("vault does not exist, run init first")
	}

	// Load metadata
	if err := s.loadMeta(); err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	// Create crypto with saved salt and params
	crypto, err := NewCrypto(s.meta.Salt, s.meta.Argon2Params)
	if err != nil {
		return fmt.Errorf("failed to create crypto: %w", err)
	}

	// Verify password
	if !crypto.VerifyPassword(password, s.meta.Verification) {
		return errors.New("invalid password")
	}

	// Unlock
	crypto.Unlock(password)
	s.crypto = crypto
	s.unlockTime = time.Now()

	// Load vault data
	if err := s.loadData(); err != nil {
		s.crypto.Lock()
		s.crypto = nil
		return fmt.Errorf("failed to load vault data: %w", err)
	}

	return nil
}

// Lock locks the vault.
func (s *EncryptedStore) Lock() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.crypto == nil {
		return nil
	}

	// Save any dirty data first
	if s.dirty {
		if err := s.saveData(); err != nil {
			return fmt.Errorf("failed to save data: %w", err)
		}
	}

	s.crypto.Lock()
	s.crypto = nil
	s.data = nil
	s.dirty = false

	return nil
}

// IsLocked returns true if the vault is locked.
func (s *EncryptedStore) IsLocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLockedUnsafe()
}

// isLockedUnsafe checks lock status without acquiring mutex (caller must hold lock).
func (s *EncryptedStore) isLockedUnsafe() bool {
	return s.crypto == nil || !s.crypto.IsUnlocked()
}

// UnlockTime returns when the vault was unlocked.
func (s *EncryptedStore) UnlockTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unlockTime
}

// Get retrieves a secret from the vault.
func (s *EncryptedStore) Get(ctx context.Context, path string) (*vault.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.isLockedUnsafe() {
		return nil, errors.New("vault is locked")
	}

	encrypted, ok := s.data.Secrets[path]
	if !ok {
		return nil, vault.ErrSecretNotFound
	}

	decrypted, err := s.crypto.DecryptString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	var secret vault.Secret
	if err := json.Unmarshal([]byte(decrypted), &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	return &secret, nil
}

// Set stores a secret in the vault.
func (s *EncryptedStore) Set(ctx context.Context, path string, secret *vault.Secret) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isLockedUnsafe() {
		return errors.New("vault is locked")
	}

	// Set metadata timestamps
	now := vault.Now()
	if secret.Metadata.CreatedAt == nil {
		secret.Metadata.CreatedAt = now
	}
	secret.Metadata.ModifiedAt = now

	// Serialize secret
	data, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	// Encrypt
	encrypted, err := s.crypto.EncryptString(string(data))
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	s.data.Secrets[path] = encrypted
	s.dirty = true

	if s.autoSave {
		return s.saveData()
	}

	return nil
}

// Delete removes a secret from the vault.
func (s *EncryptedStore) Delete(ctx context.Context, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isLockedUnsafe() {
		return errors.New("vault is locked")
	}

	delete(s.data.Secrets, path)
	s.dirty = true

	if s.autoSave {
		return s.saveData()
	}

	return nil
}

// Exists checks if a secret exists at the given path.
func (s *EncryptedStore) Exists(ctx context.Context, path string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.isLockedUnsafe() {
		return false, errors.New("vault is locked")
	}

	_, ok := s.data.Secrets[path]
	return ok, nil
}

// List returns all secret paths matching the given prefix.
func (s *EncryptedStore) List(ctx context.Context, prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.isLockedUnsafe() {
		return nil, errors.New("vault is locked")
	}

	var paths []string
	for path := range s.data.Secrets {
		if prefix == "" || strings.HasPrefix(path, prefix) {
			paths = append(paths, path)
		}
	}

	sort.Strings(paths)
	return paths, nil
}

// Name returns the provider name.
func (s *EncryptedStore) Name() string {
	return "encrypted"
}

// Capabilities returns the provider capabilities.
func (s *EncryptedStore) Capabilities() vault.Capabilities {
	return vault.Capabilities{
		Read:       true,
		Write:      true,
		Delete:     true,
		List:       true,
		Binary:     true,
		MultiField: true,
	}
}

// Close releases resources and locks the vault.
func (s *EncryptedStore) Close() error {
	return s.Lock()
}

// SecretCount returns the number of secrets in the vault.
func (s *EncryptedStore) SecretCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.data == nil {
		return 0
	}
	return len(s.data.Secrets)
}

// saveMeta saves the vault metadata to disk.
func (s *EncryptedStore) saveMeta() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.metaPath), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.metaPath, data, 0600)
}

// loadMeta loads the vault metadata from disk.
func (s *EncryptedStore) loadMeta() error {
	data, err := os.ReadFile(s.metaPath)
	if err != nil {
		return err
	}

	var meta VaultMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	s.meta = &meta
	return nil
}

// saveData saves the encrypted vault data to disk.
func (s *EncryptedStore) saveData() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.vaultPath), 0700); err != nil {
		return err
	}

	data, err := json.Marshal(s.data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.vaultPath, data, 0600); err != nil {
		return err
	}

	s.dirty = false
	return nil
}

// loadData loads the encrypted vault data from disk.
func (s *EncryptedStore) loadData() error {
	data, err := os.ReadFile(s.vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			// New vault, no data yet
			s.data = &VaultData{
				Secrets: make(map[string]string),
			}
			return nil
		}
		return err
	}

	var vaultData VaultData
	if err := json.Unmarshal(data, &vaultData); err != nil {
		return err
	}

	if vaultData.Secrets == nil {
		vaultData.Secrets = make(map[string]string)
	}

	s.data = &vaultData
	return nil
}

// ChangePassword changes the master password.
func (s *EncryptedStore) ChangePassword(oldPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify old password
	if !s.crypto.VerifyPassword(oldPassword, s.meta.Verification) {
		return errors.New("invalid current password")
	}

	// Create new crypto with new salt
	newCrypto, err := NewCrypto(nil, DefaultArgon2Params())
	if err != nil {
		return fmt.Errorf("failed to create crypto: %w", err)
	}

	newCrypto.Unlock(newPassword)

	// Create new verification blob
	verification, err := newCrypto.CreateVerificationBlob()
	if err != nil {
		newCrypto.Lock()
		return fmt.Errorf("failed to create verification: %w", err)
	}

	// Re-encrypt all secrets with new key
	newSecrets := make(map[string]string)
	for path, encrypted := range s.data.Secrets {
		// Decrypt with old key
		decrypted, err := s.crypto.DecryptString(encrypted)
		if err != nil {
			newCrypto.Lock()
			return fmt.Errorf("failed to decrypt secret %s: %w", path, err)
		}

		// Encrypt with new key
		reEncrypted, err := newCrypto.EncryptString(decrypted)
		if err != nil {
			newCrypto.Lock()
			return fmt.Errorf("failed to re-encrypt secret %s: %w", path, err)
		}

		newSecrets[path] = reEncrypted
	}

	// Update metadata
	s.meta.Salt = newCrypto.Salt()
	s.meta.Argon2Params = newCrypto.Params()
	s.meta.Verification = verification

	// Update data
	s.data.Secrets = newSecrets

	// Replace crypto
	s.crypto.Lock()
	s.crypto = newCrypto

	// Save to disk
	if err := s.saveMeta(); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	if err := s.saveData(); err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	return nil
}
