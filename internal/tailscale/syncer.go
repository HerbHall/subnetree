package tailscale

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// SyncResult summarises a single sync cycle.
type SyncResult struct {
	DevicesFound int `json:"devices_found"`
	Created      int `json:"created"`
	Updated      int `json:"updated"`
	Unchanged    int `json:"unchanged"`
}

// Syncer merges Tailscale device data into the SubNetree device store.
type Syncer struct {
	store  DeviceStore
	logger *zap.Logger
}

// NewSyncer creates a Syncer with the given store and logger.
func NewSyncer(store DeviceStore, logger *zap.Logger) *Syncer {
	return &Syncer{store: store, logger: logger}
}

// Sync fetches devices from the Tailscale API and merges them into the store.
func (s *Syncer) Sync(ctx context.Context, client *TailscaleClient) (result *SyncResult, err error) {
	tsDevices, err := client.ListDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tailscale devices: %w", err)
	}

	existing, _, err := s.store.ListDevices(ctx, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("list existing devices: %w", err)
	}

	// Build lookup indexes for merge matching.
	byHostname := make(map[string]*models.Device, len(existing))
	byIP := make(map[string]*models.Device, len(existing))
	for i := range existing {
		d := &existing[i]
		if d.Hostname != "" {
			byHostname[strings.ToLower(d.Hostname)] = d
		}
		for _, ip := range d.IPAddresses {
			byIP[ip] = d
		}
	}

	res := &SyncResult{DevicesFound: len(tsDevices)}
	seenIDs := make(map[string]bool) // tracks existing device IDs matched

	for i := range tsDevices {
		tsDev := &tsDevices[i]
		shortHostname := extractShortHostname(tsDev.Name)

		// Attempt match: hostname (case-insensitive), then IP overlap.
		var matched *models.Device
		if shortHostname != "" {
			matched = byHostname[strings.ToLower(shortHostname)]
		}
		if matched == nil {
			for _, addr := range tsDev.Addresses {
				if d, ok := byIP[addr]; ok {
					matched = d
					break
				}
			}
		}

		device := buildDevice(tsDev, shortHostname, matched)

		created, upsertErr := s.store.UpsertDevice(ctx, device)
		if upsertErr != nil {
			s.logger.Warn("failed to upsert tailscale device",
				zap.String("hostname", shortHostname),
				zap.Error(upsertErr),
			)
			continue
		}

		if matched != nil {
			seenIDs[matched.ID] = true
		} else if device.ID != "" {
			seenIDs[device.ID] = true
		}

		switch {
		case created:
			res.Created++
		case matched != nil:
			res.Updated++
		default:
			res.Unchanged++
		}
	}

	// Mark previously-seen Tailscale devices as offline if not found this sync.
	for i := range existing {
		d := &existing[i]
		if d.DiscoveryMethod != models.DiscoveryTailscale {
			continue
		}
		if seenIDs[d.ID] {
			continue
		}
		d.Status = models.DeviceStatusOffline
		if _, err := s.store.UpsertDevice(ctx, d); err != nil {
			s.logger.Warn("failed to mark tailscale device offline",
				zap.String("id", d.ID),
				zap.Error(err),
			)
		}
	}

	return res, nil
}

// buildDevice creates or updates a Device from Tailscale data.
func buildDevice(tsDev *TailscaleDevice, shortHostname string, existing *models.Device) *models.Device {
	now := time.Now().UTC()

	customFields := map[string]string{
		"tailscale_node_key":  tsDev.NodeKey,
		"tailscale_hostname":  tsDev.Name,
		"tailscale_os":        tsDev.OS,
		"tailscale_tags":      strings.Join(tsDev.Tags, ","),
		"tailscale_device_id": tsDev.ID,
	}

	status := models.DeviceStatusOffline
	if tsDev.Online {
		status = models.DeviceStatusOnline
	}

	if existing != nil {
		// Merge: add Tailscale IPs, update custom fields.
		ipSet := make(map[string]bool, len(existing.IPAddresses)+len(tsDev.Addresses))
		for _, ip := range existing.IPAddresses {
			ipSet[ip] = true
		}
		for _, ip := range tsDev.Addresses {
			ipSet[ip] = true
		}
		mergedIPs := make([]string, 0, len(ipSet))
		for ip := range ipSet {
			mergedIPs = append(mergedIPs, ip)
		}

		// Merge custom fields.
		if existing.CustomFields == nil {
			existing.CustomFields = make(map[string]string)
		}
		for k, v := range customFields {
			existing.CustomFields[k] = v
		}

		existing.IPAddresses = mergedIPs
		existing.Status = status
		existing.LastSeen = now
		if existing.OS == "" && tsDev.OS != "" {
			existing.OS = tsDev.OS
		}
		return existing
	}

	// New device.
	return &models.Device{
		Hostname:        shortHostname,
		IPAddresses:     tsDev.Addresses,
		DeviceType:      models.DeviceTypeUnknown,
		Status:          status,
		OS:              tsDev.OS,
		DiscoveryMethod: models.DiscoveryTailscale,
		LastSeen:        now,
		FirstSeen:       now,
		CustomFields:    customFields,
	}
}

// extractShortHostname strips the MagicDNS suffix from a Tailscale name.
// "myhost.tail12345.ts.net" -> "myhost"
func extractShortHostname(name string) string {
	if name == "" {
		return ""
	}
	parts := strings.SplitN(name, ".", 2)
	return parts[0]
}
