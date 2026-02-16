package recon

// Standard SNMP OIDs for device discovery.

// SNMPv2-MIB system group (1.3.6.1.2.1.1).
const (
	OIDSysDescr    = "1.3.6.1.2.1.1.1.0"
	OIDSysObjectID = "1.3.6.1.2.1.1.2.0"
	OIDSysUpTime   = "1.3.6.1.2.1.1.3.0"
	OIDSysContact  = "1.3.6.1.2.1.1.4.0"
	OIDSysName     = "1.3.6.1.2.1.1.5.0"
	OIDSysLocation = "1.3.6.1.2.1.1.6.0"
)

// sysServices (1.3.6.1.2.1.1.7.0) - OSI layer bitmask.
const OIDSysServices = "1.3.6.1.2.1.1.7.0"

// BRIDGE-MIB (1.3.6.1.2.1.17) - Bridge/switch detection.
const (
	OIDBridgeBase     = "1.3.6.1.2.1.17.1.1.0" // dot1dBaseBridgeAddress
	OIDBridgeNumPorts = "1.3.6.1.2.1.17.1.2.0" // dot1dBaseNumPorts
	OIDBridgeType     = "1.3.6.1.2.1.17.1.3.0" // dot1dBaseType
)

// BRIDGE-MIB Forwarding Database (1.3.6.1.2.1.17.4.3.1).
const (
	OIDFdbAddress = "1.3.6.1.2.1.17.4.3.1.1" // dot1dTpFdbAddress (MAC)
	OIDFdbPort    = "1.3.6.1.2.1.17.4.3.1.2" // dot1dTpFdbPort (bridge port number)
	OIDFdbStatus  = "1.3.6.1.2.1.17.4.3.1.3" // dot1dTpFdbStatus (1=other,2=invalid,3=learned,4=self,5=mgmt)
)

// BRIDGE-MIB port-to-ifIndex mapping.
const OIDBasePortIfIndex = "1.3.6.1.2.1.17.1.4.1.2" // dot1dBasePortIfIndex

// IF-MIB interface table (1.3.6.1.2.1.2.2.1).
const (
	OIDIfIndex       = "1.3.6.1.2.1.2.2.1.1"
	OIDIfDescr       = "1.3.6.1.2.1.2.2.1.2"
	OIDIfType        = "1.3.6.1.2.1.2.2.1.3"
	OIDIfMtu         = "1.3.6.1.2.1.2.2.1.4"
	OIDIfSpeed       = "1.3.6.1.2.1.2.2.1.5"
	OIDIfPhysAddress = "1.3.6.1.2.1.2.2.1.6"
	OIDIfAdminStatus = "1.3.6.1.2.1.2.2.1.7"
	OIDIfOperStatus  = "1.3.6.1.2.1.2.2.1.8"
)

// IF-MIB extensions (1.3.6.1.2.1.31.1.1.1).
const OIDIfName = "1.3.6.1.2.1.31.1.1.1.1" // ifName (short name like "Gi0/1")
