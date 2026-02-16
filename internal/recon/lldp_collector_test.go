package recon

import (
	"context"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
)

func TestInferDeviceTypeFromLLDPCaps(t *testing.T) {
	tests := []struct {
		name string
		caps uint16
		want models.DeviceType
	}{
		{"router_only", LLDPCapRouter, models.DeviceTypeRouter},
		{"bridge_only", LLDPCapBridge, models.DeviceTypeSwitch},
		{"wlan_ap_only", LLDPCapWLANAP, models.DeviceTypeAccessPoint},
		{"station_only", LLDPCapStation, models.DeviceTypeDesktop},
		{"no_bits", 0, models.DeviceTypeUnknown},
		{"other_only", LLDPCapOther, models.DeviceTypeUnknown},
		{"repeater_only", LLDPCapRepeater, models.DeviceTypeUnknown},
		{"telephone_only", LLDPCapTelephone, models.DeviceTypeUnknown},
		{"docsis_only", LLDPCapDOCSIS, models.DeviceTypeUnknown},

		// Priority: router > access_point > switch > desktop.
		{"router_and_bridge", LLDPCapRouter | LLDPCapBridge, models.DeviceTypeRouter},
		{"router_and_wlan", LLDPCapRouter | LLDPCapWLANAP, models.DeviceTypeRouter},
		{"bridge_and_wlan", LLDPCapBridge | LLDPCapWLANAP, models.DeviceTypeAccessPoint},
		{"bridge_and_station", LLDPCapBridge | LLDPCapStation, models.DeviceTypeSwitch},
		{"wlan_and_station", LLDPCapWLANAP | LLDPCapStation, models.DeviceTypeAccessPoint},
		{"all_bits", LLDPCapOther | LLDPCapRepeater | LLDPCapBridge | LLDPCapWLANAP |
			LLDPCapRouter | LLDPCapTelephone | LLDPCapDOCSIS | LLDPCapStation,
			models.DeviceTypeRouter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferDeviceTypeFromLLDPCaps(tt.caps)
			if got != tt.want {
				t.Errorf("InferDeviceTypeFromLLDPCaps(0x%02X) = %v, want %v", tt.caps, got, tt.want)
			}
		})
	}
}

func TestExtractLLDPIndex(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		baseOID  string
		wantKey  string
		wantPort string
	}{
		{
			name:     "standard_3part_index",
			oid:      ".1.0.8802.1.1.2.1.4.1.1.9.0.5.1",
			baseOID:  OIDLLDPRemSysName,
			wantKey:  "0.5.1",
			wantPort: "5",
		},
		{
			name:     "different_oid_column",
			oid:      ".1.0.8802.1.1.2.1.4.1.1.4.0.3.2",
			baseOID:  OIDLLDPRemSysDesc,
			wantKey:  "0.3.2",
			wantPort: "3",
		},
		{
			name:     "no_leading_dot",
			oid:      "1.0.8802.1.1.2.1.4.1.1.9.0.5.1",
			baseOID:  OIDLLDPRemSysName,
			wantKey:  "0.5.1",
			wantPort: "5",
		},
		{
			name:     "wrong_base_oid",
			oid:      ".1.3.6.1.2.1.1.1.0",
			baseOID:  OIDLLDPRemSysName,
			wantKey:  "",
			wantPort: "",
		},
		{
			name:     "too_few_index_parts",
			oid:      ".1.0.8802.1.1.2.1.4.1.1.9.0.5",
			baseOID:  OIDLLDPRemSysName,
			wantKey:  "",
			wantPort: "",
		},
		{
			name:     "empty_oid",
			oid:      "",
			baseOID:  OIDLLDPRemSysName,
			wantKey:  "",
			wantPort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotPort := extractLLDPIndex(tt.oid, tt.baseOID)
			if gotKey != tt.wantKey {
				t.Errorf("extractLLDPIndex() key = %q, want %q", gotKey, tt.wantKey)
			}
			if gotPort != tt.wantPort {
				t.Errorf("extractLLDPIndex() localPort = %q, want %q", gotPort, tt.wantPort)
			}
		})
	}
}

func TestExtractLLDPOIDBase(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		indexKey string
		want     string
	}{
		{
			name:     "sysname_column",
			oid:      ".1.0.8802.1.1.2.1.4.1.1.9.0.5.1",
			indexKey: "0.5.1",
			want:     OIDLLDPRemSysName,
		},
		{
			name:     "sysdesc_column",
			oid:      ".1.0.8802.1.1.2.1.4.1.1.4.0.3.2",
			indexKey: "0.3.2",
			want:     OIDLLDPRemSysDesc,
		},
		{
			name:     "index_not_at_end",
			oid:      ".1.0.8802.1.1.2.1.4.1.1.9.0.5.1.extra",
			indexKey: "0.5.1",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLLDPOIDBase(tt.oid, tt.indexKey)
			if got != tt.want {
				t.Errorf("extractLLDPOIDBase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractLLDPManAddrIndex(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		wantKey  string
		wantPort string
	}{
		{
			name:     "ipv4_address",
			oid:      ".1.0.8802.1.1.2.1.4.2.1.4.0.5.1.1.4.192.168.1.1",
			wantKey:  "0.5.1",
			wantPort: "5",
		},
		{
			name:     "wrong_prefix",
			oid:      ".1.3.6.1.2.1.1.1.0",
			wantKey:  "",
			wantPort: "",
		},
		{
			name:     "too_few_parts",
			oid:      ".1.0.8802.1.1.2.1.4.2.1.4.0.5",
			wantKey:  "",
			wantPort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotPort := extractLLDPManAddrIndex(tt.oid)
			if gotKey != tt.wantKey {
				t.Errorf("extractLLDPManAddrIndex() key = %q, want %q", gotKey, tt.wantKey)
			}
			if gotPort != tt.wantPort {
				t.Errorf("extractLLDPManAddrIndex() localPort = %q, want %q", gotPort, tt.wantPort)
			}
		})
	}
}

func TestParseLLDPCapBitmap(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  uint16
	}{
		{"two_bytes_router", []byte{0x00, 0x10}, LLDPCapRouter},
		{"two_bytes_bridge", []byte{0x00, 0x04}, LLDPCapBridge},
		{"two_bytes_multi", []byte{0x00, 0x14}, LLDPCapRouter | LLDPCapBridge},
		{"one_byte", []byte{0x10}, LLDPCapRouter},
		{"empty_bytes", []byte{}, 0},
		{"int_value", int(0x10), LLDPCapRouter},
		{"uint_value", uint(0x04), LLDPCapBridge},
		{"uint32_value", uint32(0x80), LLDPCapStation},
		{"nil_value", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := gosnmp.SnmpPDU{Value: tt.value}
			got := parseLLDPCapBitmap(pdu)
			if got != tt.want {
				t.Errorf("parseLLDPCapBitmap() = 0x%02X, want 0x%02X", got, tt.want)
			}
		})
	}
}

func TestParseLLDPPortID(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"string_port", "GigabitEthernet0/1", "GigabitEthernet0/1"},
		{"ascii_bytes", []byte("eth0"), "eth0"},
		{"mac_bytes", []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}, "AA:BB:CC:DD:EE:FF"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := gosnmp.SnmpPDU{Value: tt.value}
			got := parseLLDPPortID(pdu)
			if got != tt.want {
				t.Errorf("parseLLDPPortID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseLLDPManAddr(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"ipv4_bytes", []byte{192, 168, 1, 1}, "192.168.1.1"},
		{"ipv4_loopback", []byte{127, 0, 0, 1}, "127.0.0.1"},
		{"string_ip", "10.0.0.1", "10.0.0.1"},
		{"non_printable_3bytes", []byte{0x01, 0x02, 0x03}, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := gosnmp.SnmpPDU{Value: tt.value}
			got := parseLLDPManAddr(pdu)
			if got != tt.want {
				t.Errorf("parseLLDPManAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseLLDPManAddrFromOID(t *testing.T) {
	tests := []struct {
		name string
		oid  string
		want string
	}{
		{
			name: "ipv4_standard",
			oid:  ".1.0.8802.1.1.2.1.4.2.1.4.0.5.1.1.4.192.168.1.1",
			want: "192.168.1.1",
		},
		{
			name: "ipv4_10_network",
			oid:  ".1.0.8802.1.1.2.1.4.2.1.4.0.3.2.1.4.10.0.0.1",
			want: "10.0.0.1",
		},
		{
			name: "too_short",
			oid:  ".1.0.8802.1.1.2.1.4.2.1.4.0.5",
			want: "",
		},
		{
			name: "wrong_prefix",
			oid:  ".1.3.6.1.2.1.1.1.0",
			want: "",
		},
		{
			name: "empty",
			oid:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLLDPManAddrFromOID(tt.oid)
			if got != tt.want {
				t.Errorf("parseLLDPManAddrFromOID(%q) = %q, want %q", tt.oid, got, tt.want)
			}
		})
	}
}

func TestIsPrintableASCII(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"printable", []byte("hello"), true},
		{"with_space", []byte("hello world"), true},
		{"empty", []byte{}, false},
		{"control_char", []byte{0x01, 0x02}, false},
		{"mixed", []byte{0x41, 0x00}, false},
		{"tilde", []byte("~"), true},
		{"del", []byte{0x7F}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPrintableASCII(tt.input)
			if got != tt.want {
				t.Errorf("isPrintableASCII(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildTopologyFromLLDP(t *testing.T) {
	// Set up in-memory DB with recon schema.
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	reconStore := NewReconStore(db.DB())

	// Seed devices: a switch (source), a router, and a workstation.
	switchDev := &models.Device{
		ID:              "switch-01",
		Hostname:        "core-switch",
		IPAddresses:     []string{"192.168.1.1"},
		DeviceType:      models.DeviceTypeSwitch,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoverySNMP,
	}
	routerDev := &models.Device{
		ID:              "router-01",
		Hostname:        "edge-router",
		IPAddresses:     []string{"192.168.1.254"},
		DeviceType:      models.DeviceTypeRouter,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoverySNMP,
	}
	workstationDev := &models.Device{
		ID:              "ws-01",
		Hostname:        "dev-workstation",
		IPAddresses:     []string{"192.168.1.50"},
		DeviceType:      models.DeviceTypeDesktop,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}

	for _, d := range []*models.Device{switchDev, routerDev, workstationDev} {
		if _, err := reconStore.UpsertDevice(ctx, d); err != nil {
			t.Fatalf("seed device %s: %v", d.Hostname, err)
		}
	}

	// LLDP neighbors as seen from the switch.
	neighbors := []LLDPNeighbor{
		{
			LocalPort:     "1",
			RemoteSysName: "edge-router",
			RemoteManAddr: "192.168.1.254",
			RemotePortID:  "GigabitEthernet0/0",
			CapEnabled:    LLDPCapRouter,
		},
		{
			LocalPort:     "5",
			RemoteSysName: "dev-workstation",
			RemoteManAddr: "192.168.1.50",
			RemotePortID:  "eth0",
			CapEnabled:    LLDPCapStation,
		},
		{
			// Unknown neighbor (not in DB).
			LocalPort:     "24",
			RemoteSysName: "unknown-ap",
			RemoteManAddr: "192.168.1.200",
			RemotePortID:  "wlan0",
			CapEnabled:    LLDPCapWLANAP,
		},
	}

	collector := NewLLDPCollector(nil)
	created, err := collector.BuildTopologyFromLLDP(ctx, reconStore, neighbors, switchDev.ID)
	if err != nil {
		t.Fatalf("BuildTopologyFromLLDP: %v", err)
	}

	// Should create 2 links (router + workstation); unknown-ap is not in DB.
	if created != 2 {
		t.Errorf("links created = %d, want 2", created)
	}

	// Verify topology links.
	links, err := reconStore.GetTopologyLinks(ctx)
	if err != nil {
		t.Fatalf("GetTopologyLinks: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("topology links count = %d, want 2", len(links))
	}

	// Check each link.
	linkMap := make(map[string]TopologyLink)
	for _, l := range links {
		linkMap[l.TargetDeviceID] = l
	}

	routerLink, ok := linkMap[routerDev.ID]
	if !ok {
		t.Fatal("missing topology link to router")
	}
	if routerLink.SourceDeviceID != switchDev.ID {
		t.Errorf("router link source = %q, want %q", routerLink.SourceDeviceID, switchDev.ID)
	}
	if routerLink.LinkType != "lldp" {
		t.Errorf("router link type = %q, want %q", routerLink.LinkType, "lldp")
	}
	if routerLink.SourcePort != "1" {
		t.Errorf("router link source_port = %q, want %q", routerLink.SourcePort, "1")
	}
	if routerLink.TargetPort != "GigabitEthernet0/0" {
		t.Errorf("router link target_port = %q, want %q", routerLink.TargetPort, "GigabitEthernet0/0")
	}

	wsLink, ok := linkMap[workstationDev.ID]
	if !ok {
		t.Fatal("missing topology link to workstation")
	}
	if wsLink.SourcePort != "5" {
		t.Errorf("ws link source_port = %q, want %q", wsLink.SourcePort, "5")
	}
	if wsLink.TargetPort != "eth0" {
		t.Errorf("ws link target_port = %q, want %q", wsLink.TargetPort, "eth0")
	}
}

func TestBuildTopologyFromLLDP_SkipsSelfLinks(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	reconStore := NewReconStore(db.DB())

	device := &models.Device{
		ID:              "switch-01",
		Hostname:        "core-switch",
		IPAddresses:     []string{"192.168.1.1"},
		DeviceType:      models.DeviceTypeSwitch,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoverySNMP,
	}
	if _, err := reconStore.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("seed device: %v", err)
	}

	// Neighbor that resolves to the same device (self-link).
	neighbors := []LLDPNeighbor{
		{
			LocalPort:     "1",
			RemoteSysName: "core-switch",
			RemoteManAddr: "192.168.1.1",
			RemotePortID:  "eth0",
		},
	}

	collector := NewLLDPCollector(nil)
	created, err := collector.BuildTopologyFromLLDP(ctx, reconStore, neighbors, device.ID)
	if err != nil {
		t.Fatalf("BuildTopologyFromLLDP: %v", err)
	}
	if created != 0 {
		t.Errorf("links created = %d, want 0 (self-link should be skipped)", created)
	}
}

func TestBuildTopologyFromLLDP_MatchByHostname(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	reconStore := NewReconStore(db.DB())

	source := &models.Device{
		ID:              "switch-01",
		Hostname:        "core-switch",
		IPAddresses:     []string{"192.168.1.1"},
		DeviceType:      models.DeviceTypeSwitch,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoverySNMP,
	}
	target := &models.Device{
		ID:              "server-01",
		Hostname:        "fileserver",
		IPAddresses:     []string{"192.168.1.100"},
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	for _, d := range []*models.Device{source, target} {
		if _, err := reconStore.UpsertDevice(ctx, d); err != nil {
			t.Fatalf("seed device %s: %v", d.Hostname, err)
		}
	}

	// Neighbor with no management IP, only hostname.
	neighbors := []LLDPNeighbor{
		{
			LocalPort:     "10",
			RemoteSysName: "fileserver",
			RemotePortID:  "eno1",
		},
	}

	collector := NewLLDPCollector(nil)
	created, err := collector.BuildTopologyFromLLDP(ctx, reconStore, neighbors, source.ID)
	if err != nil {
		t.Fatalf("BuildTopologyFromLLDP: %v", err)
	}
	if created != 1 {
		t.Errorf("links created = %d, want 1", created)
	}
}
