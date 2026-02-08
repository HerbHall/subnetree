package vault

// Event topics published by the Vault module.
const (
	TopicVaultStatusChanged = "vault.status.changed"
	TopicCredentialCreated  = "vault.credential.created"
	TopicCredentialUpdated  = "vault.credential.updated"
	TopicCredentialDeleted  = "vault.credential.deleted"
	TopicKeysRotated        = "vault.keys.rotated"
)
