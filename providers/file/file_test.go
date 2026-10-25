package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	osfs "github.com/grokify/oscompat/fs"

	"github.com/plexusone/omnivault/vault"
)

// newJSONProvider returns a provider configured the way VaultFromURI builds
// one, rooted at a "vault" directory inside a temporary directory, along with
// the temporary directory so tests can place files beside the vault.
func newJSONProvider(t *testing.T) (*Provider, string) {
	t.Helper()
	parent := t.TempDir()
	p, err := New(Config{
		Directory:  filepath.Join(parent, "vault"),
		JSONFormat: true,
		Extension:  ".json",
	})
	if err != nil {
		t.Fatal(err)
	}
	return p, parent
}

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	p, _ := newJSONProvider(t)

	if err := p.Set(ctx, "nested/api-key", &vault.Secret{Value: "s3cret"}); err != nil {
		t.Fatal(err)
	}
	got, err := p.Get(ctx, "nested/api-key")
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "s3cret" {
		t.Fatalf("Value = %q, want %q", got.Value, "s3cret")
	}
	ok, err := p.Exists(ctx, "nested/api-key")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Exists = false, want true")
	}
	if err := p.Delete(ctx, "nested/api-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Get(ctx, "nested/api-key"); !errors.Is(err, vault.ErrSecretNotFound) {
		t.Fatalf("Get after Delete: %v, want ErrSecretNotFound", err)
	}
}

func TestPathsOutsideDirectoryRejected(t *testing.T) {
	ctx := context.Background()
	p, parent := newJSONProvider(t)

	// A secret beside the vault directory that a traversing path would reach.
	outside := filepath.Join(parent, "outside.json")
	if err := os.WriteFile(outside, []byte(`{"value":"not-for-the-vault"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"../outside",
		"nested/../../outside",
		filepath.Join(parent, "outside"), // absolute
		"",
	} {
		if _, err := p.Get(ctx, path); !errors.Is(err, vault.ErrInvalidPath) {
			t.Errorf("Get(%q): %v, want ErrInvalidPath", path, err)
		}
		if _, err := p.Exists(ctx, path); !errors.Is(err, vault.ErrInvalidPath) {
			t.Errorf("Exists(%q): %v, want ErrInvalidPath", path, err)
		}
		if err := p.Set(ctx, path, &vault.Secret{Value: "overwritten"}); !errors.Is(err, vault.ErrInvalidPath) {
			t.Errorf("Set(%q): %v, want ErrInvalidPath", path, err)
		}
		if err := p.Delete(ctx, path); !errors.Is(err, vault.ErrInvalidPath) {
			t.Errorf("Delete(%q): %v, want ErrInvalidPath", path, err)
		}
	}

	b, err := os.ReadFile(outside)
	if err != nil {
		t.Fatalf("file outside the vault: %v", err)
	}
	if string(b) != `{"value":"not-for-the-vault"}` {
		t.Fatalf("file outside the vault was modified: %s", b)
	}
}

func TestSymlinkOutsideDirectoryRejected(t *testing.T) {
	ctx := context.Background()
	p, parent := newJSONProvider(t)

	outside := filepath.Join(parent, "outside.json")
	if err := os.WriteFile(outside, []byte(`{"value":"not-for-the-vault"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(parent, "vault", "link.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got, err := p.Get(ctx, "link")
	if err == nil {
		t.Fatalf("Get through a symlink leaving the directory returned %q, want an error", got.Value)
	}
}

func TestGetRefusesFilesOthersCanRead(t *testing.T) {
	if !osfs.PermissionsEnforced() {
		t.Skip("file modes are not enforced on this platform")
	}
	ctx := context.Background()
	p, parent := newJSONProvider(t)

	if err := p.Set(ctx, "root_key", &vault.Secret{Value: "s3cret"}); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(parent, "vault", "root_key.json")

	for _, mode := range []os.FileMode{0o640, 0o644, 0o604} {
		if err := os.Chmod(name, mode); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Get(ctx, "root_key"); !errors.Is(err, ErrInsecurePermissions) {
			t.Fatalf("Get with mode %04o: %v, want ErrInsecurePermissions", mode, err)
		}
	}

	// The opt-out reads the same file.
	lax, err := New(Config{
		Directory:                filepath.Join(parent, "vault"),
		JSONFormat:               true,
		Extension:                ".json",
		AllowInsecurePermissions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := lax.Get(ctx, "root_key")
	if err != nil {
		t.Fatalf("Get with AllowInsecurePermissions: %v", err)
	}
	if got.Value != "s3cret" {
		t.Fatalf("Value = %q, want %q", got.Value, "s3cret")
	}

	// Restoring a private mode makes the file readable again.
	if err := os.Chmod(name, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Get(ctx, "root_key"); err != nil {
		t.Fatalf("Get with mode 0600: %v", err)
	}
}
