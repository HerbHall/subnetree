package seed

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/HerbHall/subnetree/internal/pulse"
	"github.com/google/uuid"
)

// demoDevice holds a device's identity for seed data correlation.
type demoDevice struct {
	id       string
	hostname string
	ip       string
}

// SeedPulseData populates the Pulse monitoring tables with realistic demo data.
// It creates checks, 24 hours of check results, and sample alerts for key
// devices. The function queries recon_devices to discover seeded device IDs.
//
// It is idempotent: checks are matched by device_id, and existing data is
// skipped rather than duplicated.
func SeedPulseData(ctx context.Context, pulseStore *pulse.PulseStore, db *sql.DB) error {
	devices, err := loadDemoDevices(ctx, db)
	if err != nil {
		return fmt.Errorf("load demo devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no recon devices found; seed recon data first")
	}

	// Create monitoring checks for key devices.
	checks, err := seedPulseChecks(ctx, pulseStore, devices)
	if err != nil {
		return fmt.Errorf("seed checks: %w", err)
	}

	// Generate 24 hours of check results.
	if err := seedPulseResults(ctx, pulseStore, checks); err != nil {
		return fmt.Errorf("seed results: %w", err)
	}

	// Create sample alerts.
	if err := seedPulseAlerts(ctx, pulseStore, checks); err != nil {
		return fmt.Errorf("seed alerts: %w", err)
	}

	return nil
}

// loadDemoDevices returns the seeded devices from recon_devices, keyed by hostname.
func loadDemoDevices(ctx context.Context, db *sql.DB) (map[string]demoDevice, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, hostname, json_extract(ip_addresses, '$[0]')
		FROM recon_devices
		WHERE hostname != ''
		ORDER BY hostname`)
	if err != nil {
		return nil, fmt.Errorf("query recon_devices: %w", err)
	}
	defer rows.Close()

	devices := make(map[string]demoDevice)
	for rows.Next() {
		var d demoDevice
		var ipNull sql.NullString
		if err := rows.Scan(&d.id, &d.hostname, &ipNull); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}
		if ipNull.Valid {
			d.ip = ipNull.String
		}
		devices[d.hostname] = d
	}
	return devices, rows.Err()
}

// pulseCheckSeed describes a monitoring check to create.
type pulseCheckSeed struct {
	hostname  string
	checkType string
	interval  int
}

// seedPulseChecks creates monitoring checks for key demo devices.
// Returns a map of check_id -> Check for result and alert seeding.
func seedPulseChecks(ctx context.Context, store *pulse.PulseStore, devices map[string]demoDevice) (map[string]*pulse.Check, error) {
	checkSeeds := []pulseCheckSeed{
		{"ubiquiti-gateway", "icmp", 30},
		{"cisco-switch-01", "icmp", 30},
		{"proxmox-host", "icmp", 30},
		{"docker-host", "icmp", 30},
		{"synology-nas", "icmp", 30},
		{"gaming-pc", "icmp", 60},
		{"smart-plug-living", "icmp", 60},
		{"cam-front-door", "icmp", 60},
	}

	checks := make(map[string]*pulse.Check, len(checkSeeds))
	now := time.Now().UTC()

	for _, cs := range checkSeeds {
		dev, ok := devices[cs.hostname]
		if !ok {
			continue // Device not seeded, skip.
		}

		// Idempotent: skip if a check already exists for this device.
		existing, err := store.GetCheckByDeviceID(ctx, dev.id)
		if err != nil {
			return nil, fmt.Errorf("check existence for %s: %w", cs.hostname, err)
		}
		if existing != nil {
			checks[existing.ID] = existing
			continue
		}

		check := &pulse.Check{
			ID:              uuid.New().String(),
			DeviceID:        dev.id,
			DeviceName:      dev.hostname,
			CheckType:       cs.checkType,
			Target:          dev.ip,
			IntervalSeconds: cs.interval,
			Enabled:         true,
			CreatedAt:       now.Add(-24 * time.Hour),
			UpdatedAt:       now,
		}
		if err := store.InsertCheck(ctx, check); err != nil {
			return nil, fmt.Errorf("insert check for %s: %w", cs.hostname, err)
		}
		checks[check.ID] = check
	}

	return checks, nil
}

// deviceProfile describes the simulated behavior of a device for result generation.
type deviceProfile struct {
	baseLatency float64 // ms
	jitter      float64 // ms standard deviation
	lossRate    float64 // 0.0-1.0 base packet loss probability
}

// deviceProfiles maps hostnames to their simulated network behavior.
var deviceProfiles = map[string]deviceProfile{
	"ubiquiti-gateway":  {baseLatency: 1.2, jitter: 0.3, lossRate: 0.0},
	"cisco-switch-01":   {baseLatency: 0.8, jitter: 0.2, lossRate: 0.0},
	"proxmox-host":      {baseLatency: 1.5, jitter: 0.5, lossRate: 0.001},
	"docker-host":       {baseLatency: 1.8, jitter: 0.6, lossRate: 0.002},
	"synology-nas":      {baseLatency: 2.0, jitter: 0.8, lossRate: 0.005},
	"gaming-pc":         {baseLatency: 3.5, jitter: 1.5, lossRate: 0.01},
	"smart-plug-living": {baseLatency: 15.0, jitter: 8.0, lossRate: 0.05},
	"cam-front-door":    {baseLatency: 8.0, jitter: 4.0, lossRate: 0.03},
}

// seedPulseResults generates 24 hours of simulated check results.
func seedPulseResults(ctx context.Context, store *pulse.PulseStore, checks map[string]*pulse.Check) error {
	now := time.Now().UTC()
	dayAgo := now.Add(-24 * time.Hour)

	for _, check := range checks {
		profile, ok := deviceProfiles[check.DeviceName]
		if !ok {
			profile = deviceProfile{baseLatency: 5.0, jitter: 2.0, lossRate: 0.01}
		}

		// Check if results already exist for this check.
		existing, err := store.ListResults(ctx, check.DeviceID, 1)
		if err != nil {
			return fmt.Errorf("check existing results for %s: %w", check.DeviceName, err)
		}
		if len(existing) > 0 {
			continue // Results already seeded.
		}

		interval := time.Duration(check.IntervalSeconds) * time.Second
		t := dayAgo

		for t.Before(now) {
			latency := profile.baseLatency + rand.NormFloat64()*profile.jitter //nolint:gosec // G404: seed data uses weak RNG intentionally
			if latency < 0.1 {
				latency = 0.1
			}

			// Simulate occasional spikes (every ~2 hours for IoT devices).
			hourOfDay := t.Hour()
			if check.DeviceName == "smart-plug-living" && (hourOfDay == 3 || hourOfDay == 15) {
				latency = profile.baseLatency * 5
			}

			packetLoss := 0.0
			success := true
			if rand.Float64() < profile.lossRate { //nolint:gosec // G404: seed data
				packetLoss = 33.3 // one of three packets lost
				if rand.Float64() < profile.lossRate*2 { //nolint:gosec // G404: seed data
					success = false
					packetLoss = 100.0
					latency = 0
				}
			}

			result := &pulse.CheckResult{
				CheckID:    check.ID,
				DeviceID:   check.DeviceID,
				Success:    success,
				LatencyMs:  math.Round(latency*100) / 100,
				PacketLoss: packetLoss,
				CheckedAt:  t,
			}
			if !success {
				result.ErrorMessage = "request timeout"
			}

			if err := store.InsertResult(ctx, result); err != nil {
				return fmt.Errorf("insert result for %s at %s: %w", check.DeviceName, t.Format(time.RFC3339), err)
			}

			t = t.Add(interval)
		}
	}

	return nil
}

// seedPulseAlerts creates sample alerts for a realistic demo dashboard.
func seedPulseAlerts(ctx context.Context, store *pulse.PulseStore, checks map[string]*pulse.Check) error {
	now := time.Now().UTC()

	// Find checks by device name for alert assignment.
	checkByDevice := make(map[string]*pulse.Check, len(checks))
	for _, c := range checks {
		checkByDevice[c.DeviceName] = c
	}

	type alertSeed struct {
		deviceName string
		severity   string
		message    string
		triggeredAgo time.Duration
		resolved     bool
		resolvedAgo  time.Duration
		acknowledged bool
		failures     int
	}

	alertSeeds := []alertSeed{
		{
			deviceName:   "smart-plug-living",
			severity:     "warning",
			message:      "High latency: avg 75ms (threshold 50ms)",
			triggeredAgo: 2 * time.Hour,
			resolved:     false,
			failures:     5,
		},
		{
			deviceName:   "synology-nas",
			severity:     "warning",
			message:      "Elevated packet loss: 15% over last 10 minutes",
			triggeredAgo: 45 * time.Minute,
			resolved:     false,
			failures:     3,
		},
		{
			deviceName:   "cam-front-door",
			severity:     "critical",
			message:      "Device unreachable: 3 consecutive failures",
			triggeredAgo: 6 * time.Hour,
			resolved:     true,
			resolvedAgo:  5 * time.Hour,
			failures:     3,
		},
		{
			deviceName:   "proxmox-host",
			severity:     "warning",
			message:      "Latency spike: 45ms (baseline 1.5ms)",
			triggeredAgo: 12 * time.Hour,
			resolved:     true,
			resolvedAgo:  11 * time.Hour,
			failures:     4,
		},
		{
			deviceName:   "gaming-pc",
			severity:     "info",
			message:      "Intermittent packet loss detected",
			triggeredAgo: 3 * time.Hour,
			resolved:     false,
			acknowledged: true,
			failures:     2,
		},
	}

	for _, as := range alertSeeds {
		check, ok := checkByDevice[as.deviceName]
		if !ok {
			continue
		}

		// Idempotent: skip if an alert already exists for this check.
		existing, err := store.GetActiveAlert(ctx, check.ID)
		if err != nil {
			return fmt.Errorf("check existing alert for %s: %w", as.deviceName, err)
		}
		if existing != nil {
			continue
		}

		triggeredAt := now.Add(-as.triggeredAgo)
		alert := &pulse.Alert{
			ID:                  uuid.New().String(),
			CheckID:             check.ID,
			DeviceID:            check.DeviceID,
			DeviceName:          check.DeviceName,
			Severity:            as.severity,
			Message:             as.message,
			TriggeredAt:         triggeredAt,
			ConsecutiveFailures: as.failures,
		}

		if as.resolved {
			resolvedAt := now.Add(-as.resolvedAgo)
			alert.ResolvedAt = &resolvedAt
		}

		if err := store.InsertAlert(ctx, alert); err != nil {
			return fmt.Errorf("insert alert for %s: %w", as.deviceName, err)
		}

		// Acknowledge after insert since there's no direct field on InsertAlert.
		if as.acknowledged && !as.resolved {
			if err := store.AcknowledgeAlert(ctx, alert.ID); err != nil {
				return fmt.Errorf("acknowledge alert for %s: %w", as.deviceName, err)
			}
		}
	}

	return nil
}
