package recon

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// DeviceHardwareResponse is the composite response for GET /devices/{id}/hardware.
type DeviceHardwareResponse struct {
	Hardware *models.DeviceHardware `json:"hardware"`
	Storage  []models.DeviceStorage `json:"storage"`
	GPUs     []models.DeviceGPU     `json:"gpus"`
	Services []models.DeviceService `json:"services"`
}

// HardwareQueryResponse is the paginated response for GET /devices/query/hardware.
type HardwareQueryResponse struct {
	Devices []models.Device `json:"devices"`
	Total   int             `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}

// handleGetDeviceHardware returns the full hardware profile for a device.
//
//	@Summary		Get device hardware profile
//	@Description	Returns the full hardware profile including storage, GPUs, and services for a device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{object}	DeviceHardwareResponse
//	@Failure		400	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/devices/{id}/hardware [get]
func (m *Module) handleGetDeviceHardware(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	ctx := r.Context()

	hw, err := m.store.GetDeviceHardware(ctx, id)
	if err != nil {
		m.logger.Error("failed to get device hardware", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device hardware")
		return
	}

	storage, err := m.store.GetDeviceStorage(ctx, id)
	if err != nil {
		m.logger.Error("failed to get device storage", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device storage")
		return
	}

	gpus, err := m.store.GetDeviceGPU(ctx, id)
	if err != nil {
		m.logger.Error("failed to get device GPUs", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device GPUs")
		return
	}

	svcs, err := m.store.GetDeviceServices(ctx, id)
	if err != nil {
		m.logger.Error("failed to get device services", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device services")
		return
	}

	if storage == nil {
		storage = []models.DeviceStorage{}
	}
	if gpus == nil {
		gpus = []models.DeviceGPU{}
	}
	if svcs == nil {
		svcs = []models.DeviceService{}
	}

	writeJSON(w, http.StatusOK, DeviceHardwareResponse{
		Hardware: hw,
		Storage:  storage,
		GPUs:     gpus,
		Services: svcs,
	})
}

// handleUpdateDeviceHardware manually updates the hardware profile for a device.
//
//	@Summary		Update device hardware profile
//	@Description	Manually creates or updates a device's hardware profile. Sets collection_source to "manual".
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Device ID"
//	@Param			request	body		models.DeviceHardware	true	"Hardware profile"
//	@Success		200		{object}	models.DeviceHardware
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/devices/{id}/hardware [put]
func (m *Module) handleUpdateDeviceHardware(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	var hw models.DeviceHardware
	if err := json.NewDecoder(r.Body).Decode(&hw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	hw.DeviceID = id
	hw.CollectionSource = "manual"
	now := time.Now().UTC()
	hw.CollectedAt = &now

	if err := m.store.UpsertDeviceHardware(r.Context(), &hw); err != nil {
		m.logger.Error("failed to upsert device hardware", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to update device hardware")
		return
	}

	// Re-read the stored record to return the full state.
	stored, err := m.store.GetDeviceHardware(r.Context(), id)
	if err != nil {
		m.logger.Error("failed to read updated hardware", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to read updated hardware")
		return
	}
	writeJSON(w, http.StatusOK, stored)
}

// handleGetDeviceStorage returns storage devices for a device.
//
//	@Summary		Get device storage
//	@Description	Returns all storage devices attached to a device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{array}		models.DeviceStorage
//	@Failure		400	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/devices/{id}/storage [get]
func (m *Module) handleGetDeviceStorage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	storage, err := m.store.GetDeviceStorage(r.Context(), id)
	if err != nil {
		m.logger.Error("failed to get device storage", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device storage")
		return
	}
	if storage == nil {
		storage = []models.DeviceStorage{}
	}
	writeJSON(w, http.StatusOK, storage)
}

// handleGetDeviceGPU returns GPUs for a device.
//
//	@Summary		Get device GPUs
//	@Description	Returns all GPUs installed in a device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{array}		models.DeviceGPU
//	@Failure		400	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/devices/{id}/gpu [get]
func (m *Module) handleGetDeviceGPU(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	gpus, err := m.store.GetDeviceGPU(r.Context(), id)
	if err != nil {
		m.logger.Error("failed to get device GPUs", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device GPUs")
		return
	}
	if gpus == nil {
		gpus = []models.DeviceGPU{}
	}
	writeJSON(w, http.StatusOK, gpus)
}

// handleGetDeviceServices returns running services for a device.
//
//	@Summary		Get device services
//	@Description	Returns all running services on a device.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{array}		models.DeviceService
//	@Failure		400	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/devices/{id}/services [get]
func (m *Module) handleGetDeviceServices(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	svcs, err := m.store.GetDeviceServices(r.Context(), id)
	if err != nil {
		m.logger.Error("failed to get device services", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device services")
		return
	}
	if svcs == nil {
		svcs = []models.DeviceService{}
	}
	writeJSON(w, http.StatusOK, svcs)
}

// handleHardwareSummary returns fleet-wide aggregate hardware statistics.
//
//	@Summary		Hardware inventory summary
//	@Description	Returns aggregate hardware statistics across all devices with hardware profiles.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	models.HardwareSummary
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/inventory/hardware-summary [get]
func (m *Module) handleHardwareSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := m.store.GetHardwareSummary(r.Context())
	if err != nil {
		m.logger.Error("failed to get hardware summary", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get hardware summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleQueryDevicesByHardware filters devices by hardware specifications.
//
//	@Summary		Query devices by hardware
//	@Description	Returns devices matching hardware filters like RAM, CPU model, OS, platform type, and GPU.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			min_ram_mb		query		int		false	"Minimum RAM in MB"
//	@Param			max_ram_mb		query		int		false	"Maximum RAM in MB"
//	@Param			cpu_model		query		string	false	"CPU model substring match"
//	@Param			os_name			query		string	false	"OS name substring match"
//	@Param			platform_type	query		string	false	"Platform type (baremetal, vm, container, lxc)"
//	@Param			gpu_vendor		query		string	false	"GPU vendor filter (nvidia, amd, intel)"
//	@Param			has_gpu			query		bool	false	"Filter devices with/without GPU"
//	@Param			limit			query		int		false	"Max results"	default(50)
//	@Param			offset			query		int		false	"Offset"		default(0)
//	@Success		200				{object}	HardwareQueryResponse
//	@Failure		500				{object}	models.APIProblem
//	@Router			/recon/devices/query/hardware [get]
func (m *Module) handleQueryDevicesByHardware(w http.ResponseWriter, r *http.Request) {
	q := models.HardwareQuery{
		MinRAMMB:     queryInt(r, "min_ram_mb", 0),
		MaxRAMMB:     queryInt(r, "max_ram_mb", 0),
		CPUModel:     r.URL.Query().Get("cpu_model"),
		OSName:       r.URL.Query().Get("os_name"),
		PlatformType: r.URL.Query().Get("platform_type"),
		GPUVendor:    r.URL.Query().Get("gpu_vendor"),
		Limit:        queryInt(r, "limit", 50),
		Offset:       queryInt(r, "offset", 0),
	}

	// Parse has_gpu bool parameter.
	if hasGPUStr := r.URL.Query().Get("has_gpu"); hasGPUStr != "" {
		val, err := strconv.ParseBool(hasGPUStr)
		if err == nil {
			q.HasGPU = &val
		}
	}

	// Normalize gpu_vendor to lowercase.
	if q.GPUVendor != "" {
		q.GPUVendor = strings.ToLower(q.GPUVendor)
	}

	devices, total, err := m.store.QueryDevicesByHardware(r.Context(), q)
	if err != nil {
		m.logger.Error("failed to query devices by hardware", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to query devices by hardware")
		return
	}
	if devices == nil {
		devices = []models.Device{}
	}
	writeJSON(w, http.StatusOK, HardwareQueryResponse{
		Devices: devices,
		Total:   total,
		Limit:   q.Limit,
		Offset:  q.Offset,
	})
}
