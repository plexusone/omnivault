# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.0] - 2026-01-10

### Security

- Secrets encrypted at rest using AES-256-GCM
- Master password never stored, only used for key derivation
- Constant-time password verification to prevent timing attacks
- Keys zeroed from memory on vault lock

### Added

- CLI tool (`cmd/omnivault`) for local secret management
- Encrypted local store with AES-256-GCM encryption (`internal/store`)
- Argon2id key derivation with OWASP-recommended parameters
- Daemon server with Unix socket IPC (`internal/daemon`)
- Daemon client library for IPC (`internal/client`)
- Platform-specific path configuration (`internal/config`)
- Auto-lock with configurable inactivity timeout (default: 15 minutes)
- Master password change with automatic re-encryption
- CLI commands: `init`, `unlock`, `lock`, `status`, `get`, `set`, `list`, `delete`
- Daemon commands: `daemon start`, `daemon stop`, `daemon status`, `daemon run`
- Secure password input without terminal echo
- Integration tests for daemon and encrypted store
- Windows daemon support via TCP on localhost (`127.0.0.1:19839`)
- Cross-platform IPC: Unix socket on macOS/Linux, TCP on Windows

### Changed

- Go version updated to 1.24.0

## [v0.1.0] - 2025-01-01

### Added

- Core `vault.Vault` interface for secret management
- Built-in providers: environment variables, file-based, in-memory
- URI-based secret resolution with `Resolver`
- Client API with `Get`, `Set`, `Delete`, `List`, `Exists` operations
- Extensible provider architecture for external modules
- Secret metadata support with tags and timestamps
- Multi-field secrets support

[unreleased]: https://github.com/agentplexus/omnivault/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/agentplexus/omnivault/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/agentplexus/omnivault/releases/tag/v0.1.0
