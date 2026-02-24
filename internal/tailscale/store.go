package tailscale

import (
	"context"

	"github.com/HerbHall/subnetree/pkg/models"
)

// DeviceStore provides device persistence operations for the Tailscale syncer.
// Implemented by an adapter in the composition root (cmd/subnetree/main.go).
type DeviceStore interface {
	UpsertDevice(ctx context.Context, d *models.Device) (bool, error)
	ListDevices(ctx context.Context, limit, offset int) ([]models.Device, int, error)
	GetDeviceByMAC(ctx context.Context, mac string) (*models.Device, error)
}

// CredentialDecrypter retrieves decrypted credential data from the vault.
// Implemented by the existing vaultDecryptAdapter in main.go.
type CredentialDecrypter interface {
	DecryptCredential(ctx context.Context, id string) (map[string]any, error)
}
