## Credential Vault Security

### Encryption Architecture

- **Envelope Encryption:** Each credential encrypted with a unique Data Encryption Key (DEK)
- **DEK wrapping:** Each DEK encrypted with the Master Key (KEK)
- **Master Key Derivation:** Argon2id from admin passphrase (set during first-run)
- **At Rest:** AES-256-GCM for all encrypted data
- **In Memory:** Master key protected via `memguard` (mlock'd memory pages)

### Key Hierarchy

```
Admin Passphrase
    |
    v (Argon2id)
Master Key (KEK) -- stored in memguard, never written to disk
    |
    v (AES-256-GCM wrap)
Data Encryption Key (per credential)
    |
    v (AES-256-GCM encrypt)
Credential Data
```

### Key Management

- Master key derived at server startup from passphrase (interactive or env var)
- Key rotation: new master key re-wraps all DEKs without re-encrypting data
- Passphrase change: re-derive master key, re-wrap all DEKs
- Emergency access: sealed key file encrypted to recovery key (optional)

### Credential Access Audit

Every credential access is logged:

| Field | Description |
|-------|-------------|
| Timestamp | When accessed |
| CredentialID | Which credential |
| UserID | Who accessed |
| Action | read, create, update, delete |
| Purpose | "ssh_session", "snmp_scan", "manual_view" |
| SourceIP | Requester's IP address |
