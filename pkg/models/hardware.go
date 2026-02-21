package models

import "time"

// DeviceHardware represents the hardware profile of a device.
type DeviceHardware struct {
	DeviceID           string     `json:"device_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Hostname           string     `json:"hostname,omitempty" example:"web-server-01"`
	FQDN               string     `json:"fqdn,omitempty" example:"web-server-01.local"`
	OSName             string     `json:"os_name,omitempty" example:"Ubuntu 24.04"`
	OSVersion          string     `json:"os_version,omitempty" example:"24.04"`
	OSArch             string     `json:"os_arch,omitempty" example:"amd64"`
	Kernel             string     `json:"kernel,omitempty" example:"6.5.0-44-generic"`
	CPUModel           string     `json:"cpu_model,omitempty" example:"Intel Core i9-10900K"`
	CPUCores           int        `json:"cpu_cores,omitempty" example:"10"`
	CPUThreads         int        `json:"cpu_threads,omitempty" example:"20"`
	CPUArch            string     `json:"cpu_arch,omitempty" example:"x86_64"`
	RAMTotalMB         int        `json:"ram_total_mb,omitempty" example:"32768"`
	RAMType            string     `json:"ram_type,omitempty" example:"DDR4"`
	RAMSlotsUsed       int        `json:"ram_slots_used,omitempty" example:"2"`
	RAMSlotsTotal      int        `json:"ram_slots_total,omitempty" example:"4"`
	PlatformType       string     `json:"platform_type,omitempty" example:"baremetal"`
	Hypervisor         string     `json:"hypervisor,omitempty" example:"proxmox"`
	VMHostID           string     `json:"vm_host_id,omitempty"`
	SystemManufacturer string     `json:"system_manufacturer,omitempty" example:"Dell Inc."`
	SystemModel        string     `json:"system_model,omitempty" example:"PowerEdge R730"`
	SerialNumber       string     `json:"serial_number,omitempty" example:"ABC123"`
	BIOSVersion        string     `json:"bios_version,omitempty" example:"2.17.0"`
	CollectionSource   string     `json:"collection_source,omitempty" example:"scout-wmi"`
	CollectedAt        *time.Time `json:"collected_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

// DeviceStorage represents a storage device attached to a device.
type DeviceStorage struct {
	ID               string     `json:"id"`
	DeviceID         string     `json:"device_id"`
	Name             string     `json:"name,omitempty" example:"Samsung 990 Pro 4TB"`
	DiskType         string     `json:"disk_type,omitempty" example:"nvme"`
	Interface        string     `json:"interface,omitempty" example:"pcie4"`
	CapacityGB       int        `json:"capacity_gb,omitempty" example:"4000"`
	Model            string     `json:"model,omitempty" example:"Samsung SSD 990 PRO"`
	Role             string     `json:"role,omitempty" example:"data"`
	CollectionSource string     `json:"collection_source,omitempty" example:"scout-linux"`
	CollectedAt      *time.Time `json:"collected_at,omitempty"`
}

// DeviceGPU represents a GPU in a device.
type DeviceGPU struct {
	ID               string     `json:"id"`
	DeviceID         string     `json:"device_id"`
	Model            string     `json:"model,omitempty" example:"NVIDIA RTX 3090 Ti"`
	Vendor           string     `json:"vendor,omitempty" example:"nvidia"`
	VRAMMB           int        `json:"vram_mb,omitempty" example:"24576"`
	DriverVersion    string     `json:"driver_version,omitempty" example:"535.183.01"`
	CollectionSource string     `json:"collection_source,omitempty" example:"scout-linux"`
	CollectedAt      *time.Time `json:"collected_at,omitempty"`
}

// DeviceService represents a running service on a device.
type DeviceService struct {
	ID               string     `json:"id"`
	DeviceID         string     `json:"device_id"`
	Name             string     `json:"name,omitempty" example:"plex"`
	ServiceType      string     `json:"service_type,omitempty" example:"docker"`
	Port             int        `json:"port,omitempty" example:"32400"`
	URL              string     `json:"url,omitempty" example:"http://192.168.1.10:32400"`
	Version          string     `json:"version,omitempty" example:"1.40.0"`
	Status           string     `json:"status,omitempty" example:"running"`
	CollectionSource string     `json:"collection_source,omitempty" example:"scout-linux"`
	CollectedAt      *time.Time `json:"collected_at,omitempty"`
}

// HardwareSummary provides fleet-wide aggregate hardware statistics.
type HardwareSummary struct {
	TotalWithHardware int            `json:"total_with_hardware"`
	TotalRAMMB        int64          `json:"total_ram_mb"`
	TotalStorageGB    int64          `json:"total_storage_gb"`
	TotalGPUs         int            `json:"total_gpus"`
	ByOS              map[string]int `json:"by_os"`
	ByCPUModel        map[string]int `json:"by_cpu_model"`
	ByPlatformType    map[string]int `json:"by_platform_type"`
	ByGPUVendor       map[string]int `json:"by_gpu_vendor"`
}

// HardwareQuery defines filters for hardware-based device queries.
type HardwareQuery struct {
	MinRAMMB     int    `json:"min_ram_mb,omitempty"`
	MaxRAMMB     int    `json:"max_ram_mb,omitempty"`
	CPUModel     string `json:"cpu_model,omitempty"`
	OSName       string `json:"os_name,omitempty"`
	PlatformType string `json:"platform_type,omitempty"`
	GPUVendor    string `json:"gpu_vendor,omitempty"`
	HasGPU       *bool  `json:"has_gpu,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}
