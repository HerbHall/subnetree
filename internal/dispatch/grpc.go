package dispatch

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/version"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Compile-time guard.
var _ scoutpb.ScoutServiceServer = (*scoutServer)(nil)

// currentProtoVersion is the protocol version supported by the server.
const currentProtoVersion uint32 = 1

type scoutServer struct {
	scoutpb.UnimplementedScoutServiceServer
	store  *DispatchStore
	bus    plugin.EventBus
	logger *zap.Logger
	cfg    DispatchConfig
}

func (s *scoutServer) CheckIn(ctx context.Context, req *scoutpb.CheckInRequest) (*scoutpb.CheckInResponse, error) {
	// 1. Check proto_version compatibility.
	versionStatus := s.checkProtoVersion(req.ProtoVersion)
	if versionStatus == scoutpb.VersionStatus_VERSION_REJECTED {
		return &scoutpb.CheckInResponse{
			Acknowledged:   false,
			VersionStatus:  versionStatus,
			ServerVersion:  version.Version,
			UpgradeMessage: fmt.Sprintf("proto version %d is not supported; server requires version %d", req.ProtoVersion, currentProtoVersion),
		}, nil
	}

	// 2. Handle enrollment if no agent_id but enroll_token is present.
	agentID := req.AgentId
	assignedID := ""
	if agentID == "" && req.EnrollToken != "" {
		newID, err := s.enrollAgent(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("enrollment failed: %w", err)
		}
		agentID = newID
		assignedID = newID
	} else if agentID == "" {
		return nil, fmt.Errorf("agent_id is required for check-in (use enroll_token for initial enrollment)")
	}

	// 3. Update check-in in store.
	if err := s.store.UpdateCheckIn(ctx, agentID, req.Hostname, req.Platform, req.AgentVersion, int(req.ProtoVersion)); err != nil {
		s.logger.Warn("check-in update failed", zap.String("agent_id", agentID), zap.Error(err))
	}

	// 4. Publish event.
	if s.bus != nil {
		_ = s.bus.Publish(ctx, plugin.Event{
			Topic:     TopicAgentCheckIn,
			Source:    "dispatch",
			Timestamp: time.Now(),
			Payload: map[string]string{
				"agent_id": agentID,
				"hostname": req.Hostname,
				"platform": req.Platform,
			},
		})
	}

	return &scoutpb.CheckInResponse{
		Acknowledged:         true,
		CheckIntervalSeconds: 30,
		VersionStatus:        versionStatus,
		ServerVersion:        version.Version,
		AssignedAgentId:      assignedID,
	}, nil
}

func (s *scoutServer) checkProtoVersion(v uint32) scoutpb.VersionStatus {
	if v == currentProtoVersion {
		return scoutpb.VersionStatus_VERSION_OK
	}
	if v == currentProtoVersion-1 {
		return scoutpb.VersionStatus_VERSION_DEPRECATED
	}
	return scoutpb.VersionStatus_VERSION_REJECTED
}

func (s *scoutServer) enrollAgent(ctx context.Context, req *scoutpb.CheckInRequest) (string, error) {
	// Hash the token.
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.EnrollToken)))

	// Validate the token.
	_, err := s.store.ValidateEnrollmentToken(ctx, hash)
	if err != nil {
		return "", fmt.Errorf("invalid enrollment token: %w", err)
	}

	// Create agent record.
	agentID := uuid.New().String()
	now := time.Now().UTC()
	agent := &Agent{
		ID:           agentID,
		Hostname:     req.Hostname,
		Platform:     req.Platform,
		AgentVersion: req.AgentVersion,
		ProtoVersion: int(req.ProtoVersion),
		Status:       "connected",
		EnrolledAt:   now,
		LastCheckIn:  &now,
		ConfigJSON:   "{}",
	}

	if err := s.store.UpsertAgent(ctx, agent); err != nil {
		return "", fmt.Errorf("create agent: %w", err)
	}

	// Consume the token.
	if err := s.store.ConsumeEnrollmentToken(ctx, hash, agentID); err != nil {
		s.logger.Warn("failed to consume enrollment token", zap.Error(err))
	}

	// Publish enrollment event.
	if s.bus != nil {
		_ = s.bus.Publish(ctx, plugin.Event{
			Topic:     TopicAgentEnrolled,
			Source:    "dispatch",
			Timestamp: time.Now(),
			Payload: map[string]string{
				"agent_id": agentID,
				"hostname": req.Hostname,
			},
		})
	}

	s.logger.Info("agent enrolled",
		zap.String("agent_id", agentID),
		zap.String("hostname", req.Hostname),
		zap.String("platform", req.Platform),
	)

	return agentID, nil
}
