package models

import "time"

// ServiceType categorizes a tracked service.
type ServiceType string

const (
	ServiceTypeDockerContainer ServiceType = "docker-container"
	ServiceTypeSystemdService  ServiceType = "systemd-service"
	ServiceTypeWindowsService  ServiceType = "windows-service"
	ServiceTypeApplication     ServiceType = "application"
)

// ServiceStatus represents the current state of a service.
type ServiceStatus string

const (
	ServiceStatusRunning ServiceStatus = "running"
	ServiceStatusStopped ServiceStatus = "stopped"
	ServiceStatusFailed  ServiceStatus = "failed"
	ServiceStatusUnknown ServiceStatus = "unknown"
)

// DesiredState represents the intended operational state of a service.
type DesiredState string

const (
	DesiredStateShouldRun      DesiredState = "should-run"
	DesiredStateShouldStop     DesiredState = "should-stop"
	DesiredStateMonitoringOnly DesiredState = "monitoring-only"
)

// Service represents a tracked service on a device.
type Service struct {
	ID            string        `json:"id" example:"svc-550e8400-e29b-41d4-a716-446655440000"`
	Name          string        `json:"name" example:"nginx"`
	DisplayName   string        `json:"display_name" example:"NGINX Web Server"`
	ServiceType   ServiceType   `json:"service_type" example:"docker-container"`
	DeviceID      string        `json:"device_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ApplicationID string        `json:"application_id,omitempty" example:"app-001"`
	Status        ServiceStatus `json:"status" example:"running"`
	DesiredState  DesiredState  `json:"desired_state" example:"should-run"`
	Ports         []string      `json:"ports,omitempty"`
	CPUPercent    float64       `json:"cpu_percent" example:"12.5"`
	MemoryBytes   int64         `json:"memory_bytes" example:"134217728"`
	FirstSeen     time.Time     `json:"first_seen" example:"2026-01-10T08:00:00Z"`
	LastSeen      time.Time     `json:"last_seen" example:"2026-02-13T10:30:00Z"`
}

// UtilizationSummary provides resource usage and grading for a single device.
type UtilizationSummary struct {
	DeviceID      string  `json:"device_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Hostname      string  `json:"hostname" example:"web-server-01"`
	CPUPercent    float64 `json:"cpu_percent" example:"45.2"`
	MemoryPercent float64 `json:"memory_percent" example:"62.8"`
	DiskPercent   float64 `json:"disk_percent" example:"38.1"`
	ServiceCount  int     `json:"service_count" example:"12"`
	Grade         string  `json:"grade" example:"B"`
	Headroom      float64 `json:"headroom" example:"37.2"`
}

// FleetSummary aggregates utilization across all devices.
type FleetSummary struct {
	TotalDevices  int            `json:"total_devices" example:"8"`
	TotalServices int            `json:"total_services" example:"42"`
	AvgCPU        float64        `json:"avg_cpu" example:"35.4"`
	AvgMemory     float64        `json:"avg_memory" example:"52.1"`
	ByGrade       map[string]int `json:"by_grade"`
	Underutilized []string       `json:"underutilized,omitempty"`
	Overloaded    []string       `json:"overloaded,omitempty"`
}
