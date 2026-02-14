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
