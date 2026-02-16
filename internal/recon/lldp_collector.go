package recon

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
	"go.uber.org/zap"

	"github.com/HerbHall/subnetree/pkg/models"
)

// LLDP-MIB OID constants.
// lldpRemTable (1.0.8802.1.1.2.1.4.1) columns indexed by timeMark.localPortNum.index.
const (
	OIDLLDPRemSysDesc        = "1.0.8802.1.1.2.1.4.1.1.4"  // lldpRemSysDesc
	OIDLLDPRemPortID         = "1.0.8802.1.1.2.1.4.1.1.7"  // lldpRemPortId
	OIDLLDPRemPortDesc       = "1.0.8802.1.1.2.1.4.1.1.8"  // lldpRemPortDesc
	OIDLLDPRemSysName        = "1.0.8802.1.1.2.1.4.1.1.9"  // lldpRemSysName
	OIDLLDPRemSysCapSup      = "1.0.8802.1.1.2.1.4.1.1.11" // lldpRemSysCapSupported
	OIDLLDPRemSysCapEnabled  = "1.0.8802.1.1.2.1.4.1.1.12" // lldpRemSysCapEnabled
	OIDLLDPRemManAddr        = "1.0.8802.1.1.2.1.4.2.1.4"  // lldpRemManAddrTable
)

// LLDP capabilities bitmap constants (IEEE 802.1AB).
const (
	LLDPCapOther     uint16 = 0x01
	LLDPCapRepeater  uint16 = 0x02
	LLDPCapBridge    uint16 = 0x04 // Switch
	LLDPCapWLANAP    uint16 = 0x08 // Wireless access point
	LLDPCapRouter    uint16 = 0x10
	LLDPCapTelephone uint16 = 0x20
	LLDPCapDOCSIS    uint16 = 0x40
	LLDPCapStation   uint16 = 0x80
)

// LLDPNeighbor holds information about a single LLDP neighbor discovered on a device.
type LLDPNeighbor struct {
	LocalPort      string // Local port that sees this neighbor
	RemoteSysName  string // Neighbor hostname
	RemoteSysDesc  string // Neighbor system description
	RemotePortID   string // Neighbor port identifier
	RemotePortDesc string // Neighbor port description
	RemoteManAddr  string // Neighbor management IP
	CapSupported   uint16 // Capabilities bitmap (supported)
	CapEnabled     uint16 // Capabilities bitmap (enabled)
}

// LLDPCollector discovers LLDP neighbors via SNMP queries to the LLDP-MIB.
type LLDPCollector struct {
	logger *zap.Logger
}

// NewLLDPCollector creates a new LLDP collector.
func NewLLDPCollector(logger *zap.Logger) *LLDPCollector {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LLDPCollector{logger: logger}
}

// InferDeviceTypeFromLLDPCaps maps LLDP capability bits to a SubNetree DeviceType.
// Priority order: Router > AccessPoint > Switch > Desktop > Unknown.
func InferDeviceTypeFromLLDPCaps(caps uint16) models.DeviceType {
	switch {
	case caps&LLDPCapRouter != 0:
		return models.DeviceTypeRouter
	case caps&LLDPCapWLANAP != 0:
		return models.DeviceTypeAccessPoint
	case caps&LLDPCapBridge != 0:
		return models.DeviceTypeSwitch
	case caps&LLDPCapStation != 0:
		return models.DeviceTypeDesktop
	default:
		return models.DeviceTypeUnknown
	}
}

// DiscoverNeighbors queries LLDP-MIB on the given SNMP session and returns
// all discovered neighbors. Returns an empty slice (not error) if LLDP is
// not supported or the device has no LLDP neighbors.
func (c *LLDPCollector) DiscoverNeighbors(g *gosnmp.GoSNMP) ([]LLDPNeighbor, error) {
	// Walk all columns of the lldpRemTable.
	remTableOIDs := []string{
		OIDLLDPRemSysDesc,
		OIDLLDPRemPortID,
		OIDLLDPRemPortDesc,
		OIDLLDPRemSysName,
		OIDLLDPRemSysCapSup,
		OIDLLDPRemSysCapEnabled,
	}

	// neighborMap groups PDUs by their 3-part index key: timeMark.localPortNum.index.
	type neighborEntry struct {
		localPortNum string
		neighbor     LLDPNeighbor
	}
	neighborMap := make(map[string]*neighborEntry)

	for _, baseOID := range remTableOIDs {
		pdus, err := g.BulkWalkAll(baseOID)
		if err != nil {
			// Many devices don't support LLDP; treat walk errors as "no data".
			c.logger.Debug("LLDP walk returned no data",
				zap.String("oid", baseOID),
				zap.Error(err),
			)
			continue
		}

		for _, pdu := range pdus {
			indexKey, localPort := extractLLDPIndex(pdu.Name, baseOID)
			if indexKey == "" {
				continue
			}

			entry, ok := neighborMap[indexKey]
			if !ok {
				entry = &neighborEntry{
					localPortNum: localPort,
				}
				neighborMap[indexKey] = entry
			}

			oidPrefix := extractLLDPOIDBase(pdu.Name, indexKey)
			switch oidPrefix {
			case OIDLLDPRemSysDesc:
				entry.neighbor.RemoteSysDesc = parsePDUString(pdu)
			case OIDLLDPRemPortID:
				entry.neighbor.RemotePortID = parseLLDPPortID(pdu)
			case OIDLLDPRemPortDesc:
				entry.neighbor.RemotePortDesc = parsePDUString(pdu)
			case OIDLLDPRemSysName:
				entry.neighbor.RemoteSysName = parsePDUString(pdu)
			case OIDLLDPRemSysCapSup:
				entry.neighbor.CapSupported = parseLLDPCapBitmap(pdu)
			case OIDLLDPRemSysCapEnabled:
				entry.neighbor.CapEnabled = parseLLDPCapBitmap(pdu)
			}
		}
	}

	// Walk lldpRemManAddrTable separately (different OID subtree with extra index components).
	manAddrPDUs, err := g.BulkWalkAll(OIDLLDPRemManAddr)
	if err == nil {
		for _, pdu := range manAddrPDUs {
			indexKey, _ := extractLLDPManAddrIndex(pdu.Name)
			if indexKey == "" {
				continue
			}
			if entry, ok := neighborMap[indexKey]; ok && entry.neighbor.RemoteManAddr == "" {
				entry.neighbor.RemoteManAddr = parseLLDPManAddr(pdu)
			}
		}
	}

	// Convert map to sorted slice, populating LocalPort.
	neighbors := make([]LLDPNeighbor, 0, len(neighborMap))
	keys := make([]string, 0, len(neighborMap))
	for k := range neighborMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		entry := neighborMap[k]
		entry.neighbor.LocalPort = entry.localPortNum
		neighbors = append(neighbors, entry.neighbor)
	}

	c.logger.Debug("LLDP neighbor discovery complete",
		zap.Int("neighbors", len(neighbors)),
	)

	return neighbors, nil
}

// BuildTopologyFromLLDP creates topology links from LLDP neighbor data.
// For each neighbor that can be matched to an existing device (by management IP
// or hostname), a topology link with link_type "lldp" is created.
// Returns the number of links created.
func (c *LLDPCollector) BuildTopologyFromLLDP(ctx context.Context, reconStore *ReconStore, neighbors []LLDPNeighbor, sourceDeviceID string) (int, error) {
	created := 0

	for _, n := range neighbors {
		var targetDevice *models.Device

		// Try matching by management IP first (most reliable).
		if n.RemoteManAddr != "" {
			d, err := reconStore.GetDeviceByIP(ctx, n.RemoteManAddr)
			if err == nil && d != nil {
				targetDevice = d
			}
		}

		// Fall back to hostname matching.
		if targetDevice == nil && n.RemoteSysName != "" {
			d, err := reconStore.GetDeviceByHostname(ctx, n.RemoteSysName)
			if err == nil && d != nil {
				targetDevice = d
			}
		}

		if targetDevice == nil {
			c.logger.Debug("LLDP neighbor not matched to known device",
				zap.String("remote_name", n.RemoteSysName),
				zap.String("remote_ip", n.RemoteManAddr),
			)
			continue
		}

		// Don't create self-links.
		if targetDevice.ID == sourceDeviceID {
			continue
		}

		link := &TopologyLink{
			SourceDeviceID: sourceDeviceID,
			TargetDeviceID: targetDevice.ID,
			SourcePort:     n.LocalPort,
			TargetPort:     n.RemotePortID,
			LinkType:       "lldp",
		}

		if err := reconStore.UpsertTopologyLink(ctx, link); err != nil {
			return created, fmt.Errorf("upsert LLDP topology link: %w", err)
		}
		created++

		c.logger.Debug("LLDP topology link created",
			zap.String("source", sourceDeviceID),
			zap.String("target", targetDevice.ID),
			zap.String("source_port", n.LocalPort),
			zap.String("target_port", n.RemotePortID),
		)
	}

	return created, nil
}

// extractLLDPIndex parses the 3-part index (timeMark.localPortNum.index) from
// an LLDP-MIB OID. Returns the composite key and the local port number string.
//
// Full OID format: <baseOID>.<timeMark>.<localPortNum>.<index>
// Example: .1.0.8802.1.1.2.1.4.1.1.9.0.5.1 -> key="0.5.1", localPort="5"
func extractLLDPIndex(oid, baseOID string) (indexKey, localPort string) {
	// Normalize: ensure both have leading dot.
	if !strings.HasPrefix(oid, ".") {
		oid = "." + oid
	}
	prefix := "." + baseOID + "."

	if !strings.HasPrefix(oid, prefix) {
		return "", ""
	}

	suffix := oid[len(prefix):]
	// suffix should be "timeMark.localPortNum.index"
	parts := strings.SplitN(suffix, ".", 3)
	if len(parts) != 3 {
		return "", ""
	}

	return suffix, parts[1]
}

// extractLLDPOIDBase returns the base OID (without the 3-part index suffix) from
// a full LLDP PDU OID. Given the full OID and the known index key, it strips the
// leading dot and trailing index to recover the base OID constant.
func extractLLDPOIDBase(oid, indexKey string) string {
	if !strings.HasPrefix(oid, ".") {
		oid = "." + oid
	}
	// OID = "." + baseOID + "." + indexKey
	suffix := "." + indexKey
	if !strings.HasSuffix(oid, suffix) {
		return ""
	}
	base := oid[:len(oid)-len(suffix)]
	// Remove leading dot to match our constants.
	if strings.HasPrefix(base, ".") {
		base = base[1:]
	}
	return base
}

// extractLLDPManAddrIndex extracts the 3-part neighbor index from an
// lldpRemManAddrTable OID. The management address table has additional
// index components after the standard 3-part index:
// <baseOID>.<timeMark>.<localPortNum>.<index>.<addrSubtype>.<addrLen>.<addr...>
// We extract just the first 3 index parts to correlate with lldpRemTable entries.
func extractLLDPManAddrIndex(oid string) (indexKey, localPort string) {
	if !strings.HasPrefix(oid, ".") {
		oid = "." + oid
	}
	prefix := "." + OIDLLDPRemManAddr + "."
	if !strings.HasPrefix(oid, prefix) {
		return "", ""
	}

	suffix := oid[len(prefix):]
	parts := strings.SplitN(suffix, ".", 4)
	if len(parts) < 3 {
		return "", ""
	}

	return parts[0] + "." + parts[1] + "." + parts[2], parts[1]
}

// parseLLDPPortID extracts the port identifier from an LLDP PDU.
// Port IDs can be MAC addresses (raw bytes) or human-readable strings.
func parseLLDPPortID(pdu gosnmp.SnmpPDU) string {
	switch v := pdu.Value.(type) {
	case []byte:
		// If it looks like a 6-byte MAC address, format it.
		if len(v) == 6 && !isPrintableASCII(v) {
			return formatMAC(v)
		}
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

// parseLLDPCapBitmap extracts the LLDP capabilities bitmap from a PDU.
// The bitmap is typically encoded as a 2-byte octet string.
func parseLLDPCapBitmap(pdu gosnmp.SnmpPDU) uint16 {
	switch v := pdu.Value.(type) {
	case []byte:
		if len(v) >= 2 {
			return uint16(v[0])<<8 | uint16(v[1])
		}
		if len(v) == 1 {
			return uint16(v[0])
		}
		return 0
	case int:
		return uint16(v) //nolint:gosec // G115: LLDP bitmap fits in uint16
	case uint:
		return uint16(v) //nolint:gosec // G115: LLDP bitmap fits in uint16
	case uint32:
		return uint16(v) //nolint:gosec // G115: LLDP bitmap fits in uint16
	default:
		return 0
	}
}

// parseLLDPManAddr extracts the management IP address from an
// lldpRemManAddrTable PDU. The address is typically returned as raw bytes
// (4 bytes for IPv4) or as a string.
func parseLLDPManAddr(pdu gosnmp.SnmpPDU) string {
	switch v := pdu.Value.(type) {
	case []byte:
		if len(v) == 4 {
			return net.IP(v).String()
		}
		if len(v) == 16 {
			return net.IP(v).String()
		}
		// Try interpreting as a string if printable.
		if isPrintableASCII(v) {
			return string(v)
		}
		return ""
	case string:
		return v
	default:
		return ""
	}
}

// parseLLDPManAddrFromOID extracts the management IP address encoded in the OID
// index of an lldpRemManAddrTable entry. Format:
// <baseOID>.<timeMark>.<localPortNum>.<index>.<addrSubtype>.<addrLen>.<addr bytes...>
// For IPv4 (subtype 1): last 4 OID segments are the IP octets.
func parseLLDPManAddrFromOID(oid string) string {
	if !strings.HasPrefix(oid, ".") {
		oid = "." + oid
	}
	prefix := "." + OIDLLDPRemManAddr + "."
	if !strings.HasPrefix(oid, prefix) {
		return ""
	}

	suffix := oid[len(prefix):]
	parts := strings.Split(suffix, ".")
	// Minimum: timeMark.localPort.index.subtype.len.addr(4 octets) = 9 parts for IPv4.
	if len(parts) < 9 {
		return ""
	}

	// parts[3] = address subtype (1=IPv4, 2=IPv6)
	subtype, err := strconv.Atoi(parts[3])
	if err != nil {
		return ""
	}

	addrLen, err := strconv.Atoi(parts[4])
	if err != nil || addrLen <= 0 {
		return ""
	}

	addrStart := 5
	if addrStart+addrLen > len(parts) {
		return ""
	}

	switch subtype {
	case 1: // IPv4
		if addrLen != 4 {
			return ""
		}
		octets := make([]byte, 4)
		for i := range 4 {
			val, parseErr := strconv.Atoi(parts[addrStart+i])
			if parseErr != nil || val < 0 || val > 255 {
				return ""
			}
			octets[i] = byte(val) //nolint:gosec // G115: OID octet [0,255] fits in byte
		}
		return net.IP(octets).String()
	default:
		return ""
	}
}

// isPrintableASCII returns true if all bytes are printable ASCII (0x20..0x7E).
func isPrintableASCII(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7E {
			return false
		}
	}
	return len(b) > 0
}
