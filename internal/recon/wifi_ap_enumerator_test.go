package recon

import (
	"context"
	"testing"
	"time"
)

// Compile-time interface guard for the mock enumerator.
var _ APClientEnumerator = (*mockAPClientEnumerator)(nil)

type mockAPClientEnumerator struct {
	available bool
	clients   []APClientInfo
	err       error
}

func (m *mockAPClientEnumerator) Available() bool { return m.available }

func (m *mockAPClientEnumerator) Enumerate(_ context.Context) ([]APClientInfo, error) {
	return m.clients, m.err
}

func TestAPClientInfo_Construction(t *testing.T) {
	info := APClientInfo{
		MACAddress:    "aa:bb:cc:dd:ee:ff",
		Signal:        -65,
		SignalAverage: -67,
		Connected:     2 * time.Hour,
		Inactive:      500 * time.Millisecond,
		RxBitrate:     300_000_000,
		TxBitrate:     200_000_000,
		RxBytes:       1024 * 1024,
		TxBytes:       512 * 1024,
		InterfaceName: "wlan0",
		APBSSID:       "00:11:22:33:44:55",
		APSSID:        "TestNetwork",
	}

	if info.MACAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MACAddress = %q, want %q", info.MACAddress, "aa:bb:cc:dd:ee:ff")
	}
	if info.Signal != -65 {
		t.Errorf("Signal = %d, want -65", info.Signal)
	}
	if info.Connected != 2*time.Hour {
		t.Errorf("Connected = %v, want 2h", info.Connected)
	}
	if info.APBSSID != "00:11:22:33:44:55" {
		t.Errorf("APBSSID = %q, want %q", info.APBSSID, "00:11:22:33:44:55")
	}
	if info.APSSID != "TestNetwork" {
		t.Errorf("APSSID = %q, want %q", info.APSSID, "TestNetwork")
	}
}

func TestMockAPClientEnumerator_Available(t *testing.T) {
	e := &mockAPClientEnumerator{available: false}
	if e.Available() {
		t.Error("expected Available() = false")
	}

	e.available = true
	if !e.Available() {
		t.Error("expected Available() = true")
	}
}

func TestMockAPClientEnumerator_Enumerate_Empty(t *testing.T) {
	e := &mockAPClientEnumerator{available: true}
	clients, err := e.Enumerate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clients != nil {
		t.Errorf("expected nil clients, got %d", len(clients))
	}
}
