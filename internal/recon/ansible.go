package recon

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/HerbHall/subnetree/pkg/models"
)

// AnsibleInventory represents a complete Ansible inventory in YAML format.
type AnsibleInventory struct {
	All AnsibleGroup `yaml:"all"`
}

// AnsibleGroup is an Ansible inventory group with hosts, vars, and children.
type AnsibleGroup struct {
	Hosts    map[string]AnsibleHost  `yaml:"hosts,omitempty"`
	Vars     map[string]string       `yaml:"vars,omitempty"`
	Children map[string]AnsibleGroup `yaml:"children,omitempty"`
}

// AnsibleHost holds the host variables for one Ansible inventory entry.
type AnsibleHost map[string]any

// handleExportAnsible exports devices as an Ansible YAML inventory.
//
//	@Summary		Export Ansible inventory
//	@Description	Returns all devices as an Ansible-compatible YAML inventory grouped by device type, subnet, and category.
//	@Tags			recon
//	@Produce		text/yaml
//	@Security		BearerAuth
//	@Success		200	{string}	string	"YAML inventory"
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/devices/ansible [get]
func (m *Module) handleExportAnsible(w http.ResponseWriter, r *http.Request) {
	devices, _, err := m.store.ListDevices(r.Context(), ListDevicesOptions{Limit: 100000})
	if err != nil {
		m.logger.Error("failed to list devices for ansible export", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to export devices")
		return
	}

	inventory := buildAnsibleInventory(devices)

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="subnetree-inventory.yml"`)

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(inventory); err != nil {
		m.logger.Error("failed to encode ansible inventory", zap.Error(err))
	}
	_ = enc.Close()
}

func buildAnsibleInventory(devices []models.Device) AnsibleInventory {
	typeGroups := map[string]map[string]AnsibleHost{}
	subnetGroups := map[string]map[string]AnsibleHost{}
	categoryGroups := map[string]map[string]AnsibleHost{}
	allHosts := map[string]AnsibleHost{}

	for i := range devices {
		d := &devices[i]
		if d.Hostname == "" || len(d.IPAddresses) == 0 {
			continue
		}

		host := buildHostVars(d)
		allHosts[d.Hostname] = host

		// Group by device type
		dtype := sanitizeGroupName(string(d.DeviceType))
		if _, ok := typeGroups[dtype]; !ok {
			typeGroups[dtype] = map[string]AnsibleHost{}
		}
		typeGroups[dtype][d.Hostname] = nil

		// Group by subnet
		if subnet := subnetKey(d.IPAddresses[0]); subnet != "" {
			safe := sanitizeGroupName(subnet)
			if _, ok := subnetGroups[safe]; !ok {
				subnetGroups[safe] = map[string]AnsibleHost{}
			}
			subnetGroups[safe][d.Hostname] = nil
		}

		// Group by category
		if d.Category != "" {
			cat := sanitizeGroupName(d.Category)
			if _, ok := categoryGroups[cat]; !ok {
				categoryGroups[cat] = map[string]AnsibleHost{}
			}
			categoryGroups[cat][d.Hostname] = nil
		}
	}

	children := map[string]AnsibleGroup{}

	// Type groups
	typeChildren := map[string]AnsibleGroup{}
	for name, hosts := range typeGroups {
		typeChildren[name] = AnsibleGroup{Hosts: hosts}
	}
	if len(typeChildren) > 0 {
		children["by_type"] = AnsibleGroup{Children: typeChildren}
	}

	// Subnet groups
	subnetChildren := map[string]AnsibleGroup{}
	for name, hosts := range subnetGroups {
		subnetChildren[name] = AnsibleGroup{Hosts: hosts}
	}
	if len(subnetChildren) > 0 {
		children["by_subnet"] = AnsibleGroup{Children: subnetChildren}
	}

	// Category groups
	catChildren := map[string]AnsibleGroup{}
	for name, hosts := range categoryGroups {
		catChildren[name] = AnsibleGroup{Hosts: hosts}
	}
	if len(catChildren) > 0 {
		children["by_category"] = AnsibleGroup{Children: catChildren}
	}

	return AnsibleInventory{
		All: AnsibleGroup{
			Hosts:    allHosts,
			Children: children,
		},
	}
}

func buildHostVars(d *models.Device) AnsibleHost {
	vars := AnsibleHost{
		"ansible_host":               d.IPAddresses[0],
		"subnetree_id":               d.ID,
		"subnetree_device_type":      string(d.DeviceType),
		"subnetree_status":           string(d.Status),
		"subnetree_discovery_method": string(d.DiscoveryMethod),
	}

	if d.MACAddress != "" {
		vars["subnetree_mac_address"] = d.MACAddress
	}
	if d.Manufacturer != "" {
		vars["subnetree_manufacturer"] = d.Manufacturer
	}
	if d.OS != "" {
		vars["subnetree_os"] = d.OS
	}
	if d.Location != "" {
		vars["subnetree_location"] = d.Location
	}
	if d.PrimaryRole != "" {
		vars["subnetree_primary_role"] = d.PrimaryRole
	}
	if d.Owner != "" {
		vars["subnetree_owner"] = d.Owner
	}
	if len(d.IPAddresses) > 1 {
		vars["subnetree_ip_addresses"] = d.IPAddresses
	}
	if len(d.Tags) > 0 {
		vars["subnetree_tags"] = d.Tags
	}
	if d.NetworkLayer > 0 {
		vars["subnetree_network_layer"] = d.NetworkLayer
	}
	if d.ConnectionType != "" {
		vars["subnetree_connection_type"] = d.ConnectionType
	}
	for k, v := range d.CustomFields {
		vars["subnetree_custom_"+sanitizeGroupName(k)] = v
	}

	return vars
}

func subnetKey(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	v4 := ip.To4()
	if v4 == nil {
		return ""
	}
	return fmt.Sprintf("%d_%d_%d_0_24", v4[0], v4[1], v4[2])
}

func sanitizeGroupName(name string) string {
	r := strings.NewReplacer(
		"-", "_", " ", "_", ".", "_", "/", "_", ":", "_",
	)
	return r.Replace(name)
}
