package recon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WiFiAPSyncer synchronises WiFi AP clients as child devices in the recon store
// and collects their signal/traffic snapshots.
type WiFiAPSyncer struct {
	store  *ReconStore
	oui    OUIResolver
	logger *zap.Logger
}

// NewWiFiAPSyncer creates a new WiFiAPSyncer.
func NewWiFiAPSyncer(store *ReconStore, oui OUIResolver, logger *zap.Logger) *WiFiAPSyncer {
	return &WiFiAPSyncer{store: store, oui: oui, logger: logger}
}

// WiFiAPSyncResult summarises the outcome of a WiFi AP sync operation.
type WiFiAPSyncResult struct {
	APsChecked   int `json:"aps_checked"`
	ClientsFound int `json:"clients_found"`
	Created      int `json:"created"`
	Updated      int `json:"updated"`
}

// Sync enumerates AP clients and upserts them as child devices under the
// corresponding AP device.
func (s *WiFiAPSyncer) Sync(ctx context.Context, enumerator APClientEnumerator) (result *WiFiAPSyncResult, err error) {
	result = &WiFiAPSyncResult{}

	if !enumerator.Available() {
		return result, nil
	}

	clients, enumErr := enumerator.Enumerate(ctx)
	if enumErr != nil {
		return nil, fmt.Errorf("enumerate AP clients: %w", enumErr)
	}
	result.ClientsFound = len(clients)

	now := time.Now().UTC()

	// Group clients by AP BSSID so we can track seen devices per AP.
	type apGroup struct {
		apDeviceID string
		apSSID     string
		clients    []APClientInfo
	}
	groups := make(map[string]*apGroup)

	for i := range clients {
		bssid := clients[i].APBSSID
		if bssid == "" {
			continue
		}

		g, ok := groups[bssid]
		if !ok {
			g = &apGroup{apSSID: clients[i].APSSID}
			groups[bssid] = g
		}
		g.clients = append(g.clients, clients[i])
	}

	result.APsChecked = len(groups)

	for bssid, g := range groups {
		// Find or create the AP device.
		apDevice, apErr := s.store.GetDeviceByMAC(ctx, bssid)
		if apErr != nil && !errors.Is(apErr, sql.ErrNoRows) {
			s.logger.Error("failed to look up AP device by MAC",
				zap.String("bssid", bssid), zap.Error(apErr))
			continue
		}
		if apDevice == nil {
			// Create a new AP device.
			apDevice = &models.Device{
				ID:              uuid.New().String(),
				Hostname:        g.apSSID,
				MACAddress:      bssid,
				DeviceType:      models.DeviceTypeAccessPoint,
				Status:          models.DeviceStatusOnline,
				DiscoveryMethod: models.DiscoveryWiFi,
				ConnectionType:  models.ConnectionWiFi,
				NetworkLayer:    models.NetworkLayerAccess,
				FirstSeen:       now,
				LastSeen:        now,
			}
			if s.oui != nil {
				apDevice.Manufacturer = s.oui.Lookup(bssid)
			}
			if _, uErr := s.store.UpsertDevice(ctx, apDevice); uErr != nil {
				s.logger.Error("failed to create AP device",
					zap.String("bssid", bssid), zap.Error(uErr))
				continue
			}
		}
		g.apDeviceID = apDevice.ID

		seenDeviceIDs := make(map[string]bool)

		for j := range g.clients {
			client := &g.clients[j]
			if client.MACAddress == "" {
				continue
			}

			deviceID, created, syncErr := s.upsertClientDevice(ctx, client, apDevice.ID, now)
			if syncErr != nil {
				s.logger.Error("upsert wifi client device",
					zap.String("mac", client.MACAddress), zap.Error(syncErr))
				continue
			}
			seenDeviceIDs[deviceID] = true

			if created {
				result.Created++
			} else {
				result.Updated++
			}

			// Upsert signal snapshot.
			snap := &WiFiClientSnapshot{
				DeviceID:     deviceID,
				SignalDBm:    client.Signal,
				SignalAvgDBm: client.SignalAverage,
				ConnectedSec: int64(client.Connected.Seconds()),
				InactiveSec:  int64(client.Inactive.Seconds()),
				RxBitrate:    client.RxBitrate,
				TxBitrate:    client.TxBitrate,
				RxBytes:      client.RxBytes,
				TxBytes:      client.TxBytes,
				APBSSID:      client.APBSSID,
				APSSID:       client.APSSID,
				CollectedAt:  now,
			}
			if snapErr := s.store.UpsertWiFiClient(ctx, snap); snapErr != nil {
				s.logger.Error("upsert wifi client snapshot",
					zap.String("device_id", deviceID), zap.Error(snapErr))
			}
		}

		// Mark unseen wifi children under this AP as offline.
		if markErr := s.markUnseen(ctx, apDevice.ID, seenDeviceIDs); markErr != nil {
			s.logger.Warn("failed to mark unseen wifi devices offline", zap.Error(markErr))
		}
	}

	return result, nil
}

// upsertClientDevice creates or updates a child device record for a WiFi client.
// Returns the device ID, whether it was newly created, and any error.
func (s *WiFiAPSyncer) upsertClientDevice(
	ctx context.Context, client *APClientInfo, apDeviceID string, now time.Time,
) (deviceID string, created bool, err error) {
	existing, lErr := s.store.GetDeviceByMAC(ctx, client.MACAddress)
	if lErr != nil && !errors.Is(lErr, sql.ErrNoRows) {
		return "", false, fmt.Errorf("find existing device: %w", lErr)
	}

	if existing != nil {
		// Update status and last_seen.
		if uErr := s.store.UpdateDeviceStatus(ctx, existing.ID, models.DeviceStatusOnline, now); uErr != nil {
			return "", false, fmt.Errorf("update device status: %w", uErr)
		}
		// Ensure connection type is wifi.
		if existing.ConnectionType != models.ConnectionWiFi {
			if cErr := s.store.UpdateDeviceConnectionType(ctx, existing.ID, models.ConnectionWiFi); cErr != nil {
				s.logger.Warn("failed to update connection type to wifi",
					zap.String("device_id", existing.ID), zap.Error(cErr))
			}
		}
		return existing.ID, false, nil
	}

	// Create new device.
	manufacturer := ""
	if s.oui != nil {
		manufacturer = s.oui.Lookup(client.MACAddress)
	}

	dev := &models.Device{
		ID:                       uuid.New().String(),
		MACAddress:               client.MACAddress,
		Manufacturer:             manufacturer,
		DeviceType:               models.DeviceTypeUnknown,
		Status:                   models.DeviceStatusOnline,
		DiscoveryMethod:          models.DiscoveryWiFi,
		ConnectionType:           models.ConnectionWiFi,
		ParentDeviceID:           apDeviceID,
		NetworkLayer:             models.NetworkLayerEndpoint,
		ClassificationConfidence: 100,
		ClassificationSource:     "wifi_ap",
		FirstSeen:                now,
		LastSeen:                 now,
	}
	if _, uErr := s.store.UpsertDevice(ctx, dev); uErr != nil {
		return "", false, fmt.Errorf("create device: %w", uErr)
	}
	return dev.ID, true, nil
}

// markUnseen sets offline status for WiFi-discovered devices under the AP
// that were not observed during this sync cycle.
func (s *WiFiAPSyncer) markUnseen(ctx context.Context, apDeviceID string, seen map[string]bool) error {
	devices, err := s.store.FindChildDevicesByDiscovery(ctx, apDeviceID, string(models.DiscoveryWiFi))
	if err != nil {
		return fmt.Errorf("find wifi children: %w", err)
	}
	for i := range devices {
		if !seen[devices[i].ID] && devices[i].Status == models.DeviceStatusOnline {
			if markErr := s.store.MarkDeviceOffline(ctx, devices[i].ID); markErr != nil {
				s.logger.Warn("failed to mark unseen wifi device offline",
					zap.String("device_id", devices[i].ID), zap.Error(markErr))
			}
		}
	}
	return nil
}
