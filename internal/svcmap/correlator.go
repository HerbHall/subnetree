package svcmap

import (
	"context"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// staleCutoff defines how long since last_seen before a service is marked unknown.
const staleCutoff = 5 * time.Minute

// ServiceSource provides Scout agent service data.
type ServiceSource interface {
	GetServices(ctx context.Context, agentID string) ([]ScoutService, error)
}

// AppSource provides application data (e.g., Docker containers from Docs module).
type AppSource interface {
	ListApplicationsByDevice(ctx context.Context, deviceID string) ([]AppInfo, error)
}

// ScoutService is a lightweight DTO for a service reported by a Scout agent.
type ScoutService struct {
	Name        string
	DisplayName string
	Status      string // "running", "stopped", etc.
	StartType   string // "auto", "manual", "disabled"
	CPUPercent  float64
	MemoryBytes int64
	Ports       []string
}

// AppInfo is a lightweight DTO for an application/container.
type AppInfo struct {
	ID            string
	Name          string
	ContainerName string
	Status        string // "running", "exited", "created"
}

// Correlator merges services from Scout agents and application sources
// into a unified service inventory.
type Correlator struct {
	store  *Store
	logger *zap.Logger
}

// NewCorrelator creates a new Correlator backed by the given store.
func NewCorrelator(store *Store, logger *zap.Logger) *Correlator {
	return &Correlator{store: store, logger: logger}
}

// CorrelateDevice merges Scout agent services and application data for a device.
// It upserts each discovered service and marks stale ones as unknown.
func (c *Correlator) CorrelateDevice(
	ctx context.Context,
	deviceID string,
	agentID string,
	svcSource ServiceSource,
	appSource AppSource,
) error {
	now := time.Now().UTC()

	// Fetch Scout services if agent is linked.
	if agentID != "" && svcSource != nil {
		scoutServices, err := svcSource.GetServices(ctx, agentID)
		if err != nil {
			c.logger.Warn("failed to fetch scout services",
				zap.String("agent_id", agentID),
				zap.Error(err))
		} else {
			for i := range scoutServices {
				if err := c.upsertScoutService(ctx, deviceID, &scoutServices[i], now); err != nil {
					return fmt.Errorf("upsert scout service %q: %w", scoutServices[i].Name, err)
				}
			}
		}
	}

	// Fetch application data.
	if appSource != nil {
		apps, err := appSource.ListApplicationsByDevice(ctx, deviceID)
		if err != nil {
			c.logger.Warn("failed to fetch applications",
				zap.String("device_id", deviceID),
				zap.Error(err))
		} else {
			for i := range apps {
				if err := c.upsertAppService(ctx, deviceID, &apps[i], now); err != nil {
					return fmt.Errorf("upsert app service %q: %w", apps[i].Name, err)
				}
			}
		}
	}

	// Mark stale services.
	cutoff := now.Add(-staleCutoff)
	staleCount, err := c.store.MarkStaleServices(ctx, deviceID, cutoff)
	if err != nil {
		return fmt.Errorf("mark stale services: %w", err)
	}
	if staleCount > 0 {
		c.logger.Debug("marked stale services",
			zap.String("device_id", deviceID),
			zap.Int64("count", staleCount))
	}

	return nil
}

func (c *Correlator) upsertScoutService(ctx context.Context, deviceID string, scout *ScoutService, now time.Time) error {
	existing, err := c.store.FindByDeviceAndName(ctx, deviceID, scout.Name)
	if err != nil {
		return err
	}

	svcType := inferServiceType(scout)
	status := mapScoutStatus(scout.Status)

	if existing != nil {
		existing.DisplayName = scout.DisplayName
		existing.ServiceType = svcType
		existing.Status = status
		existing.CPUPercent = scout.CPUPercent
		existing.MemoryBytes = scout.MemoryBytes
		existing.Ports = scout.Ports
		existing.LastSeen = now
		return c.store.UpsertService(ctx, existing)
	}

	svc := &models.Service{
		ID:           fmt.Sprintf("svc-%s-%s", deviceID[:8], scout.Name),
		Name:         scout.Name,
		DisplayName:  scout.DisplayName,
		ServiceType:  svcType,
		DeviceID:     deviceID,
		Status:       status,
		DesiredState: models.DesiredStateMonitoringOnly,
		Ports:        scout.Ports,
		CPUPercent:   scout.CPUPercent,
		MemoryBytes:  scout.MemoryBytes,
		FirstSeen:    now,
		LastSeen:     now,
	}
	return c.store.UpsertService(ctx, svc)
}

func (c *Correlator) upsertAppService(ctx context.Context, deviceID string, app *AppInfo, now time.Time) error {
	name := app.ContainerName
	if name == "" {
		name = app.Name
	}

	existing, err := c.store.FindByDeviceAndName(ctx, deviceID, name)
	if err != nil {
		return err
	}

	status := mapAppStatus(app.Status)

	if existing != nil {
		existing.ApplicationID = app.ID
		existing.Status = status
		existing.LastSeen = now
		return c.store.UpsertService(ctx, existing)
	}

	svc := &models.Service{
		ID:            fmt.Sprintf("svc-%s-%s", deviceID[:8], name),
		Name:          name,
		DisplayName:   app.Name,
		ServiceType:   models.ServiceTypeDockerContainer,
		DeviceID:      deviceID,
		ApplicationID: app.ID,
		Status:        status,
		DesiredState:  models.DesiredStateMonitoringOnly,
		FirstSeen:     now,
		LastSeen:      now,
	}
	return c.store.UpsertService(ctx, svc)
}

func inferServiceType(scout *ScoutService) models.ServiceType {
	switch scout.StartType {
	case "auto", "manual", "disabled":
		return models.ServiceTypeWindowsService
	default:
		return models.ServiceTypeApplication
	}
}

func mapScoutStatus(status string) models.ServiceStatus {
	switch status {
	case "running":
		return models.ServiceStatusRunning
	case "stopped":
		return models.ServiceStatusStopped
	case "failed":
		return models.ServiceStatusFailed
	default:
		return models.ServiceStatusUnknown
	}
}

func mapAppStatus(status string) models.ServiceStatus {
	switch status {
	case "running":
		return models.ServiceStatusRunning
	case "exited", "stopped":
		return models.ServiceStatusStopped
	case "created":
		return models.ServiceStatusUnknown
	default:
		return models.ServiceStatusUnknown
	}
}
