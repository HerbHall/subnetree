package recon

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestClassify_NilSignals(t *testing.T) {
	result := Classify(nil)

	if result.DeviceType != models.DeviceTypeUnknown {
		t.Errorf("expected Unknown, got %s", result.DeviceType)
	}
	if result.Confidence != 0 {
		t.Errorf("expected confidence 0, got %d", result.Confidence)
	}
	if result.Source != "none" {
		t.Errorf("expected source 'none', got %s", result.Source)
	}
}

func TestClassify_ManualOverride(t *testing.T) {
	signals := &DeviceSignals{
		ManualType:    models.DeviceTypeServer,
		OUIDeviceType: models.DeviceTypeRouter,
		Manufacturer:  "Cisco",
	}

	result := Classify(signals)

	if result.DeviceType != models.DeviceTypeServer {
		t.Errorf("expected Server (manual), got %s", result.DeviceType)
	}
	if result.Confidence != 100 {
		t.Errorf("expected confidence 100, got %d", result.Confidence)
	}
	if result.Source != "manual" {
		t.Errorf("expected source 'manual', got %s", result.Source)
	}
	if len(result.Signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(result.Signals))
	}
	if result.Signals[0].Detail != "Manually set by user" {
		t.Errorf("unexpected detail: %s", result.Signals[0].Detail)
	}
}

func TestClassify_SingleOUI(t *testing.T) {
	signals := &DeviceSignals{
		OUIDeviceType: models.DeviceTypeSwitch,
		Manufacturer:  "Aruba",
	}

	result := Classify(signals)

	if result.DeviceType != models.DeviceTypeSwitch {
		t.Errorf("expected Switch, got %s", result.DeviceType)
	}
	if result.Confidence != WeightOUIVendor {
		t.Errorf("expected confidence %d, got %d", WeightOUIVendor, result.Confidence)
	}
	if result.Source != "oui_vendor" {
		t.Errorf("expected source 'oui_vendor', got %s", result.Source)
	}
}

func TestClassify_SingleSNMPBridge(t *testing.T) {
	signals := &DeviceSignals{
		SNMPInfo: &SNMPSystemInfo{
			BridgeAddress:  "00:1a:2b:3c:4d:5e",
			BridgeNumPorts: 24,
		},
	}

	result := Classify(signals)

	if result.DeviceType != models.DeviceTypeSwitch {
		t.Errorf("expected Switch, got %s", result.DeviceType)
	}
	if result.Confidence != WeightSNMPBridgeMIB {
		t.Errorf("expected confidence %d, got %d", WeightSNMPBridgeMIB, result.Confidence)
	}
	if result.Source != "snmp_bridge_mib" {
		t.Errorf("expected source 'snmp_bridge_mib', got %s", result.Source)
	}
}

func TestClassify_SNMPBridgeWithL3(t *testing.T) {
	signals := &DeviceSignals{
		SNMPInfo: &SNMPSystemInfo{
			BridgeAddress:  "00:1a:2b:3c:4d:5e",
			BridgeNumPorts: 24,
			Services:       0x04, // L3 routing
		},
	}

	result := Classify(signals)

	if result.DeviceType != models.DeviceTypeRouter {
		t.Errorf("expected Router (L3 switch), got %s", result.DeviceType)
	}
	// BRIDGE-MIB(35) + sysServices(30) both say Router = 65
	expectedConfidence := WeightSNMPBridgeMIB + WeightSNMPSysServices
	if result.Confidence != expectedConfidence {
		t.Errorf("expected confidence %d, got %d", expectedConfidence, result.Confidence)
	}
}

func TestClassify_CombinedSignals(t *testing.T) {
	signals := &DeviceSignals{
		OUIDeviceType:  models.DeviceTypeSwitch,
		Manufacturer:   "Aruba",
		PortDeviceType: models.DeviceTypeSwitch,
		OpenPorts:      []int{22, 80, 443},
		SNMPInfo: &SNMPSystemInfo{
			BridgeAddress:  "00:1a:2b:3c:4d:5e",
			BridgeNumPorts: 48,
		},
	}

	result := Classify(signals)

	if result.DeviceType != models.DeviceTypeSwitch {
		t.Errorf("expected Switch, got %s", result.DeviceType)
	}
	// Expected: OUI weight + Port weight + BRIDGE-MIB weight
	expectedConfidence := WeightOUIVendor + WeightPortProfile + WeightSNMPBridgeMIB
	if result.Confidence != expectedConfidence {
		t.Errorf("expected confidence %d, got %d", expectedConfidence, result.Confidence)
	}
}

func TestClassify_ConflictingSignals(t *testing.T) {
	signals := &DeviceSignals{
		OUIDeviceType:  models.DeviceTypeRouter,
		Manufacturer:   "Cisco",
		PortDeviceType: models.DeviceTypeSwitch,
		OpenPorts:      []int{22, 23, 80},
		SNMPInfo: &SNMPSystemInfo{
			BridgeAddress:  "00:1a:2b:3c:4d:5e",
			BridgeNumPorts: 24,
			// Services=0 means no L3, so BRIDGE-MIB says Switch
		},
	}

	result := Classify(signals)

	// Switch wins: BRIDGE-MIB weight + Port weight > Router OUI weight alone
	if result.DeviceType != models.DeviceTypeSwitch {
		t.Errorf("expected Switch (higher aggregate), got %s", result.DeviceType)
	}
	expectedConfidence := WeightSNMPBridgeMIB + WeightPortProfile
	if result.Confidence != expectedConfidence {
		t.Errorf("expected confidence %d, got %d", expectedConfidence, result.Confidence)
	}
}

func TestClassify_LLDPHighPriority(t *testing.T) {
	signals := &DeviceSignals{
		LLDPDeviceType: models.DeviceTypeRouter,
		OUIDeviceType:  models.DeviceTypeSwitch,
		Manufacturer:   "Aruba",
	}

	result := Classify(signals)

	// LLDP(40) for Router vs OUI(25) for Switch -> Router wins
	if result.DeviceType != models.DeviceTypeRouter {
		t.Errorf("expected Router (LLDP higher weight), got %s", result.DeviceType)
	}
	if result.Confidence != WeightLLDPCaps {
		t.Errorf("expected confidence %d, got %d", WeightLLDPCaps, result.Confidence)
	}
	if result.Source != "lldp_caps" {
		t.Errorf("expected source 'lldp_caps', got %s", result.Source)
	}
}

func TestClassify_TTLBoost(t *testing.T) {
	signals := &DeviceSignals{
		OUIDeviceType: models.DeviceTypeRouter,
		Manufacturer:  "Cisco",
		TTL:           255,
	}

	result := Classify(signals)

	// OUI(25) + TTL(10) both say Router = 35
	if result.DeviceType != models.DeviceTypeRouter {
		t.Errorf("expected Router, got %s", result.DeviceType)
	}
	expectedConfidence := WeightOUIVendor + WeightTTLNetwork
	if result.Confidence != expectedConfidence {
		t.Errorf("expected confidence %d, got %d", expectedConfidence, result.Confidence)
	}
}

func TestClassify_AllSignalsAgree(t *testing.T) {
	signals := &DeviceSignals{
		OUIDeviceType:  models.DeviceTypeSwitch,
		Manufacturer:   "Aruba",
		SNMPDeviceType: models.DeviceTypeSwitch,
		SNMPInfo: &SNMPSystemInfo{
			BridgeAddress:  "00:1a:2b:3c:4d:5e",
			BridgeNumPorts: 48,
			Services:       0x02, // L2 data-link
		},
		LLDPDeviceType: models.DeviceTypeSwitch,
		PortDeviceType: models.DeviceTypeSwitch,
		OpenPorts:      []int{22, 80, 443},
		TTL:            64,
	}

	result := Classify(signals)

	if result.DeviceType != models.DeviceTypeSwitch {
		t.Errorf("expected Switch, got %s", result.DeviceType)
	}
	// OUI(25) + sysDescr(10) + BRIDGE-MIB(35) + sysServices(30) + LLDP(40) + Port(15) = 155, capped at 100
	if result.Confidence != 100 {
		t.Errorf("expected confidence capped at 100, got %d", result.Confidence)
	}
}

func TestClassify_EmptySignals(t *testing.T) {
	signals := &DeviceSignals{}

	result := Classify(signals)

	if result.DeviceType != models.DeviceTypeUnknown {
		t.Errorf("expected Unknown, got %s", result.DeviceType)
	}
	if result.Confidence != 0 {
		t.Errorf("expected confidence 0, got %d", result.Confidence)
	}
}

func TestClassify_SignalsSortedByWeight(t *testing.T) {
	signals := &DeviceSignals{
		OUIDeviceType:  models.DeviceTypeSwitch,
		Manufacturer:   "Aruba",
		LLDPDeviceType: models.DeviceTypeSwitch,
		PortDeviceType: models.DeviceTypeSwitch,
	}

	result := Classify(signals)

	if len(result.Signals) < 3 {
		t.Fatalf("expected at least 3 signals, got %d", len(result.Signals))
	}
	// Verify descending order by weight.
	for i := 1; i < len(result.Signals); i++ {
		if result.Signals[i].Weight > result.Signals[i-1].Weight {
			t.Errorf("signals not sorted: index %d (weight %d) > index %d (weight %d)",
				i, result.Signals[i].Weight, i-1, result.Signals[i-1].Weight)
		}
	}
}

func TestConfidenceLevelFor(t *testing.T) {
	tests := []struct {
		score    int
		expected ConfidenceLevel
	}{
		{100, ConfidenceIdentified},
		{50, ConfidenceIdentified},
		{49, ConfidenceProbable},
		{25, ConfidenceProbable},
		{24, ConfidenceUnknown},
		{0, ConfidenceUnknown},
	}

	for _, tc := range tests {
		got := ConfidenceLevelFor(tc.score)
		if got != tc.expected {
			t.Errorf("ConfidenceLevelFor(%d) = %s, want %s", tc.score, got, tc.expected)
		}
	}
}
