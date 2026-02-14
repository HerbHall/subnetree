package recon

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/HerbHall/subnetree/pkg/models"
)

func TestNewGoSNMP_V2c(t *testing.T) {
	c := NewSNMPCollector(nil)
	cred := &SNMPCredential{
		Type:      "snmp_v2c",
		Community: "public",
	}

	g, err := c.newGoSNMP("192.168.1.1", cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Target != "192.168.1.1" {
		t.Errorf("target = %q, want %q", g.Target, "192.168.1.1")
	}
	if g.Port != 161 {
		t.Errorf("port = %d, want 161", g.Port)
	}
	if g.Version != gosnmp.Version2c {
		t.Errorf("version = %v, want Version2c", g.Version)
	}
	if g.Community != "public" {
		t.Errorf("community = %q, want %q", g.Community, "public")
	}
	if g.Timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", g.Timeout)
	}
	if g.Retries != 1 {
		t.Errorf("retries = %d, want 1", g.Retries)
	}
}

func TestNewGoSNMP_V2c_WithPort(t *testing.T) {
	c := NewSNMPCollector(nil)
	cred := &SNMPCredential{
		Type:      "snmp_v2c",
		Community: "public",
	}

	g, err := c.newGoSNMP("192.168.1.1:1161", cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Target != "192.168.1.1" {
		t.Errorf("target = %q, want %q", g.Target, "192.168.1.1")
	}
	if g.Port != 1161 {
		t.Errorf("port = %d, want 1161", g.Port)
	}
}

func TestNewGoSNMP_V3(t *testing.T) {
	c := NewSNMPCollector(nil)
	cred := &SNMPCredential{
		Type:              "snmp_v3",
		Username:          "admin",
		AuthProtocol:      "SHA-256",
		AuthPassphrase:    "authpass123",
		PrivacyProtocol:   "AES-256",
		PrivacyPassphrase: "privpass123",
		SecurityLevel:     "authPriv",
		ContextName:       "mycontext",
	}

	g, err := c.newGoSNMP("10.0.0.1", cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Version != gosnmp.Version3 {
		t.Errorf("version = %v, want Version3", g.Version)
	}
	if g.SecurityModel != gosnmp.UserSecurityModel {
		t.Errorf("security model = %v, want UserSecurityModel", g.SecurityModel)
	}
	if g.MsgFlags != gosnmp.AuthPriv {
		t.Errorf("msg flags = %v, want AuthPriv", g.MsgFlags)
	}
	if g.ContextName != "mycontext" {
		t.Errorf("context name = %q, want %q", g.ContextName, "mycontext")
	}

	usp, ok := g.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok {
		t.Fatal("security parameters is not *UsmSecurityParameters")
	}
	if usp.UserName != "admin" {
		t.Errorf("username = %q, want %q", usp.UserName, "admin")
	}
	if usp.AuthenticationProtocol != gosnmp.SHA256 {
		t.Errorf("auth protocol = %v, want SHA256", usp.AuthenticationProtocol)
	}
	if usp.AuthenticationPassphrase != "authpass123" {
		t.Errorf("auth passphrase = %q, want %q", usp.AuthenticationPassphrase, "authpass123")
	}
	if usp.PrivacyProtocol != gosnmp.AES256 {
		t.Errorf("priv protocol = %v, want AES256", usp.PrivacyProtocol)
	}
	if usp.PrivacyPassphrase != "privpass123" {
		t.Errorf("priv passphrase = %q, want %q", usp.PrivacyPassphrase, "privpass123")
	}
}

func TestNewGoSNMP_V3_SecurityLevels(t *testing.T) {
	c := NewSNMPCollector(nil)
	tests := []struct {
		level string
		want  gosnmp.SnmpV3MsgFlags
	}{
		{"noAuthNoPriv", gosnmp.NoAuthNoPriv},
		{"authNoPriv", gosnmp.AuthNoPriv},
		{"authPriv", gosnmp.AuthPriv},
		{"unknown", gosnmp.AuthPriv}, // default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cred := &SNMPCredential{
				Type:          "snmp_v3",
				Username:      "user",
				SecurityLevel: tt.level,
			}
			g, err := c.newGoSNMP("10.0.0.1", cred)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if g.MsgFlags != tt.want {
				t.Errorf("MsgFlags = %v, want %v", g.MsgFlags, tt.want)
			}
		})
	}
}

func TestNewGoSNMP_InvalidType(t *testing.T) {
	c := NewSNMPCollector(nil)
	cred := &SNMPCredential{
		Type: "snmp_v1",
	}

	_, err := c.newGoSNMP("192.168.1.1", cred)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

func TestMapAuthProtocol(t *testing.T) {
	tests := []struct {
		input string
		want  gosnmp.SnmpV3AuthProtocol
	}{
		{"MD5", gosnmp.MD5},
		{"md5", gosnmp.MD5},
		{"SHA", gosnmp.SHA},
		{"sha", gosnmp.SHA},
		{"SHA-224", gosnmp.SHA224},
		{"SHA224", gosnmp.SHA224},
		{"SHA-256", gosnmp.SHA256},
		{"SHA256", gosnmp.SHA256},
		{"SHA-384", gosnmp.SHA384},
		{"SHA384", gosnmp.SHA384},
		{"SHA-512", gosnmp.SHA512},
		{"SHA512", gosnmp.SHA512},
		{"", gosnmp.SHA},       // default
		{"unknown", gosnmp.SHA}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapAuthProtocol(tt.input)
			if got != tt.want {
				t.Errorf("mapAuthProtocol(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapPrivProtocol(t *testing.T) {
	tests := []struct {
		input string
		want  gosnmp.SnmpV3PrivProtocol
	}{
		{"DES", gosnmp.DES},
		{"des", gosnmp.DES},
		{"AES", gosnmp.AES},
		{"aes", gosnmp.AES},
		{"AES-128", gosnmp.AES},
		{"AES128", gosnmp.AES},
		{"AES-192", gosnmp.AES192},
		{"AES192", gosnmp.AES192},
		{"AES-256", gosnmp.AES256},
		{"AES256", gosnmp.AES256},
		{"AES-192C", gosnmp.AES192C},
		{"AES192C", gosnmp.AES192C},
		{"AES-256C", gosnmp.AES256C},
		{"AES256C", gosnmp.AES256C},
		{"", gosnmp.AES},       // default
		{"unknown", gosnmp.AES}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapPrivProtocol(tt.input)
			if got != tt.want {
				t.Errorf("mapPrivProtocol(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParsePDUString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"byte_slice", []byte("hello"), "hello"},
		{"string", "world", "world"},
		{"int", 42, "42"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := gosnmp.SnmpPDU{Value: tt.value}
			got := parsePDUString(pdu)
			if got != tt.want {
				t.Errorf("parsePDUString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePDUUpTime(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  time.Duration
	}{
		{"uint32", uint32(100), time.Second},          // 100 centiseconds = 1s
		{"uint32_large", uint32(360000), time.Hour},   // 360000 * 10ms = 1h
		{"uint", uint(500), 5 * time.Second},
		{"int", int(200), 2 * time.Second},
		{"nil", nil, 0},
		{"string", "not a number", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := gosnmp.SnmpPDU{Value: tt.value}
			got := parsePDUUpTime(pdu)
			if got != tt.want {
				t.Errorf("parsePDUUpTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatMAC(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"standard_6_bytes", []byte{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E}, "00:1A:2B:3C:4D:5E"},
		{"all_zeros", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, "00:00:00:00:00:00"},
		{"all_ff", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, "FF:FF:FF:FF:FF:FF"},
		{"empty", []byte{}, ""},
		{"nil", nil, ""},
		{"single_byte", []byte{0xAB}, "AB"},
		{"eight_bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, "01:02:03:04:05:06:07:08"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMAC(tt.input)
			if got != tt.want {
				t.Errorf("formatMAC() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInferDeviceType(t *testing.T) {
	tests := []struct {
		sysDescr string
		want     models.DeviceType
	}{
		{"Cisco IOS Software, Router", models.DeviceTypeRouter},
		{"ROUTER firmware v2.1", models.DeviceTypeRouter},
		{"HP ProCurve Switch", models.DeviceTypeSwitch},
		{"Juniper Firewall SRX", models.DeviceTypeFirewall},
		{"HP LaserJet Printer", models.DeviceTypePrinter},
		{"Ubiquiti Access Point", models.DeviceTypeAccessPoint},
		{"Wireless Controller", models.DeviceTypeAccessPoint},
		{"Synology NAS DS920+", models.DeviceTypeNAS},
		{"NetApp Storage System", models.DeviceTypeNAS},
		{"Linux 5.15.0-generic", models.DeviceTypeServer},
		{"Microsoft Windows Server 2022", models.DeviceTypeServer},
		{"FreeBSD 14.0-RELEASE", models.DeviceTypeServer},
		{"Some Unknown Device", models.DeviceTypeUnknown},
		{"", models.DeviceTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.sysDescr, func(t *testing.T) {
			got := inferDeviceType(tt.sysDescr)
			if got != tt.want {
				t.Errorf("inferDeviceType(%q) = %v, want %v", tt.sysDescr, got, tt.want)
			}
		})
	}
}

func TestExtractOIDIndex(t *testing.T) {
	tests := []struct {
		oid  string
		want int
	}{
		{".1.3.6.1.2.1.2.2.1.2.3", 3},
		{".1.3.6.1.2.1.2.2.1.1.1", 1},
		{".1.3.6.1.2.1.2.2.1.6.10", 10},
		{"invalid", -1},
		{"", -1},
		{".1.3.6.1.2.1.2.2.1.2.", -1},
	}

	for _, tt := range tests {
		t.Run(tt.oid, func(t *testing.T) {
			got := extractOIDIndex(tt.oid)
			if got != tt.want {
				t.Errorf("extractOIDIndex(%q) = %d, want %d", tt.oid, got, tt.want)
			}
		})
	}
}

func TestExtractOIDPrefix(t *testing.T) {
	tests := []struct {
		oid  string
		want string
	}{
		{".1.3.6.1.2.1.2.2.1.2.3", ".1.3.6.1.2.1.2.2.1.2"},
		{".1.3.6.1.2.1.2.2.1.1.1", ".1.3.6.1.2.1.2.2.1.1"},
		{"nodots", "nodots"},
	}

	for _, tt := range tests {
		t.Run(tt.oid, func(t *testing.T) {
			got := extractOIDPrefix(tt.oid)
			if got != tt.want {
				t.Errorf("extractOIDPrefix(%q) = %q, want %q", tt.oid, got, tt.want)
			}
		})
	}
}

func TestParsePDUInt(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int
	}{
		{"int", int(42), 42},
		{"int64", int64(100), 100},
		{"uint", uint(7), 7},
		{"uint32", uint32(255), 255},
		{"uint64", uint64(1000), 1000},
		{"nil", nil, 0},
		{"string", "not a number", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := gosnmp.SnmpPDU{Value: tt.value}
			got := parsePDUInt(pdu)
			if got != tt.want {
				t.Errorf("parsePDUInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParsePDUUint64(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  uint64
	}{
		{"uint64", uint64(1000000), 1000000},
		{"uint32", uint32(500), 500},
		{"uint", uint(42), 42},
		{"int_positive", int(99), 99},
		{"int_negative", int(-1), 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := gosnmp.SnmpPDU{Value: tt.value}
			got := parsePDUUint64(pdu)
			if got != tt.want {
				t.Errorf("parsePDUUint64() = %d, want %d", got, tt.want)
			}
		})
	}
}
