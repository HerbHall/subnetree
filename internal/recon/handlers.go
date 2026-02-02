package recon

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/HerbHall/netvantage/pkg/models"
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
		"type":   "https://netvantage.io/problems/" + http.StatusText(status),
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
	ID           string               `json:"id"`
	Label        string               `json:"label"`
	DeviceType   models.DeviceType    `json:"device_type"`
	Status       models.DeviceStatus  `json:"status"`
	IPAddresses  []string             `json:"ip_addresses"`
	MACAddress   string               `json:"mac_address,omitempty"`
	Manufacturer string               `json:"manufacturer,omitempty"`
}

// TopologyEdge represents a link in the topology graph.
type TopologyEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	LinkType string `json:"link_type"`
	Speed    int    `json:"speed,omitempty"`
}

// handleScan triggers a new network scan.
// POST /scan
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
// GET /scans
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
// GET /scans/{id}
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

// handleTopology returns the network topology as a graph.
// GET /topology
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
