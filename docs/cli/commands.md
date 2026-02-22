# CLI Commands Reference

Complete reference for all `omnivault` commands.

## Vault Commands

### init

Initialize a new vault with a master password.

```bash
omnivault init
```

- Prompts for master password (minimum 8 characters)
- Prompts to confirm password
- Creates encrypted vault at `~/.omnivault/`
- Vault is unlocked after initialization

!!! note "Requires Daemon"
    The daemon must be running before initialization.

### unlock

Unlock the vault with the master password.

```bash
omnivault unlock
```

- Prompts for master password
- Vault stays unlocked until locked or auto-lock timeout

### lock

Lock the vault immediately.

```bash
omnivault lock
```

- Clears encryption key from memory
- Secrets are inaccessible until unlocked

### status

Show vault and daemon status.

```bash
omnivault status
```

Example output:

```
Daemon: running
Uptime: 1h23m45s
Vault: unlocked
Secrets: 12
Unlocked at: 2024-01-15 09:00:00
```

Status fields:

| Field | Description |
|-------|-------------|
| Daemon | `running` or `not running` |
| Uptime | Time since daemon started |
| Vault | `locked`, `unlocked`, or `not initialized` |
| Secrets | Number of stored secrets (when unlocked) |
| Unlocked at | Timestamp of last unlock |

## Secret Commands

### get

Retrieve a secret value.

```bash
omnivault get <path>
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `path` | Secret path (e.g., `database/password`) |

**Examples:**

```bash
omnivault get api/key
omnivault get database/credentials
```

**Output:**

- Prints the secret value to stdout
- If the secret has fields, prints each field on a separate line

### set

Store a secret.

```bash
omnivault set <path> [value]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `path` | Secret path (e.g., `database/password`) |
| `value` | Optional secret value |

If value is not provided, you'll be prompted to enter it (input is hidden).

**Examples:**

```bash
# Prompted input (recommended for sensitive values)
omnivault set database/password

# Direct value
omnivault set config/timeout 30

# Piped input
echo "my-secret" | omnivault set api/key
```

### list

List all secrets or filter by prefix.

```bash
omnivault list [prefix]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `prefix` | Optional path prefix filter |

**Examples:**

```bash
# List all secrets
omnivault list

# List secrets under database/
omnivault list database/
```

**Output:**

```
database/password (value+fields)
database/username
api/key [production, v2]
config/timeout

4 secret(s)
```

Indicators:

- `(value+fields)` - Secret has both value and fields
- `(fields)` - Secret has only fields
- `[tag1, tag2]` - Secret tags

### delete

Delete a secret.

```bash
omnivault delete <path>
```

**Aliases:** `rm`

**Arguments:**

| Argument | Description |
|----------|-------------|
| `path` | Secret path to delete |

Prompts for confirmation before deletion.

**Examples:**

```bash
omnivault delete api/old-key
omnivault rm database/test
```

## Daemon Commands

### daemon start

Start the daemon in background.

```bash
omnivault daemon start
```

- Starts the daemon as a background process
- Creates Unix socket at `~/.omnivault/omnivaultd.sock`
- Writes PID to `~/.omnivault/omnivaultd.pid`

### daemon stop

Stop the daemon.

```bash
omnivault daemon stop
```

- Locks the vault before stopping
- Removes socket and PID files

### daemon status

Show daemon status.

```bash
omnivault daemon status
```

### daemon run

Run the daemon in foreground.

```bash
omnivault daemon run
```

Useful for debugging. Press Ctrl+C to stop.

## Other Commands

### version

Show version information.

```bash
omnivault version
```

### help

Show help information.

```bash
omnivault help
omnivault -h
omnivault --help
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Error (message printed to stderr) |

## Environment Variables

Currently, the CLI does not use environment variables for configuration. All settings use defaults:

| Setting | Default |
|---------|---------|
| Config directory | `~/.omnivault/` |
| Auto-lock timeout | 15 minutes |
