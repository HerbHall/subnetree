package dispatch

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
)

// UpsertHardwareProfile stores or updates a hardware profile for an agent.
func (s *DispatchStore) UpsertHardwareProfile(ctx context.Context, agentID string, hw *scoutpb.HardwareProfile) error {
	data, err := json.Marshal(hw)
	if err != nil {
		return fmt.Errorf("marshal hardware profile: %w", err)
	}
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dispatch_device_profiles (agent_id, hardware_json, collected_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (agent_id) DO UPDATE SET
			hardware_json = excluded.hardware_json,
			updated_at = excluded.updated_at`,
		agentID, string(data), now, now)
	if err != nil {
		return fmt.Errorf("upsert hardware profile: %w", err)
	}
	return nil
}

// UpsertSoftwareInventory stores or updates a software inventory for an agent.
func (s *DispatchStore) UpsertSoftwareInventory(ctx context.Context, agentID string, sw *scoutpb.SoftwareInventory) error {
	data, err := json.Marshal(sw)
	if err != nil {
		return fmt.Errorf("marshal software inventory: %w", err)
	}
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dispatch_device_profiles (agent_id, software_json, collected_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (agent_id) DO UPDATE SET
			software_json = excluded.software_json,
			updated_at = excluded.updated_at`,
		agentID, string(data), now, now)
	if err != nil {
		return fmt.Errorf("upsert software inventory: %w", err)
	}
	return nil
}

// UpsertServices stores or updates the services list for an agent.
func (s *DispatchStore) UpsertServices(ctx context.Context, agentID string, services []*scoutpb.ServiceInfo) error {
	data, err := json.Marshal(services)
	if err != nil {
		return fmt.Errorf("marshal services: %w", err)
	}
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dispatch_device_profiles (agent_id, services_json, collected_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (agent_id) DO UPDATE SET
			services_json = excluded.services_json,
			updated_at = excluded.updated_at`,
		agentID, string(data), now, now)
	if err != nil {
		return fmt.Errorf("upsert services: %w", err)
	}
	return nil
}

// UpsertFullProfile stores hardware, software, and services in one operation.
func (s *DispatchStore) UpsertFullProfile(ctx context.Context, agentID string, hw *scoutpb.HardwareProfile, sw *scoutpb.SoftwareInventory, services []*scoutpb.ServiceInfo) error {
	hwJSON, err := json.Marshal(hw)
	if err != nil {
		return fmt.Errorf("marshal hardware: %w", err)
	}
	swJSON, err := json.Marshal(sw)
	if err != nil {
		return fmt.Errorf("marshal software: %w", err)
	}
	svcJSON, err := json.Marshal(services)
	if err != nil {
		return fmt.Errorf("marshal services: %w", err)
	}

	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dispatch_device_profiles (agent_id, hardware_json, software_json, services_json, collected_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (agent_id) DO UPDATE SET
			hardware_json = excluded.hardware_json,
			software_json = excluded.software_json,
			services_json = excluded.services_json,
			collected_at = excluded.collected_at,
			updated_at = excluded.updated_at`,
		agentID, string(hwJSON), string(swJSON), string(svcJSON), now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert full profile: %w", err)
	}
	return nil
}

// GetHardwareProfile retrieves the stored hardware profile for an agent.
func (s *DispatchStore) GetHardwareProfile(ctx context.Context, agentID string) (*scoutpb.HardwareProfile, error) {
	var jsonStr string
	err := s.db.QueryRowContext(ctx, `SELECT hardware_json FROM dispatch_device_profiles WHERE agent_id = ?`, agentID).Scan(&jsonStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get hardware profile: %w", err)
	}

	var hw scoutpb.HardwareProfile
	if err := json.Unmarshal([]byte(jsonStr), &hw); err != nil {
		return nil, fmt.Errorf("unmarshal hardware profile: %w", err)
	}
	return &hw, nil
}

// GetSoftwareInventory retrieves the stored software inventory for an agent.
func (s *DispatchStore) GetSoftwareInventory(ctx context.Context, agentID string) (*scoutpb.SoftwareInventory, error) {
	var jsonStr string
	err := s.db.QueryRowContext(ctx, `SELECT software_json FROM dispatch_device_profiles WHERE agent_id = ?`, agentID).Scan(&jsonStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get software inventory: %w", err)
	}

	var sw scoutpb.SoftwareInventory
	if err := json.Unmarshal([]byte(jsonStr), &sw); err != nil {
		return nil, fmt.Errorf("unmarshal software inventory: %w", err)
	}
	return &sw, nil
}

// GetServices retrieves the stored services list for an agent.
func (s *DispatchStore) GetServices(ctx context.Context, agentID string) ([]*scoutpb.ServiceInfo, error) {
	var jsonStr string
	err := s.db.QueryRowContext(ctx, `SELECT services_json FROM dispatch_device_profiles WHERE agent_id = ?`, agentID).Scan(&jsonStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get services: %w", err)
	}

	var services []*scoutpb.ServiceInfo
	if err := json.Unmarshal([]byte(jsonStr), &services); err != nil {
		return nil, fmt.Errorf("unmarshal services: %w", err)
	}
	return services, nil
}

