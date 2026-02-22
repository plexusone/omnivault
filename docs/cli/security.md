# Security Model

OmniVault uses industry-standard cryptographic practices to protect your secrets.

## Encryption

### Algorithm

**AES-256-GCM** (Galois/Counter Mode)

- 256-bit key size
- Authenticated encryption (integrity + confidentiality)
- Random 12-byte nonce per secret
- NIST-approved algorithm

### Key Derivation

**Argon2id** - Winner of the Password Hashing Competition

| Parameter | Value | Purpose |
|-----------|-------|---------|
| Time | 3 iterations | CPU cost |
| Memory | 64 MB | Memory cost (GPU resistance) |
| Threads | 4 | Parallelism |
| Key Length | 32 bytes | 256-bit key |
| Salt | 32 bytes | Random per vault |

These parameters follow [OWASP recommendations](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) for password hashing.

### Why Argon2id?

Argon2id is resistant to:

- **GPU attacks** - High memory requirement makes parallel attacks expensive
- **ASIC attacks** - Memory-hard function is difficult to optimize in hardware
- **Side-channel attacks** - Data-independent memory access pattern
- **Time-memory tradeoff** - Hybrid approach resists both

## Password Handling

### Never Stored

The master password is:

- Used only to derive the encryption key
- Never written to disk
- Never logged or displayed
- Cleared from memory after key derivation

### Verification

Password correctness is verified by:

1. Deriving key from entered password
2. Attempting to decrypt a known "magic" value
3. Using constant-time comparison to prevent timing attacks

```go
// Constant-time comparison prevents timing attacks
subtle.ConstantTimeCompare(decrypted, expectedMagic)
```

### Minimum Requirements

- Minimum 8 characters
- No maximum length
- Any characters allowed

!!! tip "Strong Passwords"
    Use a passphrase of 4+ random words for better security and memorability.

## Storage

### File Structure

**macOS / Linux:**

```
~/.omnivault/
├── vault.enc           # Encrypted secrets
├── vault.meta          # Unencrypted metadata
├── omnivaultd.sock     # Unix socket (runtime)
└── omnivaultd.pid      # PID file (runtime)
```

**Windows:**

```
%LOCALAPPDATA%\OmniVault\
├── vault.enc           # Encrypted secrets
├── vault.meta          # Unencrypted metadata
└── omnivaultd.pid      # PID file (runtime)
```

### vault.meta (Unencrypted)

Contains non-sensitive metadata:

```json
{
  "version": 1,
  "created_at": "2024-01-15T10:00:00Z",
  "salt": "base64-encoded-32-bytes",
  "argon2_params": {
    "time": 3,
    "memory": 65536,
    "threads": 4,
    "key_len": 32
  },
  "verification": "base64-encrypted-magic"
}
```

The `verification` field is an encrypted known value used to verify passwords.

### vault.enc (Encrypted)

Contains encrypted secrets:

```json
{
  "secrets": {
    "path/to/secret": "base64-nonce+ciphertext+tag"
  }
}
```

Each secret value is:

1. Serialized to JSON
2. Encrypted with AES-256-GCM
3. Base64 encoded

### File Permissions

| File | Mode | Description |
|------|------|-------------|
| `~/.omnivault/` | 700 | Owner only |
| `vault.enc` | 600 | Owner read/write |
| `vault.meta` | 600 | Owner read/write |
| `omnivaultd.sock` | 600 | Owner only |

## Memory Security

### Key Handling

When locked:

```go
// Zero out key before releasing
for i := range key {
    key[i] = 0
}
key = nil
```

### Daemon Isolation

- Encryption key exists only in daemon memory
- CLI never sees the raw key
- Key is zeroed on lock or shutdown

## Attack Resistance

### Brute Force

Argon2id parameters make brute force expensive:

- Each attempt requires 64 MB RAM
- Each attempt takes ~1 second on modern hardware
- Cannot be parallelized efficiently

### Offline Attacks

If an attacker obtains `vault.enc` and `vault.meta`:

- Must still brute force the password
- Argon2id parameters are public but still protective
- Strong password makes attack infeasible

### Replay Attacks

Each encryption uses a random nonce:

- Same plaintext produces different ciphertext
- Replaying old ciphertext is detectable (wrong nonce)

### Tampering

AES-GCM provides authentication:

- Any modification to ciphertext fails decryption
- Cannot alter secrets without the key

## Best Practices

### Password Selection

- Use a strong, unique password
- Consider a passphrase (4+ random words)
- Use a password manager for the master password

### Physical Security

- Lock your workstation when away
- Use full disk encryption
- Don't leave vault unlocked unnecessarily

### Backup

- Back up `~/.omnivault/` directory
- Store backups securely (encrypted)
- Remember: backup is useless without password

### Regular Rotation

Consider changing your master password periodically:

```bash
# Future feature
omnivault change-password
```

## Threat Model

### Protected Against

- Unauthorized file access (secrets encrypted)
- Memory dumps of CLI process (key in daemon)
- Network attacks (Unix socket only)
- Tampering (authenticated encryption)
- Brute force (Argon2id)

### Not Protected Against

- Keyloggers capturing password
- Compromised daemon process
- Physical access to unlocked machine
- Forgotten password (no recovery)
- Sophisticated malware with root access

## Comparison

| Feature | OmniVault | macOS Keychain | 1Password |
|---------|-----------|----------------|-----------|
| Encryption | AES-256-GCM | AES-256-GCM | AES-256-GCM |
| KDF | Argon2id | PBKDF2 | Argon2id |
| Open Source | Yes | No | No |
| Local Only | Yes | Yes | No |
| Cross-Platform | Yes | macOS only | Yes |
