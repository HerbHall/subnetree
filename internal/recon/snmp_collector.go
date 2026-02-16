// Phase 2: implement SNMP discovery using gosnmp/gosnmp (BSD-2-Clause).
// See .planning/phases/04-phase2-foundation/04-01-FINDINGS.md for research and API examples.

package recon

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosnmp/gosnmp"
	"go.uber.org/zap"

	"github.com/HerbHall/subnetree/pkg/models"
)

// CredentialAccessor retrieves stored credentials for SNMP authentication.
// Defined here (consumer-side) to avoid importing the vault package.
type CredentialAccessor interface {
	GetCredential(ctx context.Context, id string) (*SNMPCredential, error)
}

// SNMPCredential holds the fields needed for SNMP authentication.
type SNMPCredential struct {
	Type string // "snmp_v2c" or "snmp_v3"

	// SNMPv2c fields.
	Community string

	// SNMPv3 fields.
	Username              string
	AuthProtocol          string // "MD5", "SHA", "SHA-256", etc.
	AuthPassphrase        string
	PrivacyProtocol       string // "DES", "AES", "AES-256", etc.
	PrivacyPassphrase     string
	SecurityLevel         string // "noAuthNoPriv", "authNoPriv", "authPriv"
	ContextName           string
	AuthoritativeEngineID string
}

// SNMPSystemInfo holds basic system information retrieved via SNMP.
type SNMPSystemInfo struct {
	Description    string        // sysDescr (1.3.6.1.2.1.1.1.0)
	ObjectID       string        // sysObjectID (1.3.6.1.2.1.1.2.0)
	UpTime         time.Duration // sysUpTime (1.3.6.1.2.1.1.3.0)
	Contact        string        // sysContact (1.3.6.1.2.1.1.4.0)
	Name           string        // sysName (1.3.6.1.2.1.1.5.0)
	Location       string        // sysLocation (1.3.6.1.2.1.1.6.0)
	Services       int           // sysServices (1.3.6.1.2.1.1.7.0) - OSI layer bitmask
	BridgeAddress  string        // dot1dBaseBridgeAddress (BRIDGE-MIB)
	BridgeNumPorts int           // dot1dBaseNumPorts (BRIDGE-MIB)
	BridgeType     int           // dot1dBaseType (BRIDGE-MIB)
}

// SNMPInterface represents a network interface discovered via SNMP IF-MIB.
type SNMPInterface struct {
	Index       int    // ifIndex
	Description string // ifDescr
	Type        int    // ifType (e.g., 6=ethernet, 24=loopback)
	MTU         int    // ifMtu
	Speed       uint64 // ifSpeed (bits per second)
	PhysAddress string // ifPhysAddress (MAC)
	AdminStatus int    // ifAdminStatus (1=up, 2=down, 3=testing)
	OperStatus  int    // ifOperStatus (1=up, 2=down, 3=testing, etc.)
}

// SNMPCollector discovers device information using SNMP queries.
type SNMPCollector struct {
	logger *zap.Logger
}

// NewSNMPCollector creates a new SNMP collector.
func NewSNMPCollector(logger *zap.Logger) *SNMPCollector {
	return &SNMPCollector{logger: logger}
}

// newGoSNMP creates a configured GoSNMP instance for the given target and credential.
// The returned GoSNMP is not yet connected; the caller must call Connect().
func (c *SNMPCollector) newGoSNMP(target string, cred *SNMPCredential) (*gosnmp.GoSNMP, error) {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		// No port specified, default to 161.
		host = target
		portStr = "161"
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %w", portStr, err)
	}

	g := &gosnmp.GoSNMP{
		Target:  host,
		Port:    uint16(port),
		Timeout: 5 * time.Second,
		Retries: 1,
	}

	switch cred.Type {
	case "snmp_v2c":
		g.Version = gosnmp.Version2c
		g.Community = cred.Community

	case "snmp_v3":
		g.Version = gosnmp.Version3
		g.SecurityModel = gosnmp.UserSecurityModel

		// Map security level to MsgFlags.
		switch cred.SecurityLevel {
		case "noAuthNoPriv":
			g.MsgFlags = gosnmp.NoAuthNoPriv
		case "authNoPriv":
			g.MsgFlags = gosnmp.AuthNoPriv
		case "authPriv":
			g.MsgFlags = gosnmp.AuthPriv
		default:
			g.MsgFlags = gosnmp.AuthPriv
		}

		g.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 cred.Username,
			AuthenticationProtocol:   mapAuthProtocol(cred.AuthProtocol),
			AuthenticationPassphrase: cred.AuthPassphrase,
			PrivacyProtocol:          mapPrivProtocol(cred.PrivacyProtocol),
			PrivacyPassphrase:        cred.PrivacyPassphrase,
			AuthoritativeEngineID:    cred.AuthoritativeEngineID,
		}

		if cred.ContextName != "" {
			g.ContextName = cred.ContextName
		}

	default:
		return nil, fmt.Errorf("unsupported SNMP credential type: %s", cred.Type)
	}

	return g, nil
}

// mapAuthProtocol converts an auth protocol string to the gosnmp constant.
func mapAuthProtocol(s string) gosnmp.SnmpV3AuthProtocol {
	switch strings.ToUpper(s) {
	case "MD5":
		return gosnmp.MD5
	case "SHA":
		return gosnmp.SHA
	case "SHA-224", "SHA224":
		return gosnmp.SHA224
	case "SHA-256", "SHA256":
		return gosnmp.SHA256
	case "SHA-384", "SHA384":
		return gosnmp.SHA384
	case "SHA-512", "SHA512":
		return gosnmp.SHA512
	default:
		return gosnmp.SHA
	}
}

// mapPrivProtocol converts a privacy protocol string to the gosnmp constant.
func mapPrivProtocol(s string) gosnmp.SnmpV3PrivProtocol {
	switch strings.ToUpper(s) {
	case "DES":
		return gosnmp.DES
	case "AES", "AES-128", "AES128":
		return gosnmp.AES
	case "AES-192", "AES192":
		return gosnmp.AES192
	case "AES-256", "AES256":
		return gosnmp.AES256
	case "AES-192C", "AES192C":
		return gosnmp.AES192C
	case "AES-256C", "AES256C":
		return gosnmp.AES256C
	default:
		return gosnmp.AES
	}
}

// GetSystemInfo retrieves basic system information from an SNMP-enabled device.
// Queries: sysDescr, sysObjectID, sysUpTime, sysContact, sysName, sysLocation.
func (c *SNMPCollector) GetSystemInfo(ctx context.Context, target string, cred CredentialAccessor, credID string) (*SNMPSystemInfo, error) {
	credential, err := cred.GetCredential(ctx, credID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	g, err := c.newGoSNMP(target, credential)
	if err != nil {
		return nil, fmt.Errorf("configure SNMP: %w", err)
	}

	if err := g.Connect(); err != nil {
		return nil, fmt.Errorf("connect to %s: %w", target, err)
	}
	defer func() { _ = g.Conn.Close() }()

	oids := []string{
		OIDSysDescr,
		OIDSysObjectID,
		OIDSysUpTime,
		OIDSysContact,
		OIDSysName,
		OIDSysLocation,
		OIDSysServices,
	}

	result, err := g.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET system info: %w", err)
	}

	info := &SNMPSystemInfo{}
	for _, pdu := range result.Variables {
		switch pdu.Name {
		case "." + OIDSysDescr:
			info.Description = parsePDUString(pdu)
		case "." + OIDSysObjectID:
			info.ObjectID = parsePDUString(pdu)
		case "." + OIDSysUpTime:
			info.UpTime = parsePDUUpTime(pdu)
		case "." + OIDSysContact:
			info.Contact = parsePDUString(pdu)
		case "." + OIDSysName:
			info.Name = parsePDUString(pdu)
		case "." + OIDSysLocation:
			info.Location = parsePDUString(pdu)
		case "." + OIDSysServices:
			info.Services = parsePDUInt(pdu)
		}
	}

	// Query BRIDGE-MIB separately (not all devices support it).
	c.queryBridgeInfo(g, info)

	c.logger.Debug("SNMP system info retrieved",
		zap.String("target", target),
		zap.String("name", info.Name),
		zap.String("descr", info.Description),
		zap.Int("services", info.Services),
		zap.String("bridgeAddr", info.BridgeAddress),
		zap.Int("bridgePorts", info.BridgeNumPorts),
	)

	return info, nil
}

// queryBridgeInfo attempts to retrieve BRIDGE-MIB data from the device.
// Many devices don't support BRIDGE-MIB, so errors are silently ignored.
func (c *SNMPCollector) queryBridgeInfo(g *gosnmp.GoSNMP, info *SNMPSystemInfo) {
	oids := []string{OIDBridgeBase, OIDBridgeNumPorts, OIDBridgeType}

	result, err := g.Get(oids)
	if err != nil {
		return
	}

	for _, pdu := range result.Variables {
		if pdu.Type == gosnmp.NoSuchObject || pdu.Type == gosnmp.NoSuchInstance {
			continue
		}
		switch pdu.Name {
		case "." + OIDBridgeBase:
			if b, ok := pdu.Value.([]byte); ok && len(b) > 0 {
				info.BridgeAddress = formatMAC(b)
			}
		case "." + OIDBridgeNumPorts:
			info.BridgeNumPorts = parsePDUInt(pdu)
		case "." + OIDBridgeType:
			info.BridgeType = parsePDUInt(pdu)
		}
	}
}

// GetInterfaces retrieves the interface table from an SNMP-enabled device.
// Walks the IF-MIB ifTable for interface descriptions, types, status, and counters.
func (c *SNMPCollector) GetInterfaces(ctx context.Context, target string, cred CredentialAccessor, credID string) ([]SNMPInterface, error) {
	credential, err := cred.GetCredential(ctx, credID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	g, err := c.newGoSNMP(target, credential)
	if err != nil {
		return nil, fmt.Errorf("configure SNMP: %w", err)
	}

	if err := g.Connect(); err != nil {
		return nil, fmt.Errorf("connect to %s: %w", target, err)
	}
	defer func() { _ = g.Conn.Close() }()

	pdus, err := g.BulkWalkAll("1.3.6.1.2.1.2.2.1")
	if err != nil {
		return nil, fmt.Errorf("SNMP walk IF-MIB: %w", err)
	}

	// Group PDUs by interface index.
	ifMap := make(map[int]*SNMPInterface)

	for _, pdu := range pdus {
		// Extract ifIndex from OID suffix (last number after last dot).
		idx := extractOIDIndex(pdu.Name)
		if idx < 0 {
			continue
		}

		iface, ok := ifMap[idx]
		if !ok {
			iface = &SNMPInterface{Index: idx}
			ifMap[idx] = iface
		}

		// Match OID prefix to determine which field this PDU populates.
		oidPrefix := extractOIDPrefix(pdu.Name)
		switch oidPrefix {
		case "."+OIDIfIndex, OIDIfIndex:
			iface.Index = parsePDUInt(pdu)
		case "."+OIDIfDescr, OIDIfDescr:
			iface.Description = parsePDUString(pdu)
		case "."+OIDIfType, OIDIfType:
			iface.Type = parsePDUInt(pdu)
		case "."+OIDIfMtu, OIDIfMtu:
			iface.MTU = parsePDUInt(pdu)
		case "."+OIDIfSpeed, OIDIfSpeed:
			iface.Speed = parsePDUUint64(pdu)
		case "."+OIDIfPhysAddress, OIDIfPhysAddress:
			if b, ok := pdu.Value.([]byte); ok {
				iface.PhysAddress = formatMAC(b)
			}
		case "."+OIDIfAdminStatus, OIDIfAdminStatus:
			iface.AdminStatus = parsePDUInt(pdu)
		case "."+OIDIfOperStatus, OIDIfOperStatus:
			iface.OperStatus = parsePDUInt(pdu)
		}
	}

	// Convert map to sorted slice.
	interfaces := make([]SNMPInterface, 0, len(ifMap))
	for _, iface := range ifMap {
		interfaces = append(interfaces, *iface)
	}
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Index < interfaces[j].Index
	})

	c.logger.Debug("SNMP interfaces retrieved",
		zap.String("target", target),
		zap.Int("count", len(interfaces)),
	)

	return interfaces, nil
}

// Discover uses SNMP to discover devices at the given target IP.
// It queries standard system MIB objects and returns device information.
func (c *SNMPCollector) Discover(ctx context.Context, target string, cred CredentialAccessor, credID string) ([]models.Device, error) {
	sysInfo, err := c.GetSystemInfo(ctx, target, cred, credID)
	if err != nil {
		return nil, fmt.Errorf("get system info: %w", err)
	}

	interfaces, err := c.GetInterfaces(ctx, target, cred, credID)
	if err != nil {
		c.logger.Warn("failed to get interfaces, continuing with system info only",
			zap.String("target", target),
			zap.Error(err),
		)
		interfaces = nil
	}

	// Determine hostname.
	hostname := sysInfo.Name
	if hostname == "" {
		// Strip port from target if present.
		h, _, splitErr := net.SplitHostPort(target)
		if splitErr != nil {
			hostname = target
		} else {
			hostname = h
		}
	}

	// Find first non-loopback MAC address.
	var macAddr string
	for i := range interfaces {
		// ifType 24 = softwareLoopback.
		if interfaces[i].Type == 24 {
			continue
		}
		if interfaces[i].PhysAddress != "" && interfaces[i].PhysAddress != "00:00:00:00:00:00" {
			macAddr = interfaces[i].PhysAddress
			break
		}
	}

	// Extract IP (strip port if present).
	ip := target
	if h, _, splitErr := net.SplitHostPort(target); splitErr == nil {
		ip = h
	}

	now := time.Now()

	device := models.Device{
		ID:              uuid.New().String(),
		Hostname:        hostname,
		DeviceType:      inferDeviceType(sysInfo),
		DiscoveryMethod: models.DiscoverySNMP,
		IPAddresses:     []string{ip},
		MACAddress:      macAddr,
		Status:          models.DeviceStatusOnline,
		FirstSeen:       now,
		LastSeen:        now,
	}

	c.logger.Info("SNMP device discovered",
		zap.String("target", target),
		zap.String("hostname", device.Hostname),
		zap.String("type", string(device.DeviceType)),
		zap.String("mac", device.MACAddress),
	)

	return []models.Device{device}, nil
}

// inferDeviceType determines device type from all available SNMP data.
// Priority: BRIDGE-MIB > sysServices > sysObjectID > sysDescr keywords.
func inferDeviceType(info *SNMPSystemInfo) models.DeviceType {
	// Priority 1: BRIDGE-MIB detection (definitive for switches).
	if info.BridgeAddress != "" || info.BridgeNumPorts > 1 {
		// Device is a bridge/switch. Check if it also routes (L3 switch).
		if info.Services&0x04 != 0 {
			return models.DeviceTypeRouter
		}
		return models.DeviceTypeSwitch
	}

	// Priority 2: sysServices layer detection.
	if info.Services != 0 {
		if info.Services&0x04 != 0 && info.Services&0x02 == 0 {
			return models.DeviceTypeRouter // Layer 3 only
		}
		if info.Services&0x02 != 0 {
			return models.DeviceTypeSwitch // Layer 2
		}
	}

	// Priority 3: sysObjectID vendor prefix.
	if dt := classifyBySysObjectID(info.ObjectID); dt != models.DeviceTypeUnknown {
		return dt
	}

	// Priority 4: sysDescr keyword matching (expanded).
	return classifyBySysDescr(info.Description)
}

// classifyBySysObjectID uses the enterprise OID prefix for vendor identification.
// sysObjectID identifies the vendor/product but not reliably the device type
// (e.g. Cisco makes routers AND switches), so this returns Unknown for most cases.
func classifyBySysObjectID(objectID string) models.DeviceType {
	if objectID == "" {
		return models.DeviceTypeUnknown
	}
	return models.DeviceTypeUnknown
}

// classifyBySysDescr uses keyword matching on the sysDescr string.
func classifyBySysDescr(sysDescr string) models.DeviceType {
	lower := strings.ToLower(sysDescr)

	switch {
	// Routers.
	case strings.Contains(lower, "router"),
		strings.Contains(lower, "routeros"), //nolint:misspell // RouterOS is MikroTik's OS name
		strings.Contains(lower, "mikrotik"):
		return models.DeviceTypeRouter

	// Switches.
	case strings.Contains(lower, "switch"),
		strings.Contains(lower, "catalyst"),
		strings.Contains(lower, "procurve"),
		strings.Contains(lower, "edgeswitch"),
		strings.Contains(lower, "layer 2"),
		strings.Contains(lower, "bridge"):
		return models.DeviceTypeSwitch

	// Access points.
	case strings.Contains(lower, "access point"),
		strings.Contains(lower, "wireless"),
		strings.Contains(lower, "unifi ap"),
		strings.Contains(lower, "airmax"),
		strings.Contains(lower, "airos"):
		return models.DeviceTypeAccessPoint

	// Firewalls.
	case strings.Contains(lower, "firewall"),
		strings.Contains(lower, "pfsense"),
		strings.Contains(lower, "opnsense"),
		strings.Contains(lower, "fortigate"),
		strings.Contains(lower, "sophos"):
		return models.DeviceTypeFirewall

	// Printers.
	case strings.Contains(lower, "printer"),
		strings.Contains(lower, "laserjet"),
		strings.Contains(lower, "inkjet"):
		return models.DeviceTypePrinter

	// NAS.
	case strings.Contains(lower, "nas"),
		strings.Contains(lower, "synology"),
		strings.Contains(lower, "qnap"),
		strings.Contains(lower, "storage"):
		return models.DeviceTypeNAS

	// Servers.
	case strings.Contains(lower, "linux"),
		strings.Contains(lower, "windows"),
		strings.Contains(lower, "freebsd"),
		strings.Contains(lower, "esxi"),
		strings.Contains(lower, "proxmox"):
		return models.DeviceTypeServer

	default:
		return models.DeviceTypeUnknown
	}
}

// formatMAC formats a byte slice as a colon-separated MAC address (XX:XX:XX:XX:XX:XX).
func formatMAC(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = fmt.Sprintf("%02X", v)
	}
	return strings.Join(parts, ":")
}

// parsePDUString extracts a string value from an SNMP PDU.
func parsePDUString(pdu gosnmp.SnmpPDU) string {
	switch v := pdu.Value.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

// parsePDUUpTime extracts a TimeTicks value (hundredths of a second) from
// an SNMP PDU and converts it to a time.Duration.
func parsePDUUpTime(pdu gosnmp.SnmpPDU) time.Duration {
	switch v := pdu.Value.(type) {
	case uint32:
		return time.Duration(v) * 10 * time.Millisecond
	case uint:
		return time.Duration(int64(v)) * 10 * time.Millisecond //nolint:gosec // G115: SNMP TimeTicks fits in int64
	case int:
		return time.Duration(v) * 10 * time.Millisecond
	default:
		return 0
	}
}

// parsePDUInt extracts an integer value from an SNMP PDU.
func parsePDUInt(pdu gosnmp.SnmpPDU) int {
	switch v := pdu.Value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint:
		return int(v) //nolint:gosec // G115: SNMP integer values (ifIndex, ifType, etc.) fit in int
	case uint32:
		return int(v)
	case uint64:
		return int(v) //nolint:gosec // G115: SNMP integer values (ifIndex, ifType, etc.) fit in int
	default:
		return 0
	}
}

// parsePDUUint64 extracts a uint64 value from an SNMP PDU.
func parsePDUUint64(pdu gosnmp.SnmpPDU) uint64 {
	switch v := pdu.Value.(type) {
	case uint64:
		return v
	case uint32:
		return uint64(v)
	case uint:
		return uint64(v)
	case int:
		if v >= 0 {
			return uint64(v)
		}
		return 0
	default:
		return 0
	}
}

// extractOIDIndex extracts the last numeric segment from an OID string.
// For example, ".1.3.6.1.2.1.2.2.1.2.3" returns 3.
func extractOIDIndex(oid string) int {
	lastDot := strings.LastIndex(oid, ".")
	if lastDot < 0 || lastDot == len(oid)-1 {
		return -1
	}
	idx, err := strconv.Atoi(oid[lastDot+1:])
	if err != nil {
		return -1
	}
	return idx
}

// extractOIDPrefix returns the OID with the last segment removed.
// For example, ".1.3.6.1.2.1.2.2.1.2.3" returns ".1.3.6.1.2.1.2.2.1.2".
func extractOIDPrefix(oid string) string {
	lastDot := strings.LastIndex(oid, ".")
	if lastDot < 0 {
		return oid
	}
	return oid[:lastDot]
}
