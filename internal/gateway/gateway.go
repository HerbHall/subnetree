package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin              = (*Module)(nil)
	_ plugin.HTTPProvider        = (*Module)(nil)
	_ plugin.HealthChecker       = (*Module)(nil)
	_ roles.RemoteAccessProvider = (*Module)(nil)
)

// Module implements the Gateway remote access plugin.
type Module struct {
	logger       *zap.Logger
	cfg          GatewayConfig
	store        *GatewayStore
	bus          plugin.EventBus
	plugins      plugin.PluginResolver
	sessions     *SessionManager
	proxies      *ReverseProxyManager
	deviceLookup DeviceLookup

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Gateway plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "gateway",
		Version:     "0.1.0",
		Description: "Remote access gateway for managed devices",
		Roles:       []string{roles.RoleRemoteAccess},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger

	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal gateway config: %w", err)
		}
	}

	if deps.Store != nil {
		if err := deps.Store.Migrate(context.Background(), "gateway", migrations()); err != nil {
			return fmt.Errorf("gateway migrations: %w", err)
		}
		m.store = NewGatewayStore(deps.Store.DB())
	}

	m.bus = deps.Bus
	m.plugins = deps.Plugins
	m.sessions = NewSessionManager(m.cfg.MaxSessions)

	m.logger.Info("gateway module initialized")
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Initialize reverse proxy manager.
	m.proxies = NewReverseProxyManager(m.logger)

	// Try to resolve device lookup (optional -- proxy can work with explicit targets).
	if m.plugins != nil {
		dl, err := resolveDeviceLookup(m.plugins)
		if err != nil {
			m.logger.Debug("device lookup not available, proxy requires explicit targets",
				zap.Error(err),
			)
		} else {
			m.deviceLookup = dl
		}
	}

	if m.store != nil {
		m.startMaintenance()
	}

	m.logger.Info("gateway module started",
		zap.Int("max_sessions", m.cfg.MaxSessions),
		zap.Duration("session_timeout", m.cfg.SessionTimeout),
	)
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	// Close all active proxies.
	if m.proxies != nil {
		m.proxies.CloseAll()
	}

	// Close all active sessions.
	if m.sessions != nil {
		for _, s := range m.sessions.List() {
			m.sessions.Delete(s.ID)
			m.logSessionClosed(s, "shutdown")
		}
	}

	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()

	m.logger.Info("gateway module stopped")
	return nil
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	details := map[string]string{}

	if m.store != nil {
		details["store"] = "connected"
	} else {
		details["store"] = "unavailable"
	}

	sessionCount := 0
	if m.sessions != nil {
		sessionCount = m.sessions.Count()
	}
	details["active_sessions"] = fmt.Sprintf("%d", sessionCount)

	status := "healthy"
	if m.store == nil {
		status = "degraded"
		details["message"] = "gateway store not available"
	}

	return plugin.HealthStatus{
		Status:  status,
		Details: details,
	}
}

// Available implements roles.RemoteAccessProvider.
// Reports whether the gateway can accept new sessions for the given device.
func (m *Module) Available(_ context.Context, _ string) (bool, error) {
	if m.sessions == nil {
		return false, nil
	}
	return m.sessions.Count() < m.cfg.MaxSessions, nil
}

// startMaintenance runs periodic session cleanup and audit retention.
func (m *Module) startMaintenance() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(m.cfg.MaintenanceInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.runMaintenance()
			}
		}
	}()
}

func (m *Module) runMaintenance() {
	// Close expired sessions and their proxies.
	if m.sessions != nil {
		expired := m.sessions.CloseExpired()
		for _, s := range expired {
			if m.proxies != nil {
				m.proxies.RemoveProxy(s.ID)
			}
			m.logSessionClosed(s, "expired")
		}
		if len(expired) > 0 {
			m.logger.Info("expired sessions cleaned up",
				zap.Int("count", len(expired)),
			)
		}
	}

	// Delete old audit entries.
	if m.store != nil {
		cutoff := time.Now().AddDate(0, 0, -m.cfg.AuditRetentionDays).UTC()
		deleted, err := m.store.DeleteOldAuditEntries(m.ctx, cutoff)
		if err != nil {
			m.logger.Warn("gateway audit log maintenance failed", zap.Error(err))
			return
		}
		if deleted > 0 {
			m.logger.Info("gateway audit log maintenance complete",
				zap.Int64("deleted", deleted),
			)
		}
	}
}

// logSessionClosed writes an audit entry and publishes a session closed event.
func (m *Module) logSessionClosed(s *Session, reason string) {
	if m.store != nil {
		entry := &AuditEntry{
			SessionID:   s.ID,
			DeviceID:    s.DeviceID,
			UserID:      s.UserID,
			SessionType: string(s.SessionType),
			Target:      fmt.Sprintf("%s:%d", s.Target.Host, s.Target.Port),
			Action:      "closed:" + reason,
			BytesIn:     s.BytesInCount(),
			BytesOut:    s.BytesOutCount(),
			SourceIP:    s.SourceIP,
			Timestamp:   time.Now().UTC(),
		}
		if err := m.store.InsertAuditEntry(m.ctx, entry); err != nil {
			m.logger.Warn("failed to write session close audit entry", zap.Error(err))
		}
	}

	m.publishEvent(TopicSessionClosed, map[string]string{
		"session_id": s.ID,
		"device_id":  s.DeviceID,
		"reason":     reason,
	})
}

// publishEvent publishes an event to the bus if available.
func (m *Module) publishEvent(topic string, payload any) {
	if m.bus == nil {
		return
	}
	m.bus.PublishAsync(m.ctx, plugin.Event{
		Topic:     topic,
		Source:    "gateway",
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}
