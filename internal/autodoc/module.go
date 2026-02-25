package autodoc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/internal/pulse"
	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DeviceReader provides read access to device data for documentation generation.
type DeviceReader interface {
	GetDevice(ctx context.Context, id string) (*models.Device, error)
	ListAllDevices(ctx context.Context) ([]models.Device, error)
	GetDeviceHardware(ctx context.Context, deviceID string) (*models.DeviceHardware, error)
	GetDeviceStorage(ctx context.Context, deviceID string) ([]models.DeviceStorage, error)
	GetDeviceGPU(ctx context.Context, deviceID string) ([]models.DeviceGPU, error)
	GetDeviceServices(ctx context.Context, deviceID string) ([]models.DeviceService, error)
	GetChildDevices(ctx context.Context, parentID string) ([]models.Device, error)
}

// AlertReader provides read access to alert data for documentation generation.
type AlertReader interface {
	ListDeviceAlerts(ctx context.Context, deviceID string, limit int) ([]DeviceAlert, error)
}

// DeviceAlert is a local representation of an alert to avoid importing internal/pulse.
type DeviceAlert struct {
	Severity    string     `json:"severity"`
	Message     string     `json:"message"`
	TriggeredAt time.Time  `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// Compile-time interface guards.
var (
	_ plugin.Plugin          = (*Module)(nil)
	_ plugin.HTTPProvider    = (*Module)(nil)
	_ plugin.EventSubscriber = (*Module)(nil)
)

// Module implements the AutoDoc auto-documentation plugin.
// It subscribes to system events and automatically generates changelog entries.
type Module struct {
	logger       *zap.Logger
	store        *Store
	bus          plugin.EventBus
	cancel       context.CancelFunc
	deviceReader DeviceReader
	alertReader  AlertReader
}

// SetDeviceReader sets the device data reader for documentation generation.
func (m *Module) SetDeviceReader(r DeviceReader) { m.deviceReader = r }

// SetAlertReader sets the alert data reader for documentation generation.
func (m *Module) SetAlertReader(r AlertReader) { m.alertReader = r }

// New creates a new AutoDoc plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "autodoc",
		Version:     "0.1.0",
		Description: "Auto-documentation engine for infrastructure changes",
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(ctx context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.bus = deps.Bus

	if deps.Store != nil {
		if err := deps.Store.Migrate(ctx, "autodoc", migrations()); err != nil {
			return fmt.Errorf("autodoc migrations: %w", err)
		}
		m.store = NewStore(deps.Store.DB())
	}

	m.logger.Info("autodoc module initialized")
	return nil
}

func (m *Module) Start(_ context.Context) error {
	_, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.logger.Info("autodoc module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.logger.Info("autodoc module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/changes", Handler: m.handleListChanges},
		{Method: "GET", Path: "/export", Handler: m.handleExport},
		{Method: "GET", Path: "/stats", Handler: m.handleStats},
		{Method: "GET", Path: "/devices/{id}", Handler: m.handleDeviceDoc},
		{Method: "GET", Path: "/devices", Handler: m.handleBulkExport},
	}
}

// Subscriptions implements plugin.EventSubscriber.
func (m *Module) Subscriptions() []plugin.Subscription {
	return []plugin.Subscription{
		{Topic: TopicDeviceDiscovered, Handler: m.handleDeviceDiscovered},
		{Topic: TopicDeviceUpdated, Handler: m.handleDeviceUpdated},
		{Topic: TopicDeviceLost, Handler: m.handleDeviceLost},
		{Topic: TopicScanCompleted, Handler: m.handleScanCompleted},
		{Topic: TopicAlertTriggered, Handler: m.handleAlertTriggered},
		{Topic: TopicAlertResolved, Handler: m.handleAlertResolved},
	}
}

// handleDeviceDiscovered creates a changelog entry when a new device is discovered.
func (m *Module) handleDeviceDiscovered(_ context.Context, event plugin.Event) {
	de, ok := event.Payload.(*recon.DeviceEvent)
	if !ok {
		m.logger.Warn("unexpected payload type for device discovered event")
		return
	}

	if de.Device == nil {
		return
	}

	hostname := de.Device.Hostname
	ip := ""
	if len(de.Device.IPAddresses) > 0 {
		ip = de.Device.IPAddresses[0]
	}

	summary := fmt.Sprintf("New device discovered: %s", deviceLabel(hostname, ip))

	m.saveEntry(event, summary, de.Device.ID, de.Device)
}

// handleDeviceUpdated creates a changelog entry when a device is updated.
func (m *Module) handleDeviceUpdated(_ context.Context, event plugin.Event) {
	de, ok := event.Payload.(*recon.DeviceEvent)
	if !ok {
		m.logger.Warn("unexpected payload type for device updated event")
		return
	}

	if de.Device == nil {
		return
	}

	hostname := de.Device.Hostname
	ip := ""
	if len(de.Device.IPAddresses) > 0 {
		ip = de.Device.IPAddresses[0]
	}

	summary := fmt.Sprintf("Device updated: %s", deviceLabel(hostname, ip))

	m.saveEntry(event, summary, de.Device.ID, de.Device)
}

// handleDeviceLost creates a changelog entry when a device goes offline.
func (m *Module) handleDeviceLost(_ context.Context, event plugin.Event) {
	de, ok := event.Payload.(recon.DeviceLostEvent)
	if !ok {
		// Also try pointer form.
		dep, okp := event.Payload.(*recon.DeviceLostEvent)
		if !okp {
			m.logger.Warn("unexpected payload type for device lost event")
			return
		}
		de = *dep
	}

	summary := fmt.Sprintf("Device lost: %s (last seen %s)",
		deviceLabel("", de.IP),
		de.LastSeen.Format(time.RFC3339),
	)

	m.saveEntry(event, summary, de.DeviceID, de)
}

// handleScanCompleted creates a changelog entry when a network scan finishes.
func (m *Module) handleScanCompleted(_ context.Context, event plugin.Event) {
	scan, ok := event.Payload.(*models.ScanResult)
	if !ok {
		m.logger.Warn("unexpected payload type for scan completed event")
		return
	}

	summary := fmt.Sprintf("Network scan completed on %s: %d devices found (%d online)",
		scan.Subnet, scan.Total, scan.Online)

	m.saveEntry(event, summary, "", scan)
}

// handleAlertTriggered creates a changelog entry when an alert fires.
func (m *Module) handleAlertTriggered(_ context.Context, event plugin.Event) {
	alert, ok := event.Payload.(*pulse.Alert)
	if !ok {
		m.logger.Warn("unexpected payload type for alert triggered event")
		return
	}

	summary := fmt.Sprintf("Alert triggered: %s (severity: %s)", alert.Message, alert.Severity)

	m.saveEntry(event, summary, alert.DeviceID, alert)
}

// handleAlertResolved creates a changelog entry when an alert is resolved.
func (m *Module) handleAlertResolved(_ context.Context, event plugin.Event) {
	alert, ok := event.Payload.(*pulse.Alert)
	if !ok {
		m.logger.Warn("unexpected payload type for alert resolved event")
		return
	}

	summary := fmt.Sprintf("Alert resolved: %s", alert.Message)

	m.saveEntry(event, summary, alert.DeviceID, alert)
}

// saveEntry creates and persists a changelog entry.
func (m *Module) saveEntry(event plugin.Event, summary, deviceID string, payload any) {
	if m.store == nil {
		return
	}

	details, err := json.Marshal(payload)
	if err != nil {
		m.logger.Warn("failed to marshal event payload",
			zap.String("event_topic", event.Topic),
			zap.Error(err),
		)
		details = []byte("{}")
	}

	var devID *string
	if deviceID != "" {
		devID = &deviceID
	}

	entry := ChangelogEntry{
		ID:           uuid.New().String(),
		EventType:    event.Topic,
		Summary:      summary,
		Details:      details,
		SourceModule: event.Source,
		DeviceID:     devID,
		CreatedAt:    event.Timestamp,
	}

	if err := m.store.SaveEntry(context.Background(), entry); err != nil {
		m.logger.Warn("failed to save changelog entry",
			zap.String("event_topic", event.Topic),
			zap.Error(err),
		)
	}

	m.logger.Debug("changelog entry created",
		zap.String("event_type", event.Topic),
		zap.String("summary", summary),
	)
}

// deviceLabel returns a human-readable label for a device.
func deviceLabel(hostname, ip string) string {
	if hostname != "" && ip != "" {
		return fmt.Sprintf("%s (%s)", hostname, ip)
	}
	if hostname != "" {
		return hostname
	}
	if ip != "" {
		return ip
	}
	return "unknown"
}
