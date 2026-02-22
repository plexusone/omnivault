# Daemon Architecture

The OmniVault CLI uses a daemon (background service) architecture for secure secret management.

## Why a Daemon?

The daemon architecture provides several security benefits:

1. **Session-Based Unlock** - Unlock once, access secrets multiple times without re-entering password
2. **Memory Protection** - Encryption keys stay in daemon memory, not in CLI process
3. **Auto-Lock** - Automatic locking after inactivity
4. **Single Point of Control** - One process manages all vault access

## How It Works

```
┌─────────────┐     Unix Socket      ┌──────────────┐
│  omnivault  │ ◄──────────────────► │  omnivaultd  │
│    (CLI)    │      HTTP/JSON       │   (Daemon)   │
└─────────────┘                      └──────┬───────┘
                                            │
                                            ▼
                                     ┌──────────────┐
                                     │ ~/.omnivault │
                                     │  vault.enc   │
                                     │  vault.meta  │
                                     └──────────────┘
```

1. CLI sends commands to daemon via Unix socket (macOS/Linux) or TCP (Windows)
2. Daemon holds encryption key in memory
3. Daemon performs all cryptographic operations
4. Files on disk are always encrypted

## Communication

### Unix Socket

On macOS and Linux, the daemon listens on a Unix socket:

```
~/.omnivault/omnivaultd.sock
```

Unix sockets provide:

- Local-only access (no network exposure)
- File permission-based security
- Fast IPC performance

### HTTP API

The daemon exposes an HTTP API over the socket:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/status` | GET | Daemon and vault status |
| `/init` | POST | Initialize new vault |
| `/unlock` | POST | Unlock vault |
| `/lock` | POST | Lock vault |
| `/secrets` | GET | List secrets |
| `/secret/:path` | GET | Get secret |
| `/secret/:path` | PUT | Set secret |
| `/secret/:path` | DELETE | Delete secret |
| `/stop` | POST | Stop daemon |

## Lifecycle

### Starting

```bash
omnivault daemon start
```

1. Checks if daemon is already running
2. Starts new process in background
3. Creates Unix socket
4. Writes PID file

### Running

While running, the daemon:

- Listens for CLI commands
- Manages vault lock state
- Resets auto-lock timer on activity
- Holds encryption key in memory (when unlocked)

### Stopping

```bash
omnivault daemon stop
```

1. Sends stop command via socket
2. Daemon locks vault (clears key from memory)
3. Daemon shuts down HTTP server
4. Removes socket and PID files

### Graceful Shutdown

On SIGINT or SIGTERM:

1. Vault is locked
2. Active requests complete
3. Socket is removed
4. Process exits

## Auto-Lock

The daemon automatically locks the vault after a period of inactivity.

### How It Works

- Timer starts when vault is unlocked
- Each vault operation resets the timer
- When timer expires, vault is locked
- Default timeout: 15 minutes

### Activity Reset

These operations reset the auto-lock timer:

- `get` - Reading a secret
- `set` - Writing a secret
- `delete` - Deleting a secret
- `list` - Listing secrets

These operations do NOT reset the timer:

- `status` - Checking status
- `lock` - Manual lock
- `unlock` - Already unlocked

## Files

The daemon creates and manages these files:

| File | Purpose | Permissions |
|------|---------|-------------|
| `~/.omnivault/` | Config directory | 700 |
| `vault.enc` | Encrypted secrets | 600 |
| `vault.meta` | Salt and parameters | 600 |
| `omnivaultd.sock` | Unix socket | 600 |
| `omnivaultd.pid` | Daemon PID | 644 |

## Platform Differences

### macOS / Linux

- Uses Unix socket at `~/.omnivault/omnivaultd.sock`
- Standard Unix permissions apply
- Process daemonization via `Setpgid`
- Graceful shutdown via SIGTERM

### Windows

- Uses TCP on `127.0.0.1:19839`
- Vault files stored in `%LOCALAPPDATA%\OmniVault\`
- Process termination via `Process.Kill()`
- No socket file (TCP-based IPC)

| Feature | macOS/Linux | Windows |
|---------|-------------|---------|
| IPC | Unix Socket | TCP localhost |
| Address | `~/.omnivault/omnivaultd.sock` | `127.0.0.1:19839` |
| Config Dir | `~/.omnivault/` | `%LOCALAPPDATA%\OmniVault\` |
| Shutdown | SIGTERM | Process.Kill() |

## Debugging

Run the daemon in foreground to see logs:

```bash
omnivault daemon run
```

Example output:

```
Starting OmniVault daemon...
INFO daemon started socket=/Users/you/.omnivault/omnivaultd.sock
INFO vault unlocked
INFO secret accessed path=database/password
INFO vault auto-locked due to inactivity
```

## Security Considerations

### Socket Permissions

The Unix socket is created with permissions `600`:

- Only the owner can connect
- Other users cannot access the daemon

### Memory Security

When locked:

- Encryption key is zeroed from memory
- No secrets can be decrypted
- Vault file remains encrypted

### No Network Access

The daemon:

- Never listens on network interfaces
- Only accepts local Unix socket connections
- Cannot be accessed remotely
