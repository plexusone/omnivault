// Package file provides a file-based vault implementation.
// Secrets are stored as individual files in a directory.
//
// Usage:
//
//	v, err := file.New(file.Config{
//	    Directory: "/path/to/secrets",
//	})
//	secret, err := v.Get(ctx, "api-key")  // reads /path/to/secrets/api-key
package file

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	osfs "github.com/grokify/oscompat/fs"

	"github.com/plexusone/omnivault/vault"
)

// Config holds configuration for the file provider.
type Config struct {
	// Directory is the base directory for storing secrets.
	Directory string

	// Extension is the file extension for secret files (default: none).
	Extension string

	// JSONFormat stores secrets as JSON with metadata (default: false, plain text).
	JSONFormat bool

	// FileMode is the permission mode for secret files (default: 0600).
	FileMode os.FileMode

	// DirMode is the permission mode for directories (default: 0700).
	DirMode os.FileMode

	// ReadOnly prevents write and delete operations.
	ReadOnly bool
}

// Provider implements vault.Vault with file-based storage.
type Provider struct {
	config Config
}

// New creates a new file provider with the given configuration.
func New(config Config) (*Provider, error) {
	if config.Directory == "" {
		return nil, errors.New("directory is required")
	}

	// Set defaults
	if config.FileMode == 0 {
		config.FileMode = osfs.PrivateFilePerm
	}
	if config.DirMode == 0 {
		config.DirMode = osfs.PrivateDirPerm
	}

	// Create directory if it doesn't exist
	if !config.ReadOnly {
		if err := os.MkdirAll(config.Directory, config.DirMode); err != nil {
			return nil, err
		}
	}

	return &Provider{config: config}, nil
}

// filename returns the name of a secret's file relative to the directory. It
// rejects a path that is empty, absolute, or climbs out of the directory with
// "..", so that a secret path can only name a file inside the directory.
func (p *Provider) filename(op, path string) (string, error) {
	name := path + p.config.Extension
	if path == "" || !filepath.IsLocal(name) {
		return "", vault.NewVaultError(op, path, p.Name(), vault.ErrInvalidPath)
	}
	return name, nil
}

// openRoot opens the directory as an [os.Root], which refuses any access that
// would leave it, including through a symbolic link that points outside.
func (p *Provider) openRoot(op, path string) (*os.Root, error) {
	root, err := os.OpenRoot(p.config.Directory)
	if err != nil {
		return nil, vault.NewVaultError(op, path, p.Name(), err)
	}
	return root, nil
}

// closeRoot closes root, keeping err if it is already set.
func closeRoot(root *os.Root, err *error) {
	if cerr := root.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}

// Get retrieves a secret from a file.
func (p *Provider) Get(ctx context.Context, path string) (_ *vault.Secret, err error) {
	name, err := p.filename("Get", path)
	if err != nil {
		return nil, err
	}
	root, err := p.openRoot("Get", path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, vault.NewVaultError("Get", path, p.Name(), vault.ErrSecretNotFound)
		}
		return nil, err
	}
	defer closeRoot(root, &err)

	data, err := root.ReadFile(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, vault.NewVaultError("Get", path, p.Name(), vault.ErrSecretNotFound)
		}
		return nil, vault.NewVaultError("Get", path, p.Name(), err)
	}

	var secret *vault.Secret

	if p.config.JSONFormat {
		secret = &vault.Secret{}
		if err := json.Unmarshal(data, secret); err != nil {
			// Fall back to treating as plain text if JSON parsing fails
			secret = &vault.Secret{Value: string(data)}
		}
	} else {
		secret = &vault.Secret{Value: string(data)}
	}

	secret.Metadata.Provider = p.Name()
	secret.Metadata.Path = path

	// Add file info to metadata
	if info, err := root.Stat(name); err == nil {
		modTime := info.ModTime()
		secret.Metadata.ModifiedAt = &vault.Timestamp{Time: modTime}
	}

	return secret, nil
}

// Set stores a secret to a file.
func (p *Provider) Set(ctx context.Context, path string, secret *vault.Secret) (err error) {
	if p.config.ReadOnly {
		return vault.NewVaultError("Set", path, p.Name(), vault.ErrReadOnly)
	}

	name, err := p.filename("Set", path)
	if err != nil {
		return err
	}
	root, err := p.openRoot("Set", path)
	if err != nil {
		return err
	}
	defer closeRoot(root, &err)

	// Ensure parent directory exists
	if dir := filepath.Dir(name); dir != "." {
		if err := root.MkdirAll(dir, p.config.DirMode); err != nil {
			return vault.NewVaultError("Set", path, p.Name(), err)
		}
	}

	var data []byte

	if p.config.JSONFormat {
		data, err = json.MarshalIndent(secret, "", "  ")
		if err != nil {
			return vault.NewVaultError("Set", path, p.Name(), err)
		}
	} else {
		data = secret.Bytes()
	}

	if err := root.WriteFile(name, data, p.config.FileMode); err != nil {
		return vault.NewVaultError("Set", path, p.Name(), err)
	}

	return nil
}

// Delete removes a secret file.
func (p *Provider) Delete(ctx context.Context, path string) (err error) {
	if p.config.ReadOnly {
		return vault.NewVaultError("Delete", path, p.Name(), vault.ErrReadOnly)
	}

	name, err := p.filename("Delete", path)
	if err != nil {
		return err
	}
	root, err := p.openRoot("Delete", path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // No directory, so nothing to delete
		}
		return err
	}
	defer closeRoot(root, &err)

	if err := root.Remove(name); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, not an error
		}
		return vault.NewVaultError("Delete", path, p.Name(), err)
	}

	return nil
}

// Exists checks if a secret file exists.
func (p *Provider) Exists(ctx context.Context, path string) (_ bool, err error) {
	name, err := p.filename("Exists", path)
	if err != nil {
		return false, err
	}
	root, err := p.openRoot("Exists", path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer closeRoot(root, &err)

	_, err = root.Stat(name)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, vault.NewVaultError("Exists", path, p.Name(), err)
}

// List returns all secret paths matching the prefix.
func (p *Provider) List(ctx context.Context, prefix string) ([]string, error) {
	var results []string

	err := filepath.WalkDir(p.config.Directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Get relative path
		rel, err := filepath.Rel(p.config.Directory, path)
		if err != nil {
			return err
		}

		// Remove extension if configured
		if p.config.Extension != "" {
			rel = strings.TrimSuffix(rel, p.config.Extension)
		}

		// Filter by prefix
		if strings.HasPrefix(rel, prefix) {
			results = append(results, rel)
		}

		return nil
	})

	if err != nil {
		return nil, vault.NewVaultError("List", prefix, p.Name(), err)
	}

	return results, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "file"
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() vault.Capabilities {
	return vault.Capabilities{
		Read:       true,
		Write:      !p.config.ReadOnly,
		Delete:     !p.config.ReadOnly,
		List:       true,
		Binary:     true,
		MultiField: p.config.JSONFormat,
	}
}

// Close is a no-op for the file provider.
func (p *Provider) Close() error {
	return nil
}

// Ensure Provider implements vault.Vault.
var _ vault.Vault = (*Provider)(nil)
