package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Agent is the Scout monitoring agent.
type Agent struct {
	config *Config
	logger *zap.Logger
	cancel context.CancelFunc
	conn   *grpc.ClientConn
	client scoutpb.ScoutServiceClient
}

// NewAgent creates a new Scout agent instance.
func NewAgent(config *Config, logger *zap.Logger) *Agent {
	return &Agent{
		config: config,
		logger: logger,
	}
}

// Run starts the agent and blocks until the context is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)

	// Load persisted agent ID.
	a.loadAgentID()

	// Connect with exponential backoff.
	if err := a.connectWithBackoff(ctx); err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	defer func() {
		if a.conn != nil {
			a.conn.Close()
		}
	}()

	// If no agent ID, attempt enrollment.
	if a.config.AgentID == "" {
		if a.config.EnrollToken == "" {
			return fmt.Errorf("no agent ID and no enrollment token provided")
		}
		if err := a.enroll(ctx); err != nil {
			return fmt.Errorf("enrollment: %w", err)
		}
	}

	a.logger.Info("agent running",
		zap.String("agent_id", a.config.AgentID),
		zap.String("server", a.config.ServerAddr),
		zap.Int("interval", a.config.CheckInterval),
	)

	ticker := time.NewTicker(time.Duration(a.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	// Initial check-in.
	a.checkIn(ctx)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("agent shutting down")
			return nil
		case <-ticker.C:
			a.checkIn(ctx)
		}
	}
}

// Stop signals the agent to shut down.
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Agent) connectWithBackoff(ctx context.Context) error {
	backoff := time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := grpc.NewClient(
			a.config.ServerAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err == nil {
			a.conn = conn
			a.client = scoutpb.NewScoutServiceClient(conn)
			a.logger.Info("connected to server", zap.String("addr", a.config.ServerAddr))
			return nil
		}

		a.logger.Warn("connection failed, retrying",
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
	}
}

func (a *Agent) enroll(ctx context.Context) error {
	resp, err := a.client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:     hostname(),
		Platform:     agentPlatform(),
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		EnrollToken:  a.config.EnrollToken,
	})
	if err != nil {
		return fmt.Errorf("enrollment check-in: %w", err)
	}

	if !resp.Acknowledged {
		return fmt.Errorf("enrollment rejected: %s", resp.UpgradeMessage)
	}

	if resp.AssignedAgentId == "" {
		return fmt.Errorf("server did not assign an agent ID")
	}

	a.config.AgentID = resp.AssignedAgentId
	a.saveAgentID()

	a.logger.Info("enrolled successfully", zap.String("agent_id", a.config.AgentID))
	return nil
}

func (a *Agent) checkIn(ctx context.Context) {
	resp, err := a.client.CheckIn(ctx, &scoutpb.CheckInRequest{
		AgentId:      a.config.AgentID,
		Hostname:     hostname(),
		Platform:     agentPlatform(),
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
	})
	if err != nil {
		a.logger.Warn("check-in failed", zap.Error(err))
		return
	}

	if !resp.Acknowledged {
		a.logger.Warn("check-in not acknowledged",
			zap.String("version_status", resp.VersionStatus.String()),
			zap.String("message", resp.UpgradeMessage),
		)
	}
}

// Agent ID persistence -- simple JSON file in config directory.
type agentState struct {
	AgentID string `json:"agent_id"`
}

func (a *Agent) statePath() string {
	dir := filepath.Dir(a.config.CertPath)
	if dir == "" || dir == "." {
		dir, _ = os.UserConfigDir()
		if dir == "" {
			dir = "."
		}
		dir = filepath.Join(dir, "subnetree-scout")
	}
	return filepath.Join(dir, "agent-state.json")
}

func (a *Agent) loadAgentID() {
	data, err := os.ReadFile(a.statePath())
	if err != nil {
		return
	}
	var state agentState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	if state.AgentID != "" && a.config.AgentID == "" {
		a.config.AgentID = state.AgentID
		a.logger.Info("loaded persisted agent ID", zap.String("agent_id", state.AgentID))
	}
}

func (a *Agent) saveAgentID() {
	data, err := json.Marshal(agentState{AgentID: a.config.AgentID})
	if err != nil {
		a.logger.Warn("failed to marshal agent state", zap.Error(err))
		return
	}
	dir := filepath.Dir(a.statePath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		a.logger.Warn("failed to create state dir", zap.Error(err))
		return
	}
	if err := os.WriteFile(a.statePath(), data, 0o600); err != nil {
		a.logger.Warn("failed to save agent state", zap.Error(err))
	}
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func agentPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
