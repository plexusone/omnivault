package omnivault

import (
	"fmt"

	"github.com/plexusone/omnivault/providers/env"
	"github.com/plexusone/omnivault/providers/file"
	"github.com/plexusone/omnivault/providers/memory"
	"github.com/plexusone/omnivault/vault"
)

// newProvider creates a vault provider based on the configuration.
// This function handles built-in providers only. External providers
// should be passed via Config.CustomVault.
func newProvider(config Config) (vault.Vault, error) {
	switch config.Provider {
	case ProviderEnv:
		return newEnvProvider(config)
	case ProviderMemory:
		return newMemoryProvider(config)
	case ProviderFile:
		return newFileProvider(config)
	case "":
		return nil, ErrNoProvider
	default:
		return nil, fmt.Errorf("%w: %s (use CustomVault for external providers)", ErrUnknownScheme, config.Provider)
	}
}

// newEnvProvider creates an environment variable provider.
func newEnvProvider(config Config) (vault.Vault, error) {
	var envConfig env.Config

	if pc, ok := config.ProviderConfig.(env.Config); ok {
		envConfig = pc
	} else if pc, ok := config.ProviderConfig.(*env.Config); ok && pc != nil {
		envConfig = *pc
	}

	return env.NewWithConfig(envConfig), nil
}

// newMemoryProvider creates an in-memory provider.
func newMemoryProvider(config Config) (vault.Vault, error) {
	if secrets, ok := config.ProviderConfig.(map[string]string); ok {
		return memory.NewWithSecrets(secrets), nil
	}
	return memory.New(), nil
}

// newFileProvider creates a file-based provider.
func newFileProvider(config Config) (vault.Vault, error) {
	var fileConfig file.Config

	if pc, ok := config.ProviderConfig.(file.Config); ok {
		fileConfig = pc
	} else if pc, ok := config.ProviderConfig.(*file.Config); ok && pc != nil {
		fileConfig = *pc
	} else {
		return nil, fmt.Errorf("file provider requires file.Config in ProviderConfig")
	}

	return file.New(fileConfig)
}

// EnvConfig is an alias for env.Config for convenience.
type EnvConfig = env.Config

// FileConfig is an alias for file.Config for convenience.
type FileConfig = file.Config
