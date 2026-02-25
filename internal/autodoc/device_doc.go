package autodoc

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

// DeviceDocData contains all data needed for per-device documentation rendering.
type DeviceDocData struct {
	Device        *models.Device
	Hardware      *models.DeviceHardware
	Storage       []models.DeviceStorage
	GPUs          []models.DeviceGPU
	Services      []models.DeviceService
	Children      []models.Device
	Alerts        []DeviceAlert
	RecentChanges []ChangelogEntry
	GeneratedAt   time.Time
}

// RenderDeviceDoc renders a per-device Markdown document from the given data.
func RenderDeviceDoc(data DeviceDocData) (doc string, err error) {
	funcMap := template.FuncMap{
		"formatTime":        formatTime,
		"humanizeBytes":     humanizeBytes,
		"networkLayerLabel": networkLayerLabel,
		"deviceTypeLabel":   deviceTypeLabel,
		"primaryIP":         primaryIP,
		"eventIcon":         eventIcon,
		"sourceTag":         sourceTag,
		"derefTime": func(t *time.Time) time.Time {
			if t == nil {
				return time.Time{}
			}
			return *t
		},
	}

	tmpl, parseErr := template.New("device_doc").Funcs(funcMap).Parse(defaultDeviceTemplate)
	if parseErr != nil {
		return "", fmt.Errorf("parse device doc template: %w", parseErr)
	}

	var b strings.Builder
	if execErr := tmpl.Execute(&b, data); execErr != nil {
		return "", fmt.Errorf("execute device doc template: %w", execErr)
	}

	return b.String(), nil
}

// formatTime formats a time.Time to a human-readable string.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}

// humanizeBytes converts bytes (as int) to a human-readable string.
func humanizeBytes(megabytes int) string {
	if megabytes <= 0 {
		return "N/A"
	}
	if megabytes >= 1024 {
		gb := float64(megabytes) / 1024.0
		return fmt.Sprintf("%.1f GB", gb)
	}
	return fmt.Sprintf("%d MB", megabytes)
}

// networkLayerLabel returns a human-readable label for a network layer number.
func networkLayerLabel(layer int) string {
	switch layer {
	case 1:
		return "Gateway (Layer 1)"
	case 2:
		return "Distribution (Layer 2)"
	case 3:
		return "Access (Layer 3)"
	case 4:
		return "Endpoint (Layer 4)"
	default:
		return "Unknown"
	}
}

// deviceTypeLabel returns a human-readable label for a DeviceType.
func deviceTypeLabel(dt models.DeviceType) string {
	switch dt {
	case models.DeviceTypeServer:
		return "Server"
	case models.DeviceTypeDesktop:
		return "Desktop"
	case models.DeviceTypeLaptop:
		return "Laptop"
	case models.DeviceTypeMobile:
		return "Mobile"
	case models.DeviceTypeRouter:
		return "Router"
	case models.DeviceTypeSwitch:
		return "Switch"
	case models.DeviceTypePrinter:
		return "Printer"
	case models.DeviceTypeIoT:
		return "IoT"
	case models.DeviceTypeAccessPoint:
		return "Access Point"
	case models.DeviceTypeFirewall:
		return "Firewall"
	case models.DeviceTypeNAS:
		return "NAS"
	case models.DeviceTypePhone:
		return "Phone"
	case models.DeviceTypeTablet:
		return "Tablet"
	case models.DeviceTypeCamera:
		return "Camera"
	case models.DeviceTypeVM:
		return "Virtual Machine"
	case models.DeviceTypeContainer:
		return "Container"
	case models.DeviceTypeUnknown:
		return "Unknown"
	}
	return string(dt)
}

// primaryIP returns the first IP address from a slice of IPs, or "N/A".
func primaryIP(ips []string) string {
	if len(ips) == 0 {
		return "N/A"
	}
	return ips[0]
}

const defaultDeviceTemplate = `# {{ .Device.Hostname }}{{ if .Device.IPAddresses }} ({{ primaryIP .Device.IPAddresses }}){{ end }}

**Device Type:** {{ deviceTypeLabel .Device.DeviceType }} | **Status:** {{ .Device.Status }} | **Confidence:** {{ .Device.ClassificationConfidence }}%
**First Seen:** {{ formatTime .Device.FirstSeen }} | **Last Seen:** {{ formatTime .Device.LastSeen }}
**MAC Address:** {{ if .Device.MACAddress }}{{ .Device.MACAddress }}{{ else }}N/A{{ end }} | **Manufacturer:** {{ if .Device.Manufacturer }}{{ .Device.Manufacturer }}{{ else }}N/A{{ end }}
{{ if .Hardware }}
## Hardware Profile

| Property | Value |
|----------|-------|
| **OS** | {{ .Hardware.OSName }} {{ .Hardware.OSVersion }} ({{ .Hardware.OSArch }}) |
| **CPU** | {{ .Hardware.CPUModel }} ({{ .Hardware.CPUCores }} cores, {{ .Hardware.CPUThreads }} threads) |
| **RAM** | {{ humanizeBytes .Hardware.RAMTotalMB }} |
| **Platform** | {{ if .Hardware.PlatformType }}{{ .Hardware.PlatformType }}{{ else }}Physical{{ end }} |
| **System** | {{ if .Hardware.SystemManufacturer }}{{ .Hardware.SystemManufacturer }} {{ .Hardware.SystemModel }}{{ else }}N/A{{ end }} |
{{ if .Storage }}
### Storage

| Name | Type | Capacity | Interface | Model |
|------|------|----------|-----------|-------|
{{ range .Storage -}}
| {{ .Name }} | {{ .DiskType }} | {{ .CapacityGB }} GB | {{ .Interface }} | {{ .Model }} |
{{ end }}
{{- end }}
{{- if .GPUs }}
### GPUs

| Model | Vendor | VRAM | Driver |
|-------|--------|------|--------|
{{ range .GPUs -}}
| {{ .Model }} | {{ .Vendor }} | {{ humanizeBytes .VRAMMB }} | {{ .DriverVersion }} |
{{ end }}
{{- end }}
{{- end }}
{{- if .Services }}

## Running Services

| Service | Type | Port | Status | Version |
|---------|------|------|--------|---------|
{{ range .Services -}}
| {{ .Name }} | {{ .ServiceType }} | {{ .Port }} | {{ .Status }} | {{ .Version }} |
{{ end }}
{{- end }}

## Network Position

**Network Layer:** {{ networkLayerLabel .Device.NetworkLayer }}
**Parent Device:** {{ if .Device.ParentDeviceID }}{{ .Device.ParentDeviceID }}{{ else }}Gateway/Root{{ end }}
**Connection Type:** {{ if .Device.ConnectionType }}{{ .Device.ConnectionType }}{{ else }}N/A{{ end }}
{{ if .Children -}}
**Connected Devices:** {{ len .Children }}

| Hostname | IP | Type | Status |
|----------|----|------|--------|
{{ range .Children -}}
| {{ .Hostname }} | {{ primaryIP .IPAddresses }} | {{ deviceTypeLabel .DeviceType }} | {{ .Status }} |
{{ end }}
{{- end }}
{{- if .Alerts }}

## Active Alerts

| Severity | Message | Triggered | Resolved |
|----------|---------|-----------|----------|
{{ range .Alerts -}}
| {{ .Severity }} | {{ .Message }} | {{ formatTime .TriggeredAt }} | {{ if .ResolvedAt }}{{ formatTime (derefTime .ResolvedAt) }}{{ else }}Active{{ end }} |
{{ end }}
{{- end }}
{{- if .RecentChanges }}

## Recent Changes (Last 30 Days)

{{ range .RecentChanges -}}
- {{ eventIcon .EventType }} **[{{ .CreatedAt.Format "2006-01-02 15:04" }}]** {{ .Summary }} {{ sourceTag .SourceModule }}
{{ end }}
{{- end }}

---
*Generated by SubNetree AutoDoc on {{ formatTime .GeneratedAt }}*
`
