package tailscale

import (
	"context"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin        = (*Module)(nil)
	_ plugin.HTTPProvider  = (*Module)(nil)
	_ plugin.HealthChecker = (*Module)(nil)
)

// Module implements the Tailscale integration plugin. It periodically syncs
// devices from a Tailscale tailnet and merges them with existing device records.
type Module struct {
	logger    *zap.Logger
	cfg       TailscaleConfig
	store     DeviceStore
	decrypter CredentialDecrypter
	bus       plugin.EventBus

	syncer *Syncer
	client *TailscaleClient
	stopCh chan struct{}
	wg     sync.WaitGroup

	mu             sync.RWMutex
	lastSyncTime   time.Time
	lastSyncResult *SyncResult
	lastSyncErr    error
}

// New creates a new Tailscale plugin instance.
func New() *Module {
	return &Module{}
}

// Info implements plugin.Plugin.
func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:         "tailscale",
		Version:      "0.1.0",
		Description:  "Discovers and syncs devices from a Tailscale tailnet",
		Dependencies: []string{"recon", "vault"},
		Roles:        []string{roles.RoleOverlayNetwork},
		APIVersion:   plugin.APIVersionCurrent,
	}
}

// Init implements plugin.Plugin.
func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.bus = deps.Bus
	m.cfg = DefaultConfig()

	if deps.Config != nil {
		if deps.Config.IsSet("enabled") {
			m.cfg.Enabled = deps.Config.GetBool("enabled")
		}
		if v := deps.Config.GetString("credential_id"); v != "" {
			m.cfg.CredentialID = v
		}
		if v := deps.Config.GetString("tailnet"); v != "" {
			m.cfg.Tailnet = v
		}
		if d := deps.Config.GetDuration("sync_interval"); d > 0 {
			m.cfg.SyncInterval = d
		}
		if v := deps.Config.GetString("base_url"); v != "" {
			m.cfg.BaseURL = v
		}
	}

	m.logger.Info("tailscale module initialized",
		zap.Bool("enabled", m.cfg.Enabled),
		zap.String("tailnet", m.cfg.Tailnet),
		zap.Duration("sync_interval", m.cfg.SyncInterval),
	)
	return nil
}

// Start implements plugin.Plugin.
func (m *Module) Start(_ context.Context) error {
	if !m.cfg.Enabled {
		m.logger.Info("tailscale module started (disabled)")
		return nil
	}

	if m.cfg.CredentialID == "" {
		m.logger.Warn("tailscale module started but no credential_id configured; sync will not run")
		return nil
	}

	if m.store == nil || m.decrypter == nil {
		m.logger.Warn("tailscale module started but adapters not wired; sync will not run")
		return nil
	}

	// Decrypt the API key from the vault.
	apiKey, err := m.resolveAPIKey(context.Background())
	if err != nil {
		m.logger.Warn("tailscale: could not resolve API key; sync will not run",
			zap.Error(err),
		)
		return nil
	}

	m.client = NewClient(apiKey, m.cfg.BaseURL, m.cfg.Tailnet)
	m.syncer = NewSyncer(m.store, m.logger.Named("syncer"))
	m.stopCh = make(chan struct{})

	m.wg.Add(1)
	go m.syncLoop()

	m.logger.Info("tailscale sync loop started",
		zap.String("tailnet", m.cfg.Tailnet),
		zap.Duration("interval", m.cfg.SyncInterval),
	)
	return nil
}

// Stop implements plugin.Plugin.
func (m *Module) Stop(_ context.Context) error {
	if m.stopCh != nil {
		close(m.stopCh)
		m.wg.Wait()
		m.logger.Info("tailscale module stopped")
	}
	return nil
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	if !m.cfg.Enabled {
		return plugin.HealthStatus{
			Status:  "healthy",
			Message: "tailscale integration disabled",
		}
	}

	if m.cfg.CredentialID == "" {
		return plugin.HealthStatus{
			Status:  "unhealthy",
			Message: "no credential_id configured",
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastSyncErr != nil {
		return plugin.HealthStatus{
			Status:  "degraded",
			Message: "last sync failed: " + m.lastSyncErr.Error(),
		}
	}

	if m.lastSyncResult != nil {
		return plugin.HealthStatus{
			Status:  "healthy",
			Message: "last sync OK",
			Details: map[string]string{
				"devices_found": itoa(m.lastSyncResult.DevicesFound),
				"created":       itoa(m.lastSyncResult.Created),
				"updated":       itoa(m.lastSyncResult.Updated),
			},
		}
	}

	return plugin.HealthStatus{
		Status:  "healthy",
		Message: "awaiting first sync",
	}
}

// SetDeviceStore wires the consumer-side device store adapter.
func (m *Module) SetDeviceStore(s DeviceStore) {
	m.store = s
}

// SetCredentialDecrypter wires the credential decrypter adapter.
func (m *Module) SetCredentialDecrypter(d CredentialDecrypter) {
	m.decrypter = d
}

// syncLoop runs the periodic sync in the background.
func (m *Module) syncLoop() {
	defer m.wg.Done()

	// Run an initial sync immediately.
	m.runSync()

	ticker := time.NewTicker(m.cfg.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.runSync()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Module) runSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := m.syncer.Sync(ctx, m.client)

	m.mu.Lock()
	m.lastSyncTime = time.Now().UTC()
	m.lastSyncResult = result
	m.lastSyncErr = err
	m.mu.Unlock()

	if err != nil {
		m.logger.Error("tailscale sync failed", zap.Error(err))
		return
	}

	m.logger.Info("tailscale sync completed",
		zap.Int("devices_found", result.DevicesFound),
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated),
		zap.Int("unchanged", result.Unchanged),
	)
}

// resolveAPIKey retrieves the Tailscale API key from the vault credential.
func (m *Module) resolveAPIKey(ctx context.Context) (string, error) {
	data, err := m.decrypter.DecryptCredential(ctx, m.cfg.CredentialID)
	if err != nil {
		return "", err
	}
	key, ok := data["api_key"]
	if !ok {
		// Try common alternative field names.
		key, ok = data["token"]
		if !ok {
			key, ok = data["password"]
		}
	}
	if !ok {
		return "", errNoAPIKey
	}
	s, ok := key.(string)
	if !ok || s == "" {
		return "", errNoAPIKey
	}
	return s, nil
}

var errNoAPIKey = &apiKeyError{}

type apiKeyError struct{}

func (e *apiKeyError) Error() string {
	return "credential does not contain an api_key, token, or password field"
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	if neg {
		buf[i] = '-'
		i--
	}
	return string(buf[i+1:])
}
