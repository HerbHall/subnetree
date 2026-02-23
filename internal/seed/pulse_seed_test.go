package seed

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/pulse"
	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
)

// setupTestDB creates a test database with both recon and pulse tables.
func setupTestDB(t *testing.T) (*store.SQLiteStore, *recon.ReconStore, *pulse.PulseStore) {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()

	reconStore := recon.NewReconStore(db.DB())
	if err := db.Migrate(ctx, "recon", recon.Migrations()); err != nil {
		t.Fatalf("recon migrations: %v", err)
	}

	pulseStore := pulse.NewPulseStore(db.DB())
	if err := db.Migrate(ctx, "pulse", pulse.Migrations()); err != nil {
		t.Fatalf("pulse migrations: %v", err)
	}

	return db, reconStore, pulseStore
}

func TestSeedPulseData_Success(t *testing.T) {
	db, reconStore, pulseStore := setupTestDB(t)
	ctx := context.Background()

	// First seed the recon demo network so devices exist.
	if err := SeedDemoNetwork(ctx, reconStore); err != nil {
		t.Fatalf("SeedDemoNetwork: %v", err)
	}

	// Now seed pulse data.
	if err := SeedPulseData(ctx, pulseStore, db.DB()); err != nil {
		t.Fatalf("SeedPulseData: %v", err)
	}

	// Verify checks were created.
	checks, err := pulseStore.ListAllChecks(ctx)
	if err != nil {
		t.Fatalf("ListAllChecks: %v", err)
	}
	if len(checks) == 0 {
		t.Fatal("expected checks to be created, got 0")
	}
	if len(checks) < 5 {
		t.Errorf("expected at least 5 checks, got %d", len(checks))
	}

	// Verify results were created for at least one device.
	var foundResults bool
	for _, check := range checks {
		results, err := pulseStore.ListResults(ctx, check.DeviceID, 10)
		if err != nil {
			t.Fatalf("ListResults for %s: %v", check.DeviceName, err)
		}
		if len(results) > 0 {
			foundResults = true
			break
		}
	}
	if !foundResults {
		t.Error("expected at least some check results, found none")
	}

	// Verify alerts were created.
	alerts, err := pulseStore.ListAlerts(ctx, pulse.AlertFilters{Limit: 50})
	if err != nil {
		t.Fatalf("ListAlerts: %v", err)
	}
	if len(alerts) == 0 {
		t.Fatal("expected alerts to be created, got 0")
	}

	// Verify we have a mix of resolved and active alerts.
	var active, resolved int
	for _, a := range alerts {
		if a.ResolvedAt != nil {
			resolved++
		} else {
			active++
		}
	}
	if active == 0 {
		t.Error("expected at least one active alert")
	}
	if resolved == 0 {
		t.Error("expected at least one resolved alert")
	}
}

func TestSeedPulseData_Idempotent(t *testing.T) {
	db, reconStore, pulseStore := setupTestDB(t)
	ctx := context.Background()

	if err := SeedDemoNetwork(ctx, reconStore); err != nil {
		t.Fatalf("SeedDemoNetwork: %v", err)
	}

	// Seed pulse data twice.
	if err := SeedPulseData(ctx, pulseStore, db.DB()); err != nil {
		t.Fatalf("first SeedPulseData: %v", err)
	}

	checks1, _ := pulseStore.ListAllChecks(ctx)
	count1 := len(checks1)

	if err := SeedPulseData(ctx, pulseStore, db.DB()); err != nil {
		t.Fatalf("second SeedPulseData: %v", err)
	}

	checks2, _ := pulseStore.ListAllChecks(ctx)
	count2 := len(checks2)

	if count2 != count1 {
		t.Errorf("check count changed after second seed: %d -> %d", count1, count2)
	}
}

func TestSeedPulseData_NoDevicesError(t *testing.T) {
	db, _, pulseStore := setupTestDB(t)
	ctx := context.Background()

	// Seed pulse data without seeding recon first -- should error.
	err := SeedPulseData(ctx, pulseStore, db.DB())
	if err == nil {
		t.Fatal("expected error when no devices exist")
	}
}

func TestSeedPulseData_MetricsForKeyDevices(t *testing.T) {
	db, reconStore, pulseStore := setupTestDB(t)
	ctx := context.Background()

	if err := SeedDemoNetwork(ctx, reconStore); err != nil {
		t.Fatalf("SeedDemoNetwork: %v", err)
	}
	if err := SeedPulseData(ctx, pulseStore, db.DB()); err != nil {
		t.Fatalf("SeedPulseData: %v", err)
	}

	// Query devices from recon to get their IDs.
	devices, _, err := reconStore.ListDevices(ctx, recon.ListDevicesOptions{Limit: 100})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}

	// Check that key infrastructure devices have check results.
	keyDevices := map[string]bool{
		"ubiquiti-gateway": false,
		"proxmox-host":     false,
		"synology-nas":     false,
	}

	for _, d := range devices {
		if _, isKey := keyDevices[d.Hostname]; !isKey {
			continue
		}
		results, err := pulseStore.ListResults(ctx, d.ID, 10)
		if err != nil {
			t.Errorf("ListResults for %s: %v", d.Hostname, err)
			continue
		}
		if len(results) > 0 {
			keyDevices[d.Hostname] = true
		}
	}

	for hostname, hasResults := range keyDevices {
		if !hasResults {
			t.Errorf("device %s has no check results", hostname)
		}
	}
}

func TestSeedPulseData_AlertStatuses(t *testing.T) {
	db, reconStore, pulseStore := setupTestDB(t)
	ctx := context.Background()

	if err := SeedDemoNetwork(ctx, reconStore); err != nil {
		t.Fatalf("SeedDemoNetwork: %v", err)
	}
	if err := SeedPulseData(ctx, pulseStore, db.DB()); err != nil {
		t.Fatalf("SeedPulseData: %v", err)
	}

	alerts, err := pulseStore.ListAlerts(ctx, pulse.AlertFilters{Limit: 50})
	if err != nil {
		t.Fatalf("ListAlerts: %v", err)
	}

	// Verify severity distribution.
	severities := make(map[string]int)
	for _, a := range alerts {
		severities[a.Severity]++
	}

	if severities["warning"] == 0 {
		t.Error("expected at least one warning alert")
	}

	// Verify timing: all alerts should have triggeredAt in the last 24 hours.
	dayAgo := time.Now().Add(-25 * time.Hour)
	for _, a := range alerts {
		if a.TriggeredAt.Before(dayAgo) {
			t.Errorf("alert %s triggered too long ago: %s", a.ID, a.TriggeredAt)
		}
	}
}

// Verify that the seed function uses the same device IDs as the recon seed.
func TestSeedPulseData_DeviceIDsMatchRecon(t *testing.T) {
	db, reconStore, pulseStore := setupTestDB(t)
	ctx := context.Background()

	if err := SeedDemoNetwork(ctx, reconStore); err != nil {
		t.Fatalf("SeedDemoNetwork: %v", err)
	}
	if err := SeedPulseData(ctx, pulseStore, db.DB()); err != nil {
		t.Fatalf("SeedPulseData: %v", err)
	}

	checks, _ := pulseStore.ListAllChecks(ctx)
	devices, _, _ := reconStore.ListDevices(ctx, recon.ListDevicesOptions{Limit: 100})

	deviceIDs := make(map[string]bool)
	for _, d := range devices {
		deviceIDs[d.ID] = true
	}

	for _, c := range checks {
		if !deviceIDs[c.DeviceID] {
			t.Errorf("check %s references unknown device %s", c.ID, c.DeviceID)
		}
	}
}

// Stub: verify that the seed package can reference models.Device.
// This ensures no import cycle exists.
var _ = models.DeviceTypeRouter
