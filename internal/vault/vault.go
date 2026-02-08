package vault

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin            = (*Module)(nil)
	_ plugin.HTTPProvider      = (*Module)(nil)
	_ plugin.HealthChecker     = (*Module)(nil)
	_ roles.CredentialProvider = (*Module)(nil)
)

// PassphraseEnvVar is the environment variable for the vault passphrase.
const PassphraseEnvVar = "SUBNETREE_VAULT_PASSPHRASE"

// Module implements the Vault credential management plugin.
type Module struct {
	logger  *zap.Logger
	cfg     VaultConfig
	store   *VaultStore
	km      *KeyManager
	bus     plugin.EventBus
	plugins plugin.PluginResolver

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// readPassphrase is an injectable function for reading the passphrase
	// from the terminal. Replaced in tests to avoid blocking on stdin.
	readPassphrase func() (string, error)
}

// New creates a new Vault plugin instance.
func New() *Module {
	return &Module{
		readPassphrase: readPassphraseFromStdin,
	}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "vault",
		Version:     "0.1.0",
		Description: "Credential storage and management",
		Roles:       []string{roles.RoleCredentialStore},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger

	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal vault config: %w", err)
		}
	}

	if deps.Store != nil {
		if err := deps.Store.Migrate(context.Background(), "vault", migrations()); err != nil {
			return fmt.Errorf("vault migrations: %w", err)
		}
		m.store = NewVaultStore(deps.Store.DB())
	}

	m.bus = deps.Bus
	m.plugins = deps.Plugins

	// Create key manager and load master key metadata if it exists.
	m.km = NewKeyManager()
	if m.store != nil {
		rec, err := m.store.GetMasterKeyRecord(context.Background())
		if err != nil {
			return fmt.Errorf("load master key record: %w", err)
		}
		if rec != nil {
			m.km.Initialize(rec.Salt, rec.VerificationBlob)
		}
	}

	m.logger.Info("vault module initialized")
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Attempt to unseal the vault.
	if m.store != nil {
		m.tryUnseal()
	}

	// Start audit log maintenance.
	if m.store != nil {
		m.startMaintenance()
	}

	m.logger.Info("vault module started",
		zap.Bool("sealed", m.km.IsSealed()),
		zap.Bool("initialized", m.km.IsInitialized()),
	)
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	if m.km != nil {
		m.km.Seal()
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()

	m.logger.Info("vault module stopped")
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

	if m.km != nil && !m.km.IsSealed() {
		details["vault"] = "unsealed"
	} else {
		details["vault"] = "sealed"
	}

	status := "healthy"
	if m.store == nil {
		status = "degraded"
	} else if m.km == nil || m.km.IsSealed() {
		status = "degraded"
		details["message"] = "vault is sealed; credential operations unavailable"
	}

	return plugin.HealthStatus{
		Status:  status,
		Details: details,
	}
}

// Credential implements roles.CredentialProvider.
func (m *Module) Credential(ctx context.Context, id string) (*roles.Credential, error) {
	if m.store == nil {
		return nil, fmt.Errorf("vault store not available")
	}
	rec, err := m.store.GetCredential(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}
	if rec == nil {
		return nil, nil
	}
	return &roles.Credential{
		ID:       rec.ID,
		Name:     rec.Name,
		Type:     rec.Type,
		DeviceID: rec.DeviceID,
	}, nil
}

// CredentialsForDevice implements roles.CredentialProvider.
func (m *Module) CredentialsForDevice(ctx context.Context, deviceID string) ([]roles.Credential, error) {
	if m.store == nil {
		return nil, fmt.Errorf("vault store not available")
	}
	metas, err := m.store.ListCredentialsByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("list credentials for device: %w", err)
	}
	result := make([]roles.Credential, len(metas))
	for i := range metas {
		result[i] = roles.Credential{
			ID:       metas[i].ID,
			Name:     metas[i].Name,
			Type:     metas[i].Type,
			DeviceID: metas[i].DeviceID,
		}
	}
	return result, nil
}

// tryUnseal attempts to unseal the vault using env var or interactive prompt.
func (m *Module) tryUnseal() {
	passphrase := os.Getenv(PassphraseEnvVar)

	if passphrase == "" {
		// Try interactive prompt.
		p, err := m.readPassphrase()
		if err != nil {
			m.logger.Info("no vault passphrase available; vault remains sealed",
				zap.String("hint", "set "+PassphraseEnvVar+" or use POST /unseal"))
			return
		}
		passphrase = p
	}

	if passphrase == "" {
		m.logger.Info("empty vault passphrase; vault remains sealed")
		return
	}

	if !m.km.IsInitialized() {
		// First run: create master key.
		salt, verification, err := m.km.FirstRunSetup(passphrase)
		if err != nil {
			m.logger.Error("failed to initialize vault master key", zap.Error(err))
			return
		}
		if err := m.store.UpsertMasterKeyRecord(m.ctx, salt, verification); err != nil {
			m.logger.Error("failed to persist master key record", zap.Error(err))
			m.km.Seal()
			return
		}
		m.logger.Info("vault master key initialized and unsealed (first run)")
		return
	}

	// Subsequent run: verify and unseal.
	if err := m.km.Unseal(passphrase); err != nil {
		m.logger.Error("failed to unseal vault", zap.Error(err))
		return
	}
	m.logger.Info("vault unsealed successfully")
}

// startMaintenance runs periodic audit log cleanup.
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
	if m.store == nil {
		return
	}
	cutoff := time.Now().Add(-m.cfg.AuditRetentionPeriod).UTC()
	deleted, err := m.store.DeleteOldAuditEntries(m.ctx, cutoff)
	if err != nil {
		m.logger.Warn("audit log maintenance failed", zap.Error(err))
		return
	}
	if deleted > 0 {
		m.logger.Info("audit log maintenance complete",
			zap.Int64("deleted", deleted),
		)
	}
}

// readPassphraseFromStdin prompts for and reads a passphrase from stdin.
// Returns an error if stdin is not available or reading fails.
func readPassphraseFromStdin() (string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("stat stdin: %w", err)
	}
	// If stdin is a pipe or redirected file, don't prompt.
	if info.Mode()&os.ModeCharDevice == 0 {
		return "", fmt.Errorf("stdin is not a terminal")
	}

	fmt.Fprint(os.Stderr, "Enter vault passphrase: ") //nolint:errcheck
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read passphrase: %w", err)
	}
	return strings.TrimSpace(line), nil
}
