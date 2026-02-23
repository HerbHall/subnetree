package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
)

// SeedProxmoxData creates VM and container child devices under the "proxmox-host"
// device and populates resource snapshots. Idempotent: checks existence before insert.
func SeedProxmoxData(ctx context.Context, reconStore *recon.ReconStore, db *sql.DB) error {
	// Find the proxmox-host device.
	host, err := reconStore.GetDeviceByHostname(ctx, "proxmox-host")
	if err != nil || host == nil {
		return fmt.Errorf("proxmox-host device not found (run SeedDemoNetwork first)")
	}

	now := time.Now().UTC()

	type guest struct {
		hostname   string
		deviceType models.DeviceType
		status     models.DeviceStatus
		resource   *recon.ProxmoxResource // nil for stopped VMs
	}

	guests := []guest{
		{
			hostname:   "docker-vm",
			deviceType: models.DeviceTypeVM,
			status:     models.DeviceStatusOnline,
			resource: &recon.ProxmoxResource{
				CPUPercent: 32.5, MemUsedMB: 5120, MemTotalMB: 8192,
				DiskUsedGB: 28, DiskTotalGB: 50, UptimeSec: 432000,
				NetInBytes: 1073741824, NetOutBytes: 536870912,
			},
		},
		{
			hostname:   "pihole-vm",
			deviceType: models.DeviceTypeVM,
			status:     models.DeviceStatusOnline,
			resource: &recon.ProxmoxResource{
				CPUPercent: 5.2, MemUsedMB: 384, MemTotalMB: 1024,
				DiskUsedGB: 3, DiskTotalGB: 10, UptimeSec: 864000,
				NetInBytes: 268435456, NetOutBytes: 134217728,
			},
		},
		{
			hostname:   "windows-vm",
			deviceType: models.DeviceTypeVM,
			status:     models.DeviceStatusOffline,
			resource:   nil, // stopped
		},
		{
			hostname:   "nginx-lxc",
			deviceType: models.DeviceTypeContainer,
			status:     models.DeviceStatusOnline,
			resource: &recon.ProxmoxResource{
				CPUPercent: 8.1, MemUsedMB: 256, MemTotalMB: 512,
				DiskUsedGB: 2, DiskTotalGB: 5, UptimeSec: 864000,
				NetInBytes: 2147483648, NetOutBytes: 4294967296,
			},
		},
		{
			hostname:   "postgres-lxc",
			deviceType: models.DeviceTypeContainer,
			status:     models.DeviceStatusOnline,
			resource: &recon.ProxmoxResource{
				CPUPercent: 18.7, MemUsedMB: 1536, MemTotalMB: 2048,
				DiskUsedGB: 12, DiskTotalGB: 20, UptimeSec: 864000,
				NetInBytes: 536870912, NetOutBytes: 268435456,
			},
		},
	}

	for _, g := range guests {
		// Check if already exists (idempotent).
		existing, findErr := reconStore.FindDeviceByHostnameAndParent(ctx, g.hostname, host.ID)
		if findErr != nil {
			return fmt.Errorf("check existing %s: %w", g.hostname, findErr)
		}

		var deviceID string
		if existing != nil {
			deviceID = existing.ID
		} else {
			deviceID = uuid.New().String()
			dev := &models.Device{
				ID:              deviceID,
				Hostname:        g.hostname,
				DeviceType:      g.deviceType,
				Status:          g.status,
				DiscoveryMethod: models.DiscoveryProxmox,
				ParentDeviceID:  host.ID,
				NetworkLayer:    models.NetworkLayerEndpoint,
				FirstSeen:       now.Add(-3 * 24 * time.Hour),
				LastSeen:        now,
				Location:        "Server Rack",
				Category:        "compute",
			}
			if _, uErr := reconStore.UpsertDevice(ctx, dev); uErr != nil {
				return fmt.Errorf("create device %s: %w", g.hostname, uErr)
			}
		}

		// Upsert resource snapshot for running guests.
		if g.resource != nil {
			g.resource.DeviceID = deviceID
			g.resource.CollectedAt = now
			if rErr := reconStore.UpsertProxmoxResource(ctx, g.resource); rErr != nil {
				return fmt.Errorf("upsert resource %s: %w", g.hostname, rErr)
			}
		}
	}

	return nil
}
