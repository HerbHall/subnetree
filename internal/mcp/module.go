package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin       = (*Module)(nil)
	_ plugin.HTTPProvider = (*Module)(nil)
)

// DeviceQuerier abstracts device data access for the MCP module.
// Implemented by the recon module; resolved at runtime via PluginResolver.
type DeviceQuerier interface {
	GetDevice(ctx context.Context, id string) (*models.Device, error)
	ListDevices(ctx context.Context, limit, offset int) ([]models.Device, int, error)
	GetDeviceHardware(ctx context.Context, deviceID string) (*models.DeviceHardware, error)
	GetHardwareSummary(ctx context.Context) (*models.HardwareSummary, error)
	QueryDevicesByHardware(ctx context.Context, query models.HardwareQuery) ([]models.Device, int, error)
}

// Module implements the MCP (Model Context Protocol) server plugin.
// It exposes SubNetree device data to external AI tools via the MCP protocol.
type Module struct {
	logger  *zap.Logger
	bus     plugin.EventBus
	querier DeviceQuerier
	server  *sdkmcp.Server
	apiKey  string
}

// New creates a new MCP plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:         "mcp",
		Version:      "0.1.0",
		Description:  "Model Context Protocol server for AI tool integration",
		Dependencies: []string{"recon"},
		APIVersion:   plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.bus = deps.Bus

	if deps.Config != nil {
		m.apiKey = deps.Config.GetString("api_key")
	}

	m.logger.Info("mcp module initialized")
	return nil
}

// SetQuerier injects the device querier. Called from the composition root
// (main.go) to wire the recon module's store without cross-internal imports.
func (m *Module) SetQuerier(q DeviceQuerier) {
	m.querier = q
}

func (m *Module) Start(_ context.Context) error {
	m.server = sdkmcp.NewServer(
		&sdkmcp.Implementation{
			Name:    "subnetree",
			Version: "0.1.0",
		},
		nil,
	)

	m.registerTools()

	m.logger.Info("mcp module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("mcp module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
// The MCP streamable HTTP handler is mounted at the plugin's route prefix.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "POST", Path: "/", Handler: m.handleMCP},
		{Method: "GET", Path: "/", Handler: m.handleMCP},
		{Method: "DELETE", Path: "/", Handler: m.handleMCP},
	}
}

// handleMCP wraps the MCP streamable HTTP handler with optional API key auth.
func (m *Module) handleMCP(w http.ResponseWriter, r *http.Request) {
	if m.apiKey != "" {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != m.apiKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	if m.server == nil {
		http.Error(w, `{"error":"mcp server not started"}`, http.StatusServiceUnavailable)
		return
	}

	handler := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return m.server },
		nil,
	)
	handler.ServeHTTP(w, r)
}

// publishToolCall emits an event when an MCP tool is invoked.
func (m *Module) publishToolCall(toolName string, params any) {
	if m.bus == nil {
		return
	}

	m.bus.PublishAsync(context.Background(), plugin.Event{
		Topic:     "mcp.tool.called",
		Source:    "mcp",
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"tool":   toolName,
			"params": params,
		},
	})
}

// writeToolJSON marshals v to JSON for tool responses.
func writeToolJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"failed to marshal response"}`
	}
	return string(data)
}
