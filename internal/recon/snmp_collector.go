// Phase 2: implement SNMP discovery using gosnmp/gosnmp (BSD-2-Clause).
// See .planning/phases/04-phase2-foundation/04-01-FINDINGS.md for research and API examples.

package recon

import (
	"context"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
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
	Username             string
	AuthProtocol         string // "MD5", "SHA", "SHA-256", etc.
	AuthPassphrase       string
	PrivacyProtocol      string // "DES", "AES", "AES-256", etc.
	PrivacyPassphrase    string
	SecurityLevel        string // "noAuthNoPriv", "authNoPriv", "authPriv"
	ContextName          string
	AuthoritativeEngineID string
}

// SNMPSystemInfo holds basic system information retrieved via SNMP.
type SNMPSystemInfo struct {
	Description string        // sysDescr (1.3.6.1.2.1.1.1.0)
	ObjectID    string        // sysObjectID (1.3.6.1.2.1.1.2.0)
	UpTime      time.Duration // sysUpTime (1.3.6.1.2.1.1.3.0)
	Contact     string        // sysContact (1.3.6.1.2.1.1.4.0)
	Name        string        // sysName (1.3.6.1.2.1.1.5.0)
	Location    string        // sysLocation (1.3.6.1.2.1.1.6.0)
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

// Discover uses SNMP to discover devices at the given target IP.
// It queries standard system MIB objects and returns device information.
func (c *SNMPCollector) Discover(ctx context.Context, target string, cred CredentialAccessor, credID string) ([]models.Device, error) {
	_ = ctx
	_ = target
	_ = cred
	_ = credID
	return nil, fmt.Errorf("SNMP discovery not implemented: pending gosnmp integration")
}

// GetSystemInfo retrieves basic system information from an SNMP-enabled device.
// Queries: sysDescr, sysObjectID, sysUpTime, sysContact, sysName, sysLocation.
func (c *SNMPCollector) GetSystemInfo(ctx context.Context, target string, cred CredentialAccessor, credID string) (*SNMPSystemInfo, error) {
	_ = ctx
	_ = target
	_ = cred
	_ = credID
	return nil, fmt.Errorf("SNMP GetSystemInfo not implemented: pending gosnmp integration")
}

// GetInterfaces retrieves the interface table from an SNMP-enabled device.
// Walks the IF-MIB ifTable for interface descriptions, types, status, and counters.
func (c *SNMPCollector) GetInterfaces(ctx context.Context, target string, cred CredentialAccessor, credID string) ([]SNMPInterface, error) {
	_ = ctx
	_ = target
	_ = cred
	_ = credID
	return nil, fmt.Errorf("SNMP GetInterfaces not implemented: pending gosnmp integration")
}
