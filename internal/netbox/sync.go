package netbox

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// syncEngine orchestrates the sync between SubNetree devices and NetBox.
type syncEngine struct {
	client       *Client
	deviceReader DeviceReader
	cfg          Config
	logger       *zap.Logger
}

// newSyncEngine creates a new sync engine.
func newSyncEngine(client *Client, reader DeviceReader, cfg Config, logger *zap.Logger) *syncEngine {
	return &syncEngine{
		client:       client,
		deviceReader: reader,
		cfg:          cfg,
		logger:       logger,
	}
}

// SyncAll performs a full sync of all SubNetree devices to NetBox.
func (s *syncEngine) SyncAll(ctx context.Context, dryRun bool) (result *SyncResult, err error) {
	result = &SyncResult{DryRun: dryRun}

	// Ensure the SubNetree management tag exists.
	tagID, err := s.client.EnsureTag(ctx, s.cfg.TagName)
	if err != nil {
		return nil, fmt.Errorf("ensure tag: %w", err)
	}
	s.logger.Debug("tag ensured", zap.Int("tag_id", tagID), zap.String("tag_name", s.cfg.TagName))

	// Resolve the site ID.
	siteID, err := s.resolveSite(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve site: %w", err)
	}
	s.logger.Debug("site resolved", zap.Int("site_id", siteID))

	// Load all SubNetree devices.
	devices, err := s.deviceReader.ListAllDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	s.logger.Info("devices loaded for sync", zap.Int("count", len(devices)))

	// Load existing NetBox devices with our tag to detect updates vs creates.
	existing, err := s.client.ListDevicesByTag(ctx, s.cfg.TagName)
	if err != nil {
		return nil, fmt.Errorf("list netbox devices: %w", err)
	}
	existingByName := make(map[string]*NBDevice, len(existing))
	for i := range existing {
		existingByName[existing[i].Name] = &existing[i]
	}

	// Sync each device.
	for i := range devices {
		syncErr := s.syncOneDevice(ctx, &devices[i], siteID, tagID, existingByName, dryRun, result)
		if syncErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, syncErr.Error())
			s.logger.Warn("device sync failed",
				zap.String("device_id", devices[i].ID),
				zap.Error(syncErr),
			)
		}
	}

	return result, nil
}

// SyncDevice syncs a single SubNetree device to NetBox.
func (s *syncEngine) SyncDevice(ctx context.Context, deviceID string, dryRun bool) (result *SyncResult, err error) {
	result = &SyncResult{DryRun: dryRun}

	device, err := s.deviceReader.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", deviceID, err)
	}
	if device == nil {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	tagID, err := s.client.EnsureTag(ctx, s.cfg.TagName)
	if err != nil {
		return nil, fmt.Errorf("ensure tag: %w", err)
	}

	siteID, err := s.resolveSite(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve site: %w", err)
	}

	existing, err := s.client.ListDevicesByTag(ctx, s.cfg.TagName)
	if err != nil {
		return nil, fmt.Errorf("list netbox devices: %w", err)
	}
	existingByName := make(map[string]*NBDevice, len(existing))
	for i := range existing {
		existingByName[existing[i].Name] = &existing[i]
	}

	syncErr := s.syncOneDevice(ctx, device, siteID, tagID, existingByName, dryRun, result)
	if syncErr != nil {
		result.Failed++
		result.Errors = append(result.Errors, syncErr.Error())
	}

	return result, nil
}

// syncOneDevice handles the create-or-update logic for a single device.
func (s *syncEngine) syncOneDevice(
	ctx context.Context,
	device *models.Device,
	siteID, tagID int,
	existingByName map[string]*NBDevice,
	dryRun bool,
	result *SyncResult,
) error {
	// Resolve manufacturer.
	manufacturer := device.Manufacturer
	if manufacturer == "" {
		manufacturer = "Unknown"
	}
	manufacturerID, err := s.client.GetOrCreateManufacturer(ctx, manufacturer)
	if err != nil {
		return fmt.Errorf("manufacturer %q: %w", manufacturer, err)
	}

	// Resolve device type (hardware model).
	model := string(device.DeviceType)
	if model == "" {
		model = "unknown"
	}
	typeID, err := s.client.GetOrCreateDeviceType(ctx, manufacturerID, model)
	if err != nil {
		return fmt.Errorf("device type %q: %w", model, err)
	}

	// Resolve device role.
	roleName := MapDeviceRole(device.DeviceType)
	roleID, err := s.client.GetOrCreateDeviceRole(ctx, roleName)
	if err != nil {
		return fmt.Errorf("device role %q: %w", roleName, err)
	}

	req := DeviceToNetBoxRequest(device, roleID, typeID, siteID, tagID)

	// Check if device already exists in NetBox.
	if nbDev, exists := existingByName[req.Name]; exists {
		if dryRun {
			s.logger.Info("dry-run: would update device", zap.String("name", req.Name), zap.Int("netbox_id", nbDev.ID))
			result.Updated++
			return nil
		}
		_, err := s.client.UpdateDevice(ctx, nbDev.ID, req)
		if err != nil {
			return fmt.Errorf("update device %q: %w", req.Name, err)
		}
		s.logger.Info("device updated", zap.String("name", req.Name), zap.Int("netbox_id", nbDev.ID))
		result.Updated++
		return nil
	}

	if dryRun {
		s.logger.Info("dry-run: would create device", zap.String("name", req.Name))
		result.Created++
		return nil
	}

	// Create the device.
	nbDev, err := s.client.CreateDevice(ctx, req)
	if err != nil {
		return fmt.Errorf("create device %q: %w", req.Name, err)
	}
	s.logger.Info("device created", zap.String("name", req.Name), zap.Int("netbox_id", nbDev.ID))

	// Create interface + IP assignments.
	if device.MACAddress != "" || len(device.IPAddresses) > 0 {
		ifaceName := "eth0"
		iface, ifErr := s.client.CreateInterface(ctx, nbDev.ID, ifaceName, device.MACAddress)
		if ifErr != nil {
			s.logger.Warn("failed to create interface", zap.Error(ifErr))
		} else {
			for _, ip := range device.IPAddresses {
				// NetBox requires CIDR notation; default to /32 for single hosts.
				addr := ip
				if !containsSlash(addr) {
					addr += "/32"
				}
				if _, ipErr := s.client.CreateIPAddress(ctx, addr, iface.ID); ipErr != nil {
					s.logger.Warn("failed to create IP address", zap.String("ip", ip), zap.Error(ipErr))
				}
			}
		}
	}

	result.Created++
	return nil
}

// resolveSite determines the NetBox site ID, creating one if needed.
func (s *syncEngine) resolveSite(ctx context.Context) (int, error) {
	if s.cfg.SiteID > 0 {
		return s.cfg.SiteID, nil
	}
	siteName := s.cfg.SiteName
	if siteName == "" {
		siteName = "SubNetree Default Site"
	}
	return s.client.GetOrCreateSite(ctx, siteName)
}

// containsSlash checks if a string contains a forward slash.
func containsSlash(s string) bool {
	for _, c := range s {
		if c == '/' {
			return true
		}
	}
	return false
}
