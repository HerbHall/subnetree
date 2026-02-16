package recon

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/roles"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an RFC 7807 problem detail response.
func writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

// TopologyGraph is the response for GET /topology.
type TopologyGraph struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// TopologyNode represents a device in the topology graph.
type TopologyNode struct {
	ID           string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Label        string              `json:"label" example:"web-server-01"`
	DeviceType   models.DeviceType   `json:"device_type" example:"server"`
	Status       models.DeviceStatus `json:"status" example:"online"`
	IPAddresses  []string            `json:"ip_addresses"`
	MACAddress   string              `json:"mac_address,omitempty" example:"00:1a:2b:3c:4d:5e"`
	Manufacturer string              `json:"manufacturer,omitempty" example:"Dell Inc."`
}

// TopologyEdge represents a link in the topology graph.
type TopologyEdge struct {
	ID       string `json:"id" example:"link-001"`
	Source   string `json:"source" example:"550e8400-e29b-41d4-a716-446655440000"`
	Target   string `json:"target" example:"660f9500-f30c-52e5-b827-557766551111"`
	LinkType string `json:"link_type" example:"ethernet"`
	Speed    int    `json:"speed,omitempty" example:"1000"`
}

// ScanRequest is the request body for POST /scan.
type ScanRequest struct {
	Subnet string `json:"subnet" example:"192.168.1.0/24"`
}

// handleScan triggers a new network scan.
//
//	@Summary		Start scan
//	@Description	Trigger a new network scan on the given subnet. Returns immediately with scan ID.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		ScanRequest			true	"Subnet to scan"
//	@Success		202		{object}	models.ScanResult	"Scan accepted"
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/scan [post]
func (m *Module) handleScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subnet string `json:"subnet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Subnet == "" {
		writeError(w, http.StatusBadRequest, "subnet is required")
		return
	}

	// Validate CIDR.
	_, ipNet, err := net.ParseCIDR(req.Subnet)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid CIDR: "+err.Error())
		return
	}

	// Reject subnets larger than /16.
	ones, bits := ipNet.Mask.Size()
	if bits-ones > 16 {
		writeError(w, http.StatusBadRequest, "subnet too large: maximum /16 allowed")
		return
	}

	// Create scan record.
	scanID := uuid.New().String()
	scan := &models.ScanResult{
		ID:     scanID,
		Subnet: req.Subnet,
		Status: "running",
	}
	if err := m.store.CreateScan(r.Context(), scan); err != nil {
		m.logger.Error("failed to create scan", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to create scan")
		return
	}

	// Store cancel func for this scan.
	scanCtx, cancel := m.newScanContext()
	m.activeScans.Store(scanID, cancel)
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()
		defer m.activeScans.Delete(scanID)
		m.orchestrator.RunScan(scanCtx, scanID, req.Subnet)
	}()

	writeJSON(w, http.StatusAccepted, scan)
}

// handleListScans returns a paginated list of scans.
//
//	@Summary		List scans
//	@Description	Returns a paginated list of scan results.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Max results"	default(50)
//	@Param			offset	query		int	false	"Offset"		default(0)
//	@Success		200		{array}		models.ScanResult
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/scans [get]
func (m *Module) handleListScans(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	scans, err := m.store.ListScans(r.Context(), limit, offset)
	if err != nil {
		m.logger.Error("failed to list scans", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list scans")
		return
	}
	if scans == nil {
		scans = []models.ScanResult{}
	}
	writeJSON(w, http.StatusOK, scans)
}

// handleGetScan returns a single scan with its discovered devices.
//
//	@Summary		Get scan
//	@Description	Returns a single scan result including discovered devices.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Scan ID"
//	@Success		200	{object}	models.ScanResult
//	@Failure		400	{object}	models.APIProblem
//	@Failure		404	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/scans/{id} [get]
func (m *Module) handleGetScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "scan ID is required")
		return
	}

	scan, err := m.store.GetScan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "scan not found")
		return
	}

	// Load devices for this scan.
	devices, _, err := m.store.ListDevices(r.Context(), ListDevicesOptions{ScanID: id})
	if err != nil {
		m.logger.Error("failed to list scan devices", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load scan devices")
		return
	}
	scan.Devices = devices

	writeJSON(w, http.StatusOK, scan)
}

// handleGetScanMetrics returns timing and count metrics for a single scan.
func (m *Module) handleGetScanMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "scan ID is required")
		return
	}

	metrics, err := m.store.GetScanMetrics(r.Context(), id)
	if err != nil {
		m.logger.Error("failed to get scan metrics", zap.String("scan_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get scan metrics")
		return
	}
	if metrics == nil {
		writeError(w, http.StatusNotFound, "scan metrics not found")
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}

// handleTopology returns the network topology as a graph.
//
//	@Summary		Get topology
//	@Description	Returns the network topology as a graph of nodes and edges.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	TopologyGraph
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/topology [get]
func (m *Module) handleTopology(w http.ResponseWriter, r *http.Request) {
	devices, _, err := m.store.ListDevices(r.Context(), ListDevicesOptions{Limit: 10000})
	if err != nil {
		m.logger.Error("failed to list devices for topology", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load devices")
		return
	}

	links, err := m.store.GetTopologyLinks(r.Context())
	if err != nil {
		m.logger.Error("failed to load topology links", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load topology links")
		return
	}

	graph := TopologyGraph{
		Nodes: make([]TopologyNode, 0, len(devices)),
		Edges: make([]TopologyEdge, 0, len(links)),
	}

	for i := range devices {
		d := &devices[i]
		label := d.Hostname
		if label == "" && len(d.IPAddresses) > 0 {
			label = d.IPAddresses[0]
		}
		graph.Nodes = append(graph.Nodes, TopologyNode{
			ID:           d.ID,
			Label:        label,
			DeviceType:   d.DeviceType,
			Status:       d.Status,
			IPAddresses:  d.IPAddresses,
			MACAddress:   d.MACAddress,
			Manufacturer: d.Manufacturer,
		})
	}

	for i := range links {
		l := &links[i]
		graph.Edges = append(graph.Edges, TopologyEdge{
			ID:       l.ID,
			Source:   l.SourceDeviceID,
			Target:   l.TargetDeviceID,
			LinkType: l.LinkType,
			Speed:    l.Speed,
		})
	}

	writeJSON(w, http.StatusOK, graph)
}

// handleListTopologyLayouts returns all saved topology layouts.
//
//	@Summary		List topology layouts
//	@Description	Returns all saved topology layout configurations.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		TopologyLayout
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/topology/layouts [get]
func (m *Module) handleListTopologyLayouts(w http.ResponseWriter, r *http.Request) {
	layouts, err := m.store.ListTopologyLayouts(r.Context())
	if err != nil {
		m.logger.Error("failed to list topology layouts", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list topology layouts")
		return
	}
	if layouts == nil {
		layouts = []TopologyLayout{}
	}
	writeJSON(w, http.StatusOK, layouts)
}

// handleCreateTopologyLayout creates a new saved topology layout.
//
//	@Summary		Create topology layout
//	@Description	Creates a new saved topology layout configuration.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		TopologyLayout	true	"Layout to create"
//	@Success		201		{object}	TopologyLayout
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/topology/layouts [post]
func (m *Module) handleCreateTopologyLayout(w http.ResponseWriter, r *http.Request) {
	var layout TopologyLayout
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if layout.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := m.store.CreateTopologyLayout(r.Context(), &layout); err != nil {
		m.logger.Error("failed to create topology layout", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to create topology layout")
		return
	}
	writeJSON(w, http.StatusCreated, layout)
}

// handleUpdateTopologyLayout updates an existing saved topology layout.
//
//	@Summary		Update topology layout
//	@Description	Updates an existing saved topology layout configuration.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string			true	"Layout ID"
//	@Param			body	body		TopologyLayout	true	"Layout to update"
//	@Success		200		{object}	TopologyLayout
//	@Failure		400		{object}	models.APIProblem
//	@Failure		404		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/topology/layouts/{id} [put]
func (m *Module) handleUpdateTopologyLayout(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var layout TopologyLayout
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	layout.ID = id
	if err := m.store.UpdateTopologyLayout(r.Context(), &layout); err != nil {
		m.logger.Error("failed to update topology layout", zap.Error(err))
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "topology layout not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update topology layout")
		return
	}
	writeJSON(w, http.StatusOK, layout)
}

// handleDeleteTopologyLayout deletes a saved topology layout.
//
//	@Summary		Delete topology layout
//	@Description	Deletes a saved topology layout configuration.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Layout ID"
//	@Success		204	"No content"
//	@Failure		404	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/topology/layouts/{id} [delete]
func (m *Module) handleDeleteTopologyLayout(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := m.store.DeleteTopologyLayout(r.Context(), id); err != nil {
		m.logger.Error("failed to delete topology layout", zap.Error(err))
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "topology layout not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete topology layout")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ImportResult contains the results of a CSV import operation.
type ImportResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// handleExportCSV exports all devices as a CSV file.
//
//	@Summary		Export devices as CSV
//	@Description	Downloads all devices in CSV format for spreadsheet import or backup.
//	@Tags			recon
//	@Produce		text/csv
//	@Security		BearerAuth
//	@Success		200	{file}	file
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/devices/export [get]
func (m *Module) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	devices, _, err := m.store.ListDevices(r.Context(), ListDevicesOptions{Limit: 100000})
	if err != nil {
		m.logger.Error("failed to list devices for export", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to export devices")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="subnetree-devices.csv"`)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	_ = writer.Write(csvHeaders())

	for i := range devices {
		_ = writer.Write(deviceToCSVRow(devices[i]))
	}
}

// handleImportCSV imports devices from a CSV file.
//
//	@Summary		Import devices from CSV
//	@Description	Uploads a CSV file to create or update devices. Duplicates detected by MAC address or hostname.
//	@Tags			recon
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"CSV file"
//	@Success		200		{object}	ImportResult
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/devices/import [post]
func (m *Module) handleImportCSV(w http.ResponseWriter, r *http.Request) {
	// Limit upload to 10 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing or invalid file field")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read and validate header.
	header, err := reader.Read()
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read CSV header")
		return
	}
	if len(header) < 2 {
		writeError(w, http.StatusBadRequest, "invalid CSV: too few columns")
		return
	}

	result := ImportResult{}
	rowNum := 1 // 1-indexed, header is row 1

	for {
		row, readErr := reader.Read()
		if readErr != nil {
			break // EOF or error
		}
		rowNum++

		device, parseErr := csvRowToDevice(row)
		if parseErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", rowNum, parseErr))
			result.Skipped++
			continue
		}

		if device.Hostname == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: hostname is required", rowNum))
			result.Skipped++
			continue
		}

		// Check for existing device by MAC.
		if device.MACAddress != "" {
			existing, _ := m.store.GetDeviceByMAC(r.Context(), device.MACAddress)
			if existing != nil {
				tags := device.Tags
				dtStr := string(device.DeviceType)
				_ = m.store.UpdateDevice(r.Context(), existing.ID, UpdateDeviceParams{
					Notes:       &device.Notes,
					Tags:        &tags,
					DeviceType:  &dtStr,
					Location:    &device.Location,
					Category:    &device.Category,
					PrimaryRole: &device.PrimaryRole,
					Owner:       &device.Owner,
				})
				result.Updated++
				continue
			}
		}

		// Check for existing device by hostname.
		existing, _ := m.store.GetDeviceByHostname(r.Context(), device.Hostname)
		if existing != nil {
			tags := device.Tags
			dtStr := string(device.DeviceType)
			_ = m.store.UpdateDevice(r.Context(), existing.ID, UpdateDeviceParams{
				Notes:       &device.Notes,
				Tags:        &tags,
				DeviceType:  &dtStr,
				Location:    &device.Location,
				Category:    &device.Category,
				PrimaryRole: &device.PrimaryRole,
				Owner:       &device.Owner,
			})
			result.Updated++
			continue
		}

		// Create new device.
		if device.ID == "" {
			device.ID = uuid.New().String()
		}
		if device.Status == "" {
			device.Status = models.DeviceStatusUnknown
		}
		if device.DiscoveryMethod == "" {
			device.DiscoveryMethod = models.DiscoveryManual
		}

		if createErr := m.store.InsertManualDevice(r.Context(), &device); createErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", rowNum, createErr))
			result.Skipped++
			continue
		}
		result.Created++
	}

	writeJSON(w, http.StatusOK, result)
}

// DeviceListResponse is the paginated response for GET /devices.
type DeviceListResponse struct {
	Devices []models.Device `json:"devices"`
	Total   int             `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}

// DeviceStatusEvent is the frontend-compatible status history entry.
type DeviceStatusEvent struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// DeviceScanEvent is the frontend-compatible scan history entry.
type DeviceScanEvent struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	TargetCIDR   string `json:"target_cidr"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	DevicesFound int    `json:"devices_found"`
}

// CreateDeviceRequest is the request body for POST /devices.
type CreateDeviceRequest struct {
	Hostname    string            `json:"hostname"`
	IPAddresses []string          `json:"ip_addresses,omitempty"`
	MACAddress  string            `json:"mac_address,omitempty"`
	DeviceType  string            `json:"device_type,omitempty"`
	Notes       string            `json:"notes,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// handleListDevices returns a paginated list of devices with optional filters.
//
//	@Summary		List devices
//	@Description	Returns a paginated list of devices with optional status, type, category, and owner filters.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit		query		int		false	"Max results"			default(50)
//	@Param			offset		query		int		false	"Offset"				default(0)
//	@Param			status		query		string	false	"Filter by status"
//	@Param			type		query		string	false	"Filter by device type"
//	@Param			category	query		string	false	"Filter by category"
//	@Param			owner		query		string	false	"Filter by owner"
//	@Success		200			{object}	DeviceListResponse
//	@Failure		500			{object}	models.APIProblem
//	@Router			/recon/devices [get]
func (m *Module) handleListDevices(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)
	status := r.URL.Query().Get("status")
	deviceType := r.URL.Query().Get("type")
	category := r.URL.Query().Get("category")
	owner := r.URL.Query().Get("owner")

	devices, total, err := m.store.ListDevices(r.Context(), ListDevicesOptions{
		Limit:      limit,
		Offset:     offset,
		Status:     status,
		DeviceType: deviceType,
		Category:   category,
		Owner:      owner,
	})
	if err != nil {
		m.logger.Error("failed to list devices", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}
	if devices == nil {
		devices = []models.Device{}
	}
	writeJSON(w, http.StatusOK, DeviceListResponse{
		Devices: devices,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// handleGetDevice returns a single device by ID.
//
//	@Summary		Get device
//	@Description	Returns a single device by its ID.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{object}	models.Device
//	@Failure		400	{object}	models.APIProblem
//	@Failure		404	{object}	models.APIProblem
//	@Router			/recon/devices/{id} [get]
func (m *Module) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	device, err := m.store.GetDevice(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	writeJSON(w, http.StatusOK, device)
}

// handleUpdateDevice applies a partial update to a device.
//
//	@Summary		Update device
//	@Description	Partially updates a device's notes, tags, custom fields, or device type.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Device ID"
//	@Param			request	body		UpdateDeviceParams	true	"Fields to update"
//	@Success		200		{object}	models.Device
//	@Failure		400		{object}	models.APIProblem
//	@Failure		404		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/devices/{id} [put]
func (m *Module) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	var params UpdateDeviceParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := m.store.UpdateDevice(r.Context(), id, params); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		m.logger.Error("failed to update device", zap.String("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to update device")
		return
	}

	// Return the updated device.
	device, err := m.store.GetDevice(r.Context(), id)
	if err != nil {
		m.logger.Error("failed to read updated device", zap.String("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to read updated device")
		return
	}
	writeJSON(w, http.StatusOK, device)
}

// handleDeleteDevice removes a device by ID.
//
//	@Summary		Delete device
//	@Description	Permanently removes a device and its history.
//	@Tags			recon
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Device ID"
//	@Success		204	"No content"
//	@Failure		400	{object}	models.APIProblem
//	@Failure		404	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/devices/{id} [delete]
func (m *Module) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	if err := m.store.DeleteDevice(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "device not found")
			return
		}
		m.logger.Error("failed to delete device", zap.String("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to delete device")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCreateDevice manually creates a new device.
//
//	@Summary		Create device
//	@Description	Manually creates a new device record.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateDeviceRequest	true	"Device to create"
//	@Success		201		{object}	models.Device
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/devices [post]
func (m *Module) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Hostname == "" {
		writeError(w, http.StatusBadRequest, "hostname is required")
		return
	}

	dt := models.DeviceTypeUnknown
	if req.DeviceType != "" {
		dt = models.DeviceType(req.DeviceType)
	}

	device := &models.Device{
		Hostname:    req.Hostname,
		IPAddresses: req.IPAddresses,
		MACAddress:  req.MACAddress,
		DeviceType:  dt,
		Notes:       req.Notes,
		Tags:        req.Tags,
	}

	if err := m.store.InsertManualDevice(r.Context(), device); err != nil {
		m.logger.Error("failed to create device", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to create device")
		return
	}
	writeJSON(w, http.StatusCreated, device)
}

// handleDeviceHistory returns status change history for a device.
//
//	@Summary		Device status history
//	@Description	Returns the status change timeline for a device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Device ID"
//	@Param			limit	query		int		false	"Max results"	default(50)
//	@Success		200		{array}		DeviceStatusEvent
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/devices/{id}/history [get]
func (m *Module) handleDeviceHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	limit := queryInt(r, "limit", 50)

	changes, _, err := m.store.GetDeviceHistory(r.Context(), id, limit, 0)
	if err != nil {
		m.logger.Error("failed to get device history", zap.String("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device history")
		return
	}

	// Map to frontend-compatible format.
	events := make([]DeviceStatusEvent, 0, len(changes))
	for i := range changes {
		events = append(events, DeviceStatusEvent{
			ID:        changes[i].ID,
			DeviceID:  changes[i].DeviceID,
			Status:    changes[i].NewStatus,
			Timestamp: changes[i].ChangedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, events)
}

// handleDeviceScans returns scans that discovered or updated a device.
//
//	@Summary		Device scan history
//	@Description	Returns scans that discovered or updated the specified device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Device ID"
//	@Param			limit	query		int		false	"Max results"	default(50)
//	@Success		200		{array}		DeviceScanEvent
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/devices/{id}/scans [get]
func (m *Module) handleDeviceScans(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	limit := queryInt(r, "limit", 50)

	scans, _, err := m.store.GetDeviceScans(r.Context(), id, limit, 0)
	if err != nil {
		m.logger.Error("failed to get device scans", zap.String("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device scans")
		return
	}

	// Map to frontend-compatible format.
	events := make([]DeviceScanEvent, 0, len(scans))
	for i := range scans {
		events = append(events, DeviceScanEvent{
			ID:           scans[i].ID,
			Status:       scans[i].Status,
			TargetCIDR:   scans[i].Subnet,
			StartedAt:    scans[i].StartedAt,
			DevicesFound: scans[i].Total,
		})
	}
	writeJSON(w, http.StatusOK, events)
}

// BulkUpdateRequest is the request body for PATCH /devices/bulk.
type BulkUpdateRequest struct {
	DeviceIDs []string           `json:"device_ids"`
	Updates   UpdateDeviceParams `json:"updates"`
}

// BulkUpdateResponse is the response body for PATCH /devices/bulk.
type BulkUpdateResponse struct {
	Updated int `json:"updated"`
}

// handleInventorySummary returns aggregate inventory statistics.
//
//	@Summary		Inventory summary
//	@Description	Returns aggregate device inventory statistics including counts by status, category, and type.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			stale_days	query		int	false	"Days since last seen to consider stale"	default(30)
//	@Success		200			{object}	InventorySummary
//	@Failure		500			{object}	models.APIProblem
//	@Router			/recon/inventory/summary [get]
func (m *Module) handleInventorySummary(w http.ResponseWriter, r *http.Request) {
	staleDays := queryInt(r, "stale_days", 30)

	summary, err := m.store.GetInventorySummary(r.Context(), staleDays)
	if err != nil {
		m.logger.Error("failed to get inventory summary", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get inventory summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleBulkUpdateDevices applies the same update to multiple devices.
//
//	@Summary		Bulk update devices
//	@Description	Applies the same partial update to multiple devices by ID.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		BulkUpdateRequest	true	"Device IDs and fields to update"
//	@Success		200		{object}	BulkUpdateResponse
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/devices/bulk [patch]
func (m *Module) handleBulkUpdateDevices(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.DeviceIDs) == 0 {
		writeError(w, http.StatusBadRequest, "device_ids is required")
		return
	}

	updated, err := m.store.BulkUpdateDevices(r.Context(), req.DeviceIDs, req.Updates)
	if err != nil {
		m.logger.Error("failed to bulk update devices", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to bulk update devices")
		return
	}
	writeJSON(w, http.StatusOK, BulkUpdateResponse{Updated: updated})
}

// queryInt extracts an integer query parameter with a default value.
func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}

// ============================================================================
// SNMP Handlers
// ============================================================================

// SNMPDiscoverRequest is the request body for POST /snmp/discover.
type SNMPDiscoverRequest struct {
	Target       string `json:"target" example:"192.168.1.1"`
	CredentialID string `json:"credential_id" example:"cred-snmp-001"`
}

// SNMPSystemInfoResponse wraps SNMPSystemInfo for JSON serialization.
type SNMPSystemInfoResponse struct {
	Description string `json:"description"`
	ObjectID    string `json:"object_id"`
	UpTimeMs    int64  `json:"up_time_ms"`
	Contact     string `json:"contact"`
	Name        string `json:"name"`
	Location    string `json:"location"`
}

// SNMPInterfaceResponse wraps SNMPInterface for JSON serialization.
type SNMPInterfaceResponse struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Type        int    `json:"type"`
	MTU         int    `json:"mtu"`
	Speed       uint64 `json:"speed"`
	PhysAddress string `json:"phys_address"`
	AdminStatus int    `json:"admin_status"`
	OperStatus  int    `json:"oper_status"`
}

// handleListServiceMovements returns recent service movements.
//
//	@Summary		List service movements
//	@Description	Returns recent service movements where a port migrated from one device to another.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Max results"	default(50)
//	@Success		200		{array}		ServiceMovement
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/movements [get]
func (m *Module) handleListServiceMovements(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)

	movements, err := m.store.ListServiceMovements(r.Context(), limit)
	if err != nil {
		m.logger.Error("failed to list service movements", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list service movements")
		return
	}
	if movements == nil {
		movements = []ServiceMovement{}
	}
	writeJSON(w, http.StatusOK, movements)
}

// handleSNMPDiscover triggers SNMP discovery for a specific target.
//
//	@Summary		SNMP discover
//	@Description	Discover a device via SNMP using the given credentials.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		SNMPDiscoverRequest	true	"SNMP target and credentials"
//	@Success		200		{array}		models.Device		"Discovered devices"
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/snmp/discover [post]
func (m *Module) handleSNMPDiscover(w http.ResponseWriter, r *http.Request) {
	var req SNMPDiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if req.CredentialID == "" {
		writeError(w, http.StatusBadRequest, "credential_id is required")
		return
	}
	if m.snmpCollector == nil {
		writeError(w, http.StatusServiceUnavailable, "SNMP collector not available")
		return
	}
	if m.credAccessor == nil {
		writeError(w, http.StatusServiceUnavailable, "credential accessor not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	discovered, err := m.snmpCollector.Discover(ctx, req.Target, m.credAccessor, req.CredentialID)
	if err != nil {
		m.logger.Error("SNMP discovery failed",
			zap.String("target", req.Target),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "SNMP discovery failed: "+err.Error())
		return
	}

	// Upsert each discovered device to the store.
	devices := make([]models.Device, 0, len(discovered))
	for i := range discovered {
		if _, upsertErr := m.store.UpsertDevice(ctx, &discovered[i]); upsertErr != nil {
			m.logger.Error("failed to upsert SNMP-discovered device",
				zap.String("hostname", discovered[i].Hostname),
				zap.Error(upsertErr),
			)
			continue
		}
		devices = append(devices, discovered[i])
	}

	writeJSON(w, http.StatusOK, devices)
}

// handleSNMPSystemInfo retrieves SNMP system information for a device.
//
//	@Summary		SNMP system info
//	@Description	Query SNMP system information from a device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id	path		string					true	"Device ID"
//	@Success		200			{object}	SNMPSystemInfoResponse	"System information"
//	@Failure		400		{object}	models.APIProblem
//	@Failure		404		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/snmp/system/{device_id} [get]
func (m *Module) handleSNMPSystemInfo(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	device, err := m.store.GetDevice(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	credID, err := m.findSNMPCredential(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "no SNMP credentials configured for device")
		return
	}

	if len(device.IPAddresses) == 0 {
		writeError(w, http.StatusBadRequest, "device has no IP addresses")
		return
	}
	ip := device.IPAddresses[0]

	if m.snmpCollector == nil {
		writeError(w, http.StatusServiceUnavailable, "SNMP collector not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := m.snmpCollector.GetSystemInfo(ctx, ip, m.credAccessor, credID)
	if err != nil {
		m.logger.Error("SNMP system info query failed",
			zap.String("device_id", deviceID),
			zap.String("ip", ip),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "SNMP system info query failed: "+err.Error())
		return
	}

	resp := SNMPSystemInfoResponse{
		Description: info.Description,
		ObjectID:    info.ObjectID,
		UpTimeMs:    info.UpTime.Milliseconds(),
		Contact:     info.Contact,
		Name:        info.Name,
		Location:    info.Location,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleSNMPInterfaces retrieves SNMP interface table for a device.
//
//	@Summary		SNMP interfaces
//	@Description	Query SNMP interface table from a device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id	path		string					true	"Device ID"
//	@Success		200			{array}		SNMPInterfaceResponse	"Interface list"
//	@Failure		400		{object}	models.APIProblem
//	@Failure		404		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/snmp/interfaces/{device_id} [get]
func (m *Module) handleSNMPInterfaces(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	device, err := m.store.GetDevice(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	credID, err := m.findSNMPCredential(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "no SNMP credentials configured for device")
		return
	}

	if len(device.IPAddresses) == 0 {
		writeError(w, http.StatusBadRequest, "device has no IP addresses")
		return
	}
	ip := device.IPAddresses[0]

	if m.snmpCollector == nil {
		writeError(w, http.StatusServiceUnavailable, "SNMP collector not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	interfaces, err := m.snmpCollector.GetInterfaces(ctx, ip, m.credAccessor, credID)
	if err != nil {
		m.logger.Error("SNMP interfaces query failed",
			zap.String("device_id", deviceID),
			zap.String("ip", ip),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "SNMP interfaces query failed: "+err.Error())
		return
	}

	resp := make([]SNMPInterfaceResponse, len(interfaces))
	for i := range interfaces {
		resp[i] = SNMPInterfaceResponse{
			Index:       interfaces[i].Index,
			Description: interfaces[i].Description,
			Type:        interfaces[i].Type,
			MTU:         interfaces[i].MTU,
			Speed:       interfaces[i].Speed,
			PhysAddress: interfaces[i].PhysAddress,
			AdminStatus: interfaces[i].AdminStatus,
			OperStatus:  interfaces[i].OperStatus,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// findSNMPCredential looks up the first SNMP credential associated with a device.
func (m *Module) findSNMPCredential(ctx context.Context, deviceID string) (string, error) {
	if m.credProvider == nil {
		return "", errors.New("credential provider not configured")
	}

	creds, err := m.credProvider.CredentialsForDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}

	for i := range creds {
		if creds[i].Type == "snmp_v2c" || creds[i].Type == "snmp_v3" {
			return creds[i].ID, nil
		}
	}
	return "", errors.New("no SNMP credentials found for device")
}

// SetCredentialProvider sets the credential provider for SNMP device lookups.
// Called from the composition root after all plugins are initialized.
func (m *Module) SetCredentialProvider(cp roles.CredentialProvider) {
	m.credProvider = cp
}
