package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/internal/svcmap"
	"github.com/HerbHall/subnetree/internal/version"
	"github.com/HerbHall/subnetree/pkg/models"
)

// runMCPStdio runs a standalone MCP server over stdio for Claude Desktop integration.
// It opens the database read-only and registers the same tools as the HTTP module.
func runMCPStdio() {
	dbPath := "subnetree.db"
	if p := os.Getenv("SUBNETREE_DB_PATH"); p != "" {
		dbPath = p
	}

	db, err := store.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	reconStore := recon.NewReconStore(db.DB())
	adapter := &mcpStdioAdapter{store: reconStore}

	svcStore, err := svcmap.NewStore(db.DB())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize svcmap store: %v\n", err)
		os.Exit(1)
	}

	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{
			Name:    "subnetree",
			Version: version.Short(),
		},
		nil,
	)

	registerStdioTools(server, adapter, svcStore)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	err = server.Run(ctx, &sdkmcp.StdioTransport{})
	cancel()
	db.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		os.Exit(1)
	}
}

// mcpStdioAdapter provides DeviceQuerier-compatible methods wrapping a ReconStore.
type mcpStdioAdapter struct {
	store *recon.ReconStore
}

func (a *mcpStdioAdapter) GetDevice(ctx context.Context, id string) (*models.Device, error) {
	return a.store.GetDevice(ctx, id)
}

func (a *mcpStdioAdapter) ListDevices(ctx context.Context, limit, offset int) ([]models.Device, int, error) {
	return a.store.ListDevices(ctx, recon.ListDevicesOptions{
		Limit:  limit,
		Offset: offset,
	})
}

func (a *mcpStdioAdapter) GetDeviceHardware(ctx context.Context, deviceID string) (*models.DeviceHardware, error) {
	return a.store.GetDeviceHardware(ctx, deviceID)
}

func (a *mcpStdioAdapter) GetHardwareSummary(ctx context.Context) (*models.HardwareSummary, error) {
	return a.store.GetHardwareSummary(ctx)
}

func (a *mcpStdioAdapter) QueryDevicesByHardware(ctx context.Context, query models.HardwareQuery) ([]models.Device, int, error) {
	return a.store.QueryDevicesByHardware(ctx, query)
}

func (a *mcpStdioAdapter) FindStaleDevices(ctx context.Context, threshold time.Time) ([]models.Device, error) {
	return a.store.FindStaleDevices(ctx, threshold)
}

// registerStdioTools registers MCP tools on a standalone server for stdio mode.
// These mirror the tools in internal/mcp/tools.go but use the adapter directly.
func registerStdioTools(server *sdkmcp.Server, adapter *mcpStdioAdapter, svcStore *svcmap.Store) {
	type getDeviceArgs struct {
		DeviceID string `json:"device_id" jsonschema:"The unique device identifier"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_device",
		Description: "Get detailed information about a single network device by its ID.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args getDeviceArgs) (*sdkmcp.CallToolResult, any, error) {
		device, err := adapter.GetDevice(ctx, args.DeviceID)
		if err != nil {
			return stdioErrorResult(fmt.Sprintf("failed to get device: %v", err)), nil, nil
		}
		if device == nil {
			return stdioTextResult(fmt.Sprintf("No device found with ID %q", args.DeviceID)), nil, nil
		}
		return stdioTextResult(stdioJSON(device)), nil, nil
	})

	type listDevicesArgs struct {
		Limit  int `json:"limit,omitempty" jsonschema:"Maximum number of devices to return (default 50)"`
		Offset int `json:"offset,omitempty" jsonschema:"Number of devices to skip"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_devices",
		Description: "List all discovered network devices with pagination.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args listDevicesArgs) (*sdkmcp.CallToolResult, any, error) {
		limit := args.Limit
		if limit <= 0 {
			limit = 50
		}
		devices, total, err := adapter.ListDevices(ctx, limit, args.Offset)
		if err != nil {
			return stdioErrorResult(fmt.Sprintf("failed to list devices: %v", err)), nil, nil
		}
		resp := map[string]any{"devices": devices, "total": total, "limit": limit, "offset": args.Offset}
		return stdioTextResult(stdioJSON(resp)), nil, nil
	})

	type getHardwareArgs struct {
		DeviceID string `json:"device_id" jsonschema:"The unique device identifier"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_hardware_profile",
		Description: "Get full hardware profile for a device.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args getHardwareArgs) (*sdkmcp.CallToolResult, any, error) {
		hw, err := adapter.GetDeviceHardware(ctx, args.DeviceID)
		if err != nil {
			return stdioErrorResult(fmt.Sprintf("failed to get hardware: %v", err)), nil, nil
		}
		if hw == nil {
			return stdioTextResult(fmt.Sprintf("No hardware profile for device %q", args.DeviceID)), nil, nil
		}
		return stdioTextResult(stdioJSON(hw)), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_fleet_summary",
		Description: "Get aggregate hardware statistics across all devices.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		summary, err := adapter.GetHardwareSummary(ctx)
		if err != nil {
			return stdioErrorResult(fmt.Sprintf("failed to get summary: %v", err)), nil, nil
		}
		if summary == nil {
			return stdioTextResult("No hardware summary available"), nil, nil
		}
		return stdioTextResult(stdioJSON(summary)), nil, nil
	})

	type queryArgs struct {
		MinRAMMB     int    `json:"min_ram_mb,omitempty"`
		MaxRAMMB     int    `json:"max_ram_mb,omitempty"`
		CPUModel     string `json:"cpu_model,omitempty"`
		OSName       string `json:"os_name,omitempty"`
		PlatformType string `json:"platform_type,omitempty"`
		GPUVendor    string `json:"gpu_vendor,omitempty"`
		Limit        int    `json:"limit,omitempty"`
		Offset       int    `json:"offset,omitempty"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "query_devices",
		Description: "Query devices by hardware characteristics.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args queryArgs) (*sdkmcp.CallToolResult, any, error) {
		limit := args.Limit
		if limit <= 0 {
			limit = 50
		}
		query := models.HardwareQuery{
			MinRAMMB: args.MinRAMMB, MaxRAMMB: args.MaxRAMMB,
			CPUModel: args.CPUModel, OSName: args.OSName,
			PlatformType: args.PlatformType, GPUVendor: args.GPUVendor,
			Limit: limit, Offset: args.Offset,
		}
		devices, total, err := adapter.QueryDevicesByHardware(ctx, query)
		if err != nil {
			return stdioErrorResult(fmt.Sprintf("failed to query: %v", err)), nil, nil
		}
		resp := map[string]any{"devices": devices, "total": total, "limit": limit, "offset": args.Offset}
		return stdioTextResult(stdioJSON(resp)), nil, nil
	})

	type staleArgs struct {
		StaleAfterHours int `json:"stale_after_hours,omitempty" jsonschema:"Hours since last seen to consider stale (default 24)"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_stale_devices",
		Description: "Get devices that are marked online but haven't been seen within the specified time window.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args staleArgs) (*sdkmcp.CallToolResult, any, error) {
		hours := args.StaleAfterHours
		if hours <= 0 {
			hours = 24
		}
		threshold := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
		devices, err := adapter.FindStaleDevices(ctx, threshold)
		if err != nil {
			return stdioErrorResult(fmt.Sprintf("failed to find stale devices: %v", err)), nil, nil
		}
		resp := map[string]any{"devices": devices, "count": len(devices), "stale_after_hours": hours}
		return stdioTextResult(stdioJSON(resp)), nil, nil
	})

	type serviceArgs struct {
		DeviceID    string `json:"device_id,omitempty" jsonschema:"Filter by device ID"`
		ServiceType string `json:"service_type,omitempty" jsonschema:"Filter by type: docker-container, systemd-service, windows-service, application"`
		Status      string `json:"status,omitempty" jsonschema:"Filter by status: running, stopped, failed, unknown"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_service_inventory",
		Description: "Get tracked services optionally filtered by device, type, or status.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args serviceArgs) (*sdkmcp.CallToolResult, any, error) {
		if svcStore == nil {
			return stdioTextResult("Service data not available."), nil, nil
		}
		services, err := svcStore.ListServicesFiltered(ctx, svcmap.ServiceFilter{
			DeviceID:    args.DeviceID,
			ServiceType: args.ServiceType,
			Status:      args.Status,
		})
		if err != nil {
			return stdioErrorResult(fmt.Sprintf("failed to list services: %v", err)), nil, nil
		}
		grouped := make(map[string][]models.Service)
		for i := range services {
			grouped[services[i].DeviceID] = append(grouped[services[i].DeviceID], services[i])
		}
		resp := map[string]any{"services_by_device": grouped, "total_services": len(services)}
		return stdioTextResult(stdioJSON(resp)), nil, nil
	})
}

func stdioTextResult(text string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: text},
		},
	}
}

func stdioErrorResult(msg string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}

func stdioJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"failed to marshal response"}`
	}
	return string(data)
}
