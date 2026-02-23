package recon

import (
	"context"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProxmoxSyncer synchronises Proxmox VMs/containers as child devices in the
// recon store and collects their resource snapshots.
type ProxmoxSyncer struct {
	store  *ReconStore
	logger *zap.Logger
}

// NewProxmoxSyncer creates a new ProxmoxSyncer.
func NewProxmoxSyncer(store *ReconStore, logger *zap.Logger) *ProxmoxSyncer {
	return &ProxmoxSyncer{store: store, logger: logger}
}

// ProxmoxSyncResult summarises the outcome of a Proxmox sync operation.
type ProxmoxSyncResult struct {
	NodesScanned int `json:"nodes_scanned"`
	VMsFound     int `json:"vms_found"`
	LXCsFound    int `json:"lxcs_found"`
	Created      int `json:"created"`
	Updated      int `json:"updated"`
}

// Sync enumerates nodes, VMs and containers from the given collector and
// upserts them as child devices under hostDeviceID.
func (s *ProxmoxSyncer) Sync(ctx context.Context, collector *ProxmoxCollector, hostDeviceID string) (*ProxmoxSyncResult, error) {
	result := &ProxmoxSyncResult{}

	nodes, err := collector.CollectNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect nodes: %w", err)
	}
	result.NodesScanned = len(nodes)

	now := time.Now().UTC()
	seenDeviceIDs := make(map[string]bool)

	for _, node := range nodes {
		// Process QEMU VMs.
		vms, vmErr := collector.CollectVMs(ctx, node.Node)
		if vmErr != nil {
			s.logger.Warn("failed to collect VMs for node", zap.String("node", node.Node), zap.Error(vmErr))
			continue
		}
		result.VMsFound += len(vms)

		for _, vm := range vms {
			deviceID, created, uErr := s.upsertGuestDevice(ctx, vm.Name, models.DeviceTypeVM, vm.Status, hostDeviceID, now)
			if uErr != nil {
				s.logger.Error("upsert VM device", zap.String("name", vm.Name), zap.Error(uErr))
				continue
			}
			seenDeviceIDs[deviceID] = true
			if created {
				result.Created++
			} else {
				result.Updated++
			}

			// Collect resource status for running VMs.
			if vm.Status == "running" {
				status, sErr := collector.CollectVMStatus(ctx, node.Node, vm.VMID)
				if sErr != nil {
					s.logger.Debug("failed to collect VM status", zap.String("name", vm.Name), zap.Error(sErr))
					continue
				}
				s.upsertResourceSnapshot(ctx, deviceID, status, now)
			}
		}

		// Process LXC containers.
		containers, lxcErr := collector.CollectContainers(ctx, node.Node)
		if lxcErr != nil {
			s.logger.Warn("failed to collect containers for node", zap.String("node", node.Node), zap.Error(lxcErr))
			continue
		}
		result.LXCsFound += len(containers)

		for _, ct := range containers {
			deviceID, created, uErr := s.upsertGuestDevice(ctx, ct.Name, models.DeviceTypeContainer, ct.Status, hostDeviceID, now)
			if uErr != nil {
				s.logger.Error("upsert container device", zap.String("name", ct.Name), zap.Error(uErr))
				continue
			}
			seenDeviceIDs[deviceID] = true
			if created {
				result.Created++
			} else {
				result.Updated++
			}

			if ct.Status == "running" {
				status, sErr := collector.CollectContainerStatus(ctx, node.Node, ct.VMID)
				if sErr != nil {
					s.logger.Debug("failed to collect container status", zap.String("name", ct.Name), zap.Error(sErr))
					continue
				}
				s.upsertResourceSnapshot(ctx, deviceID, status, now)
			}
		}
	}

	// Mark previously-discovered Proxmox children that were not seen this cycle as offline.
	if err := s.markUnseen(ctx, hostDeviceID, seenDeviceIDs); err != nil {
		s.logger.Warn("failed to mark unseen proxmox devices offline", zap.Error(err))
	}

	return result, nil
}

// upsertGuestDevice creates or updates a child device record. Returns the
// device ID, whether it was newly created, and any error.
func (s *ProxmoxSyncer) upsertGuestDevice(
	ctx context.Context, hostname string, deviceType models.DeviceType,
	pveStatus, parentID string, now time.Time,
) (deviceID string, created bool, err error) {
	status := models.DeviceStatusOnline
	if pveStatus != "running" {
		status = models.DeviceStatusOffline
	}

	// Look for an existing device with matching hostname and parent.
	existing, lErr := s.store.FindDeviceByHostnameAndParent(ctx, hostname, parentID)
	if lErr != nil {
		return "", false, fmt.Errorf("find existing device: %w", lErr)
	}

	if existing != nil {
		// Update status and last_seen directly (avoids UpsertDevice's MAC/IP lookup).
		if uErr := s.store.UpdateDeviceStatus(ctx, existing.ID, status, now); uErr != nil {
			return "", false, fmt.Errorf("update device: %w", uErr)
		}
		return existing.ID, false, nil
	}

	// Create new device.
	dev := &models.Device{
		ID:              uuid.New().String(),
		Hostname:        hostname,
		DeviceType:      deviceType,
		Status:          status,
		DiscoveryMethod: models.DiscoveryProxmox,
		ParentDeviceID:  parentID,
		NetworkLayer:    models.NetworkLayerEndpoint,
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, uErr := s.store.UpsertDevice(ctx, dev); uErr != nil {
		return "", false, fmt.Errorf("create device: %w", uErr)
	}
	return dev.ID, true, nil
}

// upsertResourceSnapshot stores a resource snapshot for a device.
func (s *ProxmoxSyncer) upsertResourceSnapshot(ctx context.Context, deviceID string, status *ProxmoxResourceStatus, now time.Time) {
	r := &ProxmoxResource{
		DeviceID:    deviceID,
		CPUPercent:  status.CPUPercent,
		MemUsedMB:   status.MemUsedMB,
		MemTotalMB:  status.MemTotalMB,
		DiskUsedGB:  status.DiskUsedGB,
		DiskTotalGB: status.DiskTotalGB,
		UptimeSec:   status.UptimeSec,
		NetInBytes:  status.NetInBytes,
		NetOutBytes: status.NetOutBytes,
		CollectedAt: now,
	}
	if err := s.store.UpsertProxmoxResource(ctx, r); err != nil {
		s.logger.Error("upsert resource snapshot", zap.String("device_id", deviceID), zap.Error(err))
	}
}

// markUnseen sets offline status for Proxmox-discovered devices under the host
// that were not observed during this sync cycle.
func (s *ProxmoxSyncer) markUnseen(ctx context.Context, hostDeviceID string, seen map[string]bool) error {
	devices, err := s.store.FindChildDevicesByDiscovery(ctx, hostDeviceID, string(models.DiscoveryProxmox))
	if err != nil {
		return fmt.Errorf("find proxmox children: %w", err)
	}
	for _, d := range devices {
		if !seen[d.ID] && d.Status == models.DeviceStatusOnline {
			if markErr := s.store.MarkDeviceOffline(ctx, d.ID); markErr != nil {
				s.logger.Warn("failed to mark unseen device offline", zap.String("device_id", d.ID), zap.Error(markErr))
			}
		}
	}
	return nil
}
