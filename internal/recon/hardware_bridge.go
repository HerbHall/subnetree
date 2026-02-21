package recon

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProfileSource provides raw hardware profiles from the agent system.
// Defined here (consumer-side interface) to avoid coupling recon -> dispatch.
type ProfileSource interface {
	GetAgent(ctx context.Context, agentID string) (*AgentInfo, error)
	GetHardwareProfile(ctx context.Context, agentID string) (*HardwareProfileData, error)
	GetServices(ctx context.Context, agentID string) ([]*ServiceData, error)
}

// AgentInfo is the minimal agent data needed for profile bridging.
type AgentInfo struct {
	ID       string
	DeviceID string
	Platform string
	Hostname string
}

// HardwareProfileData holds the raw hardware profile from Scout agents.
type HardwareProfileData struct {
	CPUModel           string
	CPUCores           int32
	CPUThreads         int32
	RAMBytes           int64
	BIOSVersion        string
	SystemManufacturer string
	SystemModel        string
	SerialNumber       string
	Disks              []DiskData
	GPUs               []GPUData
}

// DiskData represents a storage disk from the Scout agent profile.
type DiskData struct {
	Name      string
	SizeBytes int64
	DiskType  string
	Model     string
}

// GPUData represents a GPU from the Scout agent profile.
type GPUData struct {
	Model         string
	VRAMBytes     int64
	DriverVersion string
}

// ServiceData represents a running service from the Scout agent profile.
type ServiceData struct {
	Name        string
	DisplayName string
	Status      string
	StartType   string
	CPUPercent  float64
	MemoryBytes int64
	Ports       []int32
}

// handleDeviceProfiled bridges dispatch's TopicDeviceProfiled event to recon's
// normalized hardware tables. It maps raw proto-style profile data into the
// models.DeviceHardware, DeviceStorage, DeviceGPU, and DeviceService records.
func (m *Module) handleDeviceProfiled(ctx context.Context, evt plugin.Event) {
	// Extract agent_id from event payload (map[string]string).
	payload, ok := evt.Payload.(map[string]string)
	if !ok {
		m.logger.Warn("unexpected payload type for device profiled event")
		return
	}
	agentID := payload["agent_id"]
	if agentID == "" {
		m.logger.Warn("device profiled event missing agent_id")
		return
	}

	if m.profileSource == nil {
		m.logger.Debug("profile source not configured, skipping hardware bridge")
		return
	}

	// Look up agent to get device_id.
	agent, err := m.profileSource.GetAgent(ctx, agentID)
	if err != nil {
		m.logger.Error("failed to get agent for hardware bridge",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
		return
	}
	if agent == nil {
		m.logger.Warn("agent not found for hardware bridge", zap.String("agent_id", agentID))
		return
	}
	if agent.DeviceID == "" {
		m.logger.Warn("agent has no device_id, skipping hardware bridge",
			zap.String("agent_id", agentID),
		)
		return
	}

	// Determine collection source based on agent platform.
	source := collectionSourceForPlatform(agent.Platform)

	// Fetch raw hardware profile.
	rawHW, err := m.profileSource.GetHardwareProfile(ctx, agentID)
	if err != nil {
		m.logger.Error("failed to get hardware profile for bridge",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
		return
	}

	now := time.Now().UTC()

	// Map raw hardware to models.DeviceHardware.
	if rawHW != nil {
		hw := &models.DeviceHardware{
			DeviceID:           agent.DeviceID,
			Hostname:           agent.Hostname,
			CPUModel:           rawHW.CPUModel,
			CPUCores:           int(rawHW.CPUCores),
			CPUThreads:         int(rawHW.CPUThreads),
			RAMTotalMB:         int(rawHW.RAMBytes / (1024 * 1024)),
			BIOSVersion:        rawHW.BIOSVersion,
			SystemManufacturer: rawHW.SystemManufacturer,
			SystemModel:        rawHW.SystemModel,
			SerialNumber:       rawHW.SerialNumber,
			CollectionSource:   source,
			CollectedAt:        &now,
		}
		if err := m.store.UpsertDeviceHardware(ctx, hw); err != nil {
			m.logger.Error("failed to upsert device hardware from bridge",
				zap.String("device_id", agent.DeviceID),
				zap.Error(err),
			)
		}

		// Map disks to models.DeviceStorage.
		if len(rawHW.Disks) > 0 {
			storage := make([]models.DeviceStorage, len(rawHW.Disks))
			for i, d := range rawHW.Disks {
				storage[i] = models.DeviceStorage{
					ID:               uuid.New().String(),
					DeviceID:         agent.DeviceID,
					Name:             d.Name,
					DiskType:         d.DiskType,
					CapacityGB:       int(d.SizeBytes / (1024 * 1024 * 1024)),
					Model:            d.Model,
					CollectionSource: source,
					CollectedAt:      &now,
				}
			}
			if err := m.store.UpsertDeviceStorage(ctx, agent.DeviceID, storage); err != nil {
				m.logger.Error("failed to upsert device storage from bridge",
					zap.String("device_id", agent.DeviceID),
					zap.Error(err),
				)
			}
		}

		// Map GPUs to models.DeviceGPU.
		if len(rawHW.GPUs) > 0 {
			gpus := make([]models.DeviceGPU, len(rawHW.GPUs))
			for i, g := range rawHW.GPUs {
				gpus[i] = models.DeviceGPU{
					ID:               uuid.New().String(),
					DeviceID:         agent.DeviceID,
					Model:            g.Model,
					Vendor:           detectGPUVendor(g.Model),
					VRAMMB:           int(g.VRAMBytes / (1024 * 1024)),
					DriverVersion:    g.DriverVersion,
					CollectionSource: source,
					CollectedAt:      &now,
				}
			}
			if err := m.store.UpsertDeviceGPU(ctx, agent.DeviceID, gpus); err != nil {
				m.logger.Error("failed to upsert device GPUs from bridge",
					zap.String("device_id", agent.DeviceID),
					zap.Error(err),
				)
			}
		}
	}

	// Fetch and map services.
	rawSvcs, err := m.profileSource.GetServices(ctx, agentID)
	if err != nil {
		m.logger.Error("failed to get services for bridge",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
	} else if len(rawSvcs) > 0 {
		svcs := make([]models.DeviceService, len(rawSvcs))
		for i, s := range rawSvcs {
			// Use the first port if available.
			port := 0
			if len(s.Ports) > 0 {
				port = int(s.Ports[0])
			}

			svcName := s.Name
			if s.DisplayName != "" {
				svcName = s.DisplayName
			}

			svcs[i] = models.DeviceService{
				ID:               uuid.New().String(),
				DeviceID:         agent.DeviceID,
				Name:             svcName,
				ServiceType:      serviceTypeFromStatus(s.StartType),
				Port:             port,
				Status:           s.Status,
				CollectionSource: source,
				CollectedAt:      &now,
			}
		}
		if err := m.store.UpsertDeviceServices(ctx, agent.DeviceID, svcs); err != nil {
			m.logger.Error("failed to upsert device services from bridge",
				zap.String("device_id", agent.DeviceID),
				zap.Error(err),
			)
		}
	}

	// Publish recon.device.hardware.updated event.
	m.publishEvent(ctx, TopicDeviceHardwareUpdated, DeviceHardwareUpdatedEvent{
		DeviceID:         agent.DeviceID,
		CollectionSource: source,
	})

	m.logger.Info("hardware profile bridged",
		zap.String("agent_id", agentID),
		zap.String("device_id", agent.DeviceID),
		zap.String("source", source),
	)
}

// collectionSourceForPlatform returns the collection source identifier
// based on the Scout agent's reported platform.
func collectionSourceForPlatform(platform string) string {
	p := strings.ToLower(platform)
	switch {
	case strings.Contains(p, "windows"):
		return "scout-wmi"
	case strings.Contains(p, "linux"):
		return "scout-linux"
	case strings.Contains(p, "darwin") || strings.Contains(p, "macos"):
		return "scout-macos"
	default:
		return fmt.Sprintf("scout-%s", p)
	}
}

// detectGPUVendor infers the GPU vendor from the model string.
func detectGPUVendor(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "nvidia") ||
		strings.Contains(lower, "geforce") ||
		strings.Contains(lower, "rtx") ||
		strings.Contains(lower, "gtx"):
		return "nvidia"
	case strings.Contains(lower, "amd") ||
		strings.Contains(lower, "radeon"):
		return "amd"
	case strings.Contains(lower, "intel") ||
		strings.Contains(lower, "arc") ||
		strings.Contains(lower, "iris"):
		return "intel"
	default:
		return "unknown"
	}
}

// serviceTypeFromStatus maps the agent start_type to a service type label.
func serviceTypeFromStatus(startType string) string {
	switch strings.ToLower(startType) {
	case "auto":
		return "system"
	case "manual":
		return "manual"
	case "disabled":
		return "disabled"
	default:
		return "other"
	}
}
