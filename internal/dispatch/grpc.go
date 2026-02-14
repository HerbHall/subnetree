package dispatch

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/ca"
	"github.com/HerbHall/subnetree/internal/version"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// Compile-time guard.
var _ scoutpb.ScoutServiceServer = (*scoutServer)(nil)

// currentProtoVersion is the protocol version supported by the server.
const currentProtoVersion uint32 = 1

type scoutServer struct {
	scoutpb.UnimplementedScoutServiceServer
	store     *DispatchStore
	bus       plugin.EventBus
	logger    *zap.Logger
	cfg       DispatchConfig
	authority *ca.Authority
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
	var certDER, caCertDER []byte
	if agentID == "" && req.EnrollToken != "" {
		result, err := s.enrollAgent(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("enrollment failed: %w", err)
		}
		agentID = result.agentID
		assignedID = result.agentID
		certDER = result.certDER
		caCertDER = result.caCertDER
	} else if agentID == "" {
		return nil, fmt.Errorf("agent_id is required for check-in (use enroll_token for initial enrollment)")
	}

	// 2b. Verify client certificate identity matches agent_id when present.
	if certCN, ok := extractAgentIDFromCert(ctx); ok && assignedID == "" {
		if certCN != agentID {
			s.logger.Warn("agent_id mismatch with client certificate",
				zap.String("agent_id", agentID),
				zap.String("cert_cn", certCN),
			)
			return nil, fmt.Errorf("agent_id %q does not match client certificate CN %q", agentID, certCN)
		}
		s.logger.Debug("client certificate identity verified",
			zap.String("agent_id", agentID),
		)
	}

	// 3. Handle certificate renewal for existing agents.
	if assignedID == "" && len(req.CertificateRequest) > 0 && s.authority != nil {
		renewedCert, serial, expiresAt, signErr := s.authority.SignCSR(
			req.CertificateRequest, agentID, s.cfg.CAConfig.Validity,
		)
		if signErr != nil {
			s.logger.Error("failed to sign renewal CSR",
				zap.String("agent_id", agentID),
				zap.Error(signErr),
			)
		} else {
			certDER = renewedCert
			caCertDER = s.authority.CACertDER()
			if err := s.store.UpdateAgentCert(ctx, agentID, serial, expiresAt); err != nil {
				s.logger.Warn("failed to update agent cert fields",
					zap.String("agent_id", agentID),
					zap.Error(err),
				)
			}
			s.logger.Info("renewed agent certificate",
				zap.String("agent_id", agentID),
				zap.String("cert_serial", serial),
				zap.Time("cert_expires", expiresAt),
			)
		}
	}

	// 4. Update check-in in store.
	if err := s.store.UpdateCheckIn(ctx, agentID, req.Hostname, req.Platform, req.AgentVersion, int(req.ProtoVersion)); err != nil {
		s.logger.Warn("check-in update failed", zap.String("agent_id", agentID), zap.Error(err))
	}

	// 5. Log and publish metrics if present.
	payload := map[string]string{
		"agent_id": agentID,
		"hostname": req.Hostname,
		"platform": req.Platform,
	}
	if req.Metrics != nil {
		s.logger.Debug("received agent metrics",
			zap.String("agent_id", agentID),
			zap.Float64("cpu_percent", req.Metrics.CpuPercent),
			zap.Float64("memory_percent", req.Metrics.MemoryPercent),
			zap.Float64("memory_used_bytes", req.Metrics.MemoryUsedBytes),
			zap.Float64("memory_total_bytes", req.Metrics.MemoryTotalBytes),
			zap.Int("disk_count", len(req.Metrics.Disks)),
			zap.Int("network_count", len(req.Metrics.Networks)),
		)
		payload["cpu_percent"] = strconv.FormatFloat(req.Metrics.CpuPercent, 'f', 2, 64)
		payload["memory_percent"] = strconv.FormatFloat(req.Metrics.MemoryPercent, 'f', 2, 64)
	}
	if s.bus != nil {
		_ = s.bus.Publish(ctx, plugin.Event{
			Topic:     TopicAgentCheckIn,
			Source:    "dispatch",
			Timestamp: time.Now(),
			Payload:   payload,
		})
	}

	return &scoutpb.CheckInResponse{
		Acknowledged:         true,
		CheckIntervalSeconds: 30,
		VersionStatus:        versionStatus,
		ServerVersion:        version.Version,
		AssignedAgentId:      assignedID,
		SignedCertificate:    certDER,
		CaCertificate:        caCertDER,
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

func (s *scoutServer) ReportProfile(ctx context.Context, req *scoutpb.ProfileReport) (*scoutpb.Ack, error) {
	if req.AgentId == "" {
		return &scoutpb.Ack{Success: false}, fmt.Errorf("agent_id is required")
	}

	profile := req.GetProfile()
	if profile == nil {
		return &scoutpb.Ack{Success: false}, fmt.Errorf("profile is required")
	}

	hw := profile.GetHardware()
	if hw == nil {
		hw = &scoutpb.HardwareProfile{}
	}
	sw := profile.GetSoftware()
	if sw == nil {
		sw = &scoutpb.SoftwareInventory{}
	}
	services := profile.GetServices()

	if err := s.store.UpsertFullProfile(ctx, req.AgentId, hw, sw, services); err != nil {
		s.logger.Error("failed to store profile",
			zap.String("agent_id", req.AgentId),
			zap.Error(err),
		)
		return &scoutpb.Ack{Success: false}, fmt.Errorf("store profile: %w", err)
	}

	s.logger.Info("profile received",
		zap.String("agent_id", req.AgentId),
		zap.String("cpu_model", hw.GetCpuModel()),
		zap.String("os_name", sw.GetOsName()),
		zap.Int("services_count", len(services)),
	)

	if s.bus != nil {
		_ = s.bus.Publish(ctx, plugin.Event{
			Topic:     TopicDeviceProfiled,
			Source:    "dispatch",
			Timestamp: time.Now(),
			Payload: map[string]string{
				"agent_id":  req.AgentId,
				"cpu_model": hw.GetCpuModel(),
				"os_name":   sw.GetOsName(),
			},
		})
	}

	return &scoutpb.Ack{Success: true}, nil
}

// enrollResult holds the outcome of an enrollment including optional certificate data.
type enrollResult struct {
	agentID       string
	certDER       []byte
	caCertDER     []byte
}

func (s *scoutServer) enrollAgent(ctx context.Context, req *scoutpb.CheckInRequest) (*enrollResult, error) {
	// Hash the token.
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.EnrollToken)))

	// Validate the token.
	_, err := s.store.ValidateEnrollmentToken(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("invalid enrollment token: %w", err)
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

	result := &enrollResult{agentID: agentID}

	// Sign CSR if provided (mTLS-capable agent).
	if len(req.CertificateRequest) > 0 && s.authority != nil {
		certDER, serial, expiresAt, signErr := s.authority.SignCSR(
			req.CertificateRequest, agentID, s.cfg.CAConfig.Validity,
		)
		if signErr != nil {
			return nil, fmt.Errorf("sign agent CSR: %w", signErr)
		}
		agent.CertSerial = serial
		agent.CertExpires = &expiresAt
		result.certDER = certDER
		result.caCertDER = s.authority.CACertDER()

		s.logger.Info("signed agent certificate during enrollment",
			zap.String("agent_id", agentID),
			zap.String("cert_serial", serial),
			zap.Time("cert_expires", expiresAt),
		)
	}

	if err := s.store.UpsertAgent(ctx, agent); err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
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

	return result, nil
}

// extractAgentIDFromCert extracts the agent ID (CommonName) from the client's
// TLS certificate presented during mTLS. Returns empty string and false if no
// client certificate is present (e.g., during enrollment or insecure connections).
func extractAgentIDFromCert(ctx context.Context) (string, bool) {
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return "", false
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok || len(tlsInfo.State.PeerCertificates) == 0 {
		return "", false
	}
	return tlsInfo.State.PeerCertificates[0].Subject.CommonName, true
}
