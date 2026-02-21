package testutil

import (
	"time"

	"github.com/google/uuid"

	"github.com/HerbHall/subnetree/pkg/models"
)

// NewDevice returns a Device with sensible defaults, suitable for test fixtures.
// Override individual fields after creation as needed.
func NewDevice(opts ...func(*models.Device)) models.Device {
	d := models.Device{
		ID:              uuid.New().String(),
		Hostname:        "test-device",
		IPAddresses:     []string{"192.168.1.100"},
		MACAddress:      "00:11:22:33:44:55",
		DeviceType:      models.DeviceTypeDesktop,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(&d)
	}
	return d
}

// WithHostname sets the device hostname.
func WithHostname(name string) func(*models.Device) {
	return func(d *models.Device) { d.Hostname = name }
}

// WithIP sets the device's IP address list.
func WithIP(ips ...string) func(*models.Device) {
	return func(d *models.Device) { d.IPAddresses = ips }
}

// WithMAC sets the device's MAC address.
func WithMAC(mac string) func(*models.Device) {
	return func(d *models.Device) { d.MACAddress = mac }
}

// WithStatus sets the device status.
func WithStatus(s models.DeviceStatus) func(*models.Device) {
	return func(d *models.Device) { d.Status = s }
}

// WithLastSeen sets the device's last_seen timestamp.
func WithLastSeen(t time.Time) func(*models.Device) {
	return func(d *models.Device) { d.LastSeen = t }
}

// WithDeviceType sets the device type.
func WithDeviceType(dt models.DeviceType) func(*models.Device) {
	return func(d *models.Device) { d.DeviceType = dt }
}

// NewDeviceHardware returns a DeviceHardware with sensible defaults.
func NewDeviceHardware(deviceID string, opts ...func(*models.DeviceHardware)) models.DeviceHardware {
	hw := models.DeviceHardware{
		DeviceID:           deviceID,
		Hostname:           "test-device",
		OSName:             "Ubuntu 24.04",
		OSVersion:          "24.04",
		OSArch:             "amd64",
		CPUModel:           "Intel Core i7-12700K",
		CPUCores:           12,
		CPUThreads:         20,
		RAMTotalMB:         32768,
		PlatformType:       "baremetal",
		SystemManufacturer: "Dell Inc.",
		CollectionSource:   "scout-linux",
	}
	for _, opt := range opts {
		opt(&hw)
	}
	return hw
}

// WithCPU sets the CPU model, cores, and threads on a DeviceHardware.
func WithCPU(model string, cores, threads int) func(*models.DeviceHardware) {
	return func(hw *models.DeviceHardware) {
		hw.CPUModel = model
		hw.CPUCores = cores
		hw.CPUThreads = threads
	}
}

// WithRAM sets the total RAM in MB on a DeviceHardware.
func WithRAM(totalMB int) func(*models.DeviceHardware) {
	return func(hw *models.DeviceHardware) { hw.RAMTotalMB = totalMB }
}

// WithOSInfo sets the OS name and version on a DeviceHardware.
func WithOSInfo(name, version string) func(*models.DeviceHardware) {
	return func(hw *models.DeviceHardware) {
		hw.OSName = name
		hw.OSVersion = version
	}
}

// WithPlatformType sets the platform type on a DeviceHardware.
func WithPlatformType(pt string) func(*models.DeviceHardware) {
	return func(hw *models.DeviceHardware) { hw.PlatformType = pt }
}

// WithCollectionSource sets the collection source on a DeviceHardware.
func WithCollectionSource(src string) func(*models.DeviceHardware) {
	return func(hw *models.DeviceHardware) { hw.CollectionSource = src }
}
