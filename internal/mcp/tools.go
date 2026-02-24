package mcp

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/HerbHall/subnetree/pkg/models"
)

// Tool input types.

type getDeviceInput struct {
	DeviceID string `json:"device_id" jsonschema:"The unique device identifier"`
}

type listDevicesInput struct {
	Limit  int `json:"limit,omitempty" jsonschema:"Maximum number of devices to return (default 50)"`
	Offset int `json:"offset,omitempty" jsonschema:"Number of devices to skip for pagination"`
}

type getHardwareProfileInput struct {
	DeviceID string `json:"device_id" jsonschema:"The unique device identifier"`
}

type queryDevicesInput struct {
	MinRAMMB     int    `json:"min_ram_mb,omitempty" jsonschema:"Minimum RAM in megabytes"`
	MaxRAMMB     int    `json:"max_ram_mb,omitempty" jsonschema:"Maximum RAM in megabytes"`
	CPUModel     string `json:"cpu_model,omitempty" jsonschema:"CPU model substring filter"`
	OSName       string `json:"os_name,omitempty" jsonschema:"Operating system name filter"`
	PlatformType string `json:"platform_type,omitempty" jsonschema:"Platform type filter (baremetal/vm/container)"`
	GPUVendor    string `json:"gpu_vendor,omitempty" jsonschema:"GPU vendor filter (nvidia/amd/intel)"`
	Limit        int    `json:"limit,omitempty" jsonschema:"Maximum number of devices to return (default 50)"`
	Offset       int    `json:"offset,omitempty" jsonschema:"Number of devices to skip for pagination"`
}

type getStaleDevicesInput struct {
	StaleAfterHours int `json:"stale_after_hours,omitempty" jsonschema:"Hours since last seen to consider stale (default 24)"`
}

type getServiceInventoryInput struct {
	DeviceID    string `json:"device_id,omitempty" jsonschema:"Filter by device ID"`
	ServiceType string `json:"service_type,omitempty" jsonschema:"Filter by type: docker-container, systemd-service, windows-service, application"`
	Status      string `json:"status,omitempty" jsonschema:"Filter by status: running, stopped, failed, unknown"`
}

// registerTools adds all MCP tools to the server.
func (m *Module) registerTools() {
	sdkmcp.AddTool(m.server, &sdkmcp.Tool{
		Name:        "get_device",
		Description: "Get detailed information about a single network device by its ID, including hostname, IP addresses, device type, status, and classification metadata.",
	}, m.handleGetDevice)

	sdkmcp.AddTool(m.server, &sdkmcp.Tool{
		Name:        "list_devices",
		Description: "List all discovered network devices with pagination. Returns devices with basic metadata and a total count.",
	}, m.handleListDevices)

	sdkmcp.AddTool(m.server, &sdkmcp.Tool{
		Name:        "get_hardware_profile",
		Description: "Get the full hardware profile for a device, including CPU, RAM, storage, GPU, and services.",
	}, m.handleGetHardwareProfile)

	sdkmcp.AddTool(m.server, &sdkmcp.Tool{
		Name:        "get_fleet_summary",
		Description: "Get aggregate hardware statistics across all devices: total RAM, storage, GPU counts, and breakdowns by OS, CPU model, and platform type.",
	}, m.handleGetFleetSummary)

	sdkmcp.AddTool(m.server, &sdkmcp.Tool{
		Name:        "query_devices",
		Description: "Query devices by hardware characteristics. Filter by minimum/maximum RAM, CPU model, OS, platform type, or GPU vendor.",
	}, m.handleQueryDevices)

	sdkmcp.AddTool(m.server, &sdkmcp.Tool{
		Name:        "get_stale_devices",
		Description: "Get devices that are marked online but haven't been seen within the specified time window. Useful for identifying potentially offline or unreachable devices.",
	}, m.handleGetStaleDevices)

	sdkmcp.AddTool(m.server, &sdkmcp.Tool{
		Name:        "get_service_inventory",
		Description: "Get tracked services (Docker containers, systemd services, Windows services, applications) optionally filtered by device, type, or status.",
	}, m.handleGetServiceInventory)
}

func (m *Module) handleGetDevice(_ context.Context, _ *sdkmcp.CallToolRequest, input getDeviceInput) (*sdkmcp.CallToolResult, any, error) {
	m.publishToolCall("get_device", input)

	if m.querier == nil {
		return textResult("Device querier not available. The recon module may not be loaded."), nil, nil
	}

	device, err := m.querier.GetDevice(context.Background(), input.DeviceID)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get device: %v", err)), nil, nil
	}
	if device == nil {
		return textResult(fmt.Sprintf("No device found with ID %q", input.DeviceID)), nil, nil
	}

	return textResult(writeToolJSON(device)), nil, nil
}

func (m *Module) handleListDevices(_ context.Context, _ *sdkmcp.CallToolRequest, input listDevicesInput) (*sdkmcp.CallToolResult, any, error) {
	m.publishToolCall("list_devices", input)

	if m.querier == nil {
		return textResult("Device querier not available. The recon module may not be loaded."), nil, nil
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	devices, total, err := m.querier.ListDevices(context.Background(), limit, input.Offset)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list devices: %v", err)), nil, nil
	}

	resp := struct {
		Devices []models.Device `json:"devices"`
		Total   int             `json:"total"`
		Limit   int             `json:"limit"`
		Offset  int             `json:"offset"`
	}{
		Devices: devices,
		Total:   total,
		Limit:   limit,
		Offset:  input.Offset,
	}

	return textResult(writeToolJSON(resp)), nil, nil
}

func (m *Module) handleGetHardwareProfile(_ context.Context, _ *sdkmcp.CallToolRequest, input getHardwareProfileInput) (*sdkmcp.CallToolResult, any, error) {
	m.publishToolCall("get_hardware_profile", input)

	if m.querier == nil {
		return textResult("Device querier not available. The recon module may not be loaded."), nil, nil
	}

	hw, err := m.querier.GetDeviceHardware(context.Background(), input.DeviceID)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get hardware profile: %v", err)), nil, nil
	}
	if hw == nil {
		return textResult(fmt.Sprintf("No hardware profile found for device %q", input.DeviceID)), nil, nil
	}

	return textResult(writeToolJSON(hw)), nil, nil
}

func (m *Module) handleGetFleetSummary(_ context.Context, _ *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
	m.publishToolCall("get_fleet_summary", nil)

	if m.querier == nil {
		return textResult("Device querier not available. The recon module may not be loaded."), nil, nil
	}

	summary, err := m.querier.GetHardwareSummary(context.Background())
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get fleet summary: %v", err)), nil, nil
	}
	if summary == nil {
		return textResult("No hardware summary available"), nil, nil
	}

	return textResult(writeToolJSON(summary)), nil, nil
}

func (m *Module) handleQueryDevices(_ context.Context, _ *sdkmcp.CallToolRequest, input queryDevicesInput) (*sdkmcp.CallToolResult, any, error) {
	m.publishToolCall("query_devices", input)

	if m.querier == nil {
		return textResult("Device querier not available. The recon module may not be loaded."), nil, nil
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	query := models.HardwareQuery{
		MinRAMMB:     input.MinRAMMB,
		MaxRAMMB:     input.MaxRAMMB,
		CPUModel:     input.CPUModel,
		OSName:       input.OSName,
		PlatformType: input.PlatformType,
		GPUVendor:    input.GPUVendor,
		Limit:        limit,
		Offset:       input.Offset,
	}

	devices, total, err := m.querier.QueryDevicesByHardware(context.Background(), query)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to query devices: %v", err)), nil, nil
	}

	resp := struct {
		Devices []models.Device `json:"devices"`
		Total   int             `json:"total"`
		Limit   int             `json:"limit"`
		Offset  int             `json:"offset"`
	}{
		Devices: devices,
		Total:   total,
		Limit:   limit,
		Offset:  input.Offset,
	}

	return textResult(writeToolJSON(resp)), nil, nil
}

func (m *Module) handleGetStaleDevices(_ context.Context, _ *sdkmcp.CallToolRequest, input getStaleDevicesInput) (*sdkmcp.CallToolResult, any, error) {
	m.publishToolCall("get_stale_devices", input)

	if m.querier == nil {
		return textResult("Device querier not available. The recon module may not be loaded."), nil, nil
	}

	hours := input.StaleAfterHours
	if hours <= 0 {
		hours = 24
	}

	threshold := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	devices, err := m.querier.FindStaleDevices(context.Background(), threshold)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to find stale devices: %v", err)), nil, nil
	}

	resp := struct {
		Devices         []models.Device `json:"devices"`
		Count           int             `json:"count"`
		StaleAfterHours int             `json:"stale_after_hours"`
	}{
		Devices:         devices,
		Count:           len(devices),
		StaleAfterHours: hours,
	}

	return textResult(writeToolJSON(resp)), nil, nil
}

func (m *Module) handleGetServiceInventory(_ context.Context, _ *sdkmcp.CallToolRequest, input getServiceInventoryInput) (*sdkmcp.CallToolResult, any, error) {
	m.publishToolCall("get_service_inventory", input)

	if m.serviceQuerier == nil {
		return textResult("Service data not available. The svcmap module may not be loaded."), nil, nil
	}

	services, err := m.serviceQuerier.ListServicesFiltered(context.Background(), input.DeviceID, input.ServiceType, input.Status)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list services: %v", err)), nil, nil
	}

	// Group services by device ID for readability.
	grouped := make(map[string][]models.Service)
	for i := range services {
		grouped[services[i].DeviceID] = append(grouped[services[i].DeviceID], services[i])
	}

	resp := struct {
		ServicesByDevice map[string][]models.Service `json:"services_by_device"`
		TotalServices    int                         `json:"total_services"`
	}{
		ServicesByDevice: grouped,
		TotalServices:    len(services),
	}

	return textResult(writeToolJSON(resp)), nil, nil
}

// textResult creates a successful CallToolResult with text content.
func textResult(text string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: text},
		},
	}
}

// errorResult creates an error CallToolResult with text content.
func errorResult(msg string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
