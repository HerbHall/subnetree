//go:build windows

package recon

import (
	"context"
	"encoding/binary"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"go.uber.org/zap"
)

var (
	wlanapi               = syscall.NewLazyDLL("wlanapi.dll")
	procWlanOpenHandle    = wlanapi.NewProc("WlanOpenHandle")
	procWlanCloseHandle   = wlanapi.NewProc("WlanCloseHandle")
	procWlanEnumInterfaces = wlanapi.NewProc("WlanEnumInterfaces")
	procWlanScan          = wlanapi.NewProc("WlanScan")
	procWlanGetNetworkBssList = wlanapi.NewProc("WlanGetNetworkBssList")
	procWlanFreeMemory    = wlanapi.NewProc("WlanFreeMemory")
)

// WLAN API constants.
const (
	wlanClientVersion2 = 2
	wlanBssTypeAny     = 3 // DOT11_BSS_TYPE_ANY
)

// wlanInterfaceInfo mirrors WLAN_INTERFACE_INFO (packed layout).
type wlanInterfaceInfo struct {
	InterfaceGUID  [16]byte // GUID
	Description    [512]byte // WCHAR[256] = 512 bytes
	State          uint32   // WLAN_INTERFACE_STATE
}

// wlanBssEntry mirrors the fixed-size prefix of WLAN_BSS_ENTRY.
// The variable-length IE data follows, but we only need the fixed fields.
type wlanBssEntry struct {
	SSID               [36]byte // DOT11_SSID: 4 bytes length + 32 bytes data
	PhyID              uint32
	BSSID              [6]byte  // DOT11_MAC_ADDRESS
	BssType            uint32   // DOT11_BSS_TYPE
	PhyType            uint32   // DOT11_PHY_TYPE
	RSSI               int32    // signal strength in dBm
	LinkQuality        uint32   // 0-100 percent
	InRegDomain        uint8
	BeaconPeriod       uint16
	Timestamp          uint64
	HostTimestamp       uint64
	CapabilityInfo     uint16
	ChCenterFrequency  uint32   // in kHz
	IEOffset           uint32
	IESize             uint32
}

type windowsWifiScanner struct {
	logger *zap.Logger
}

// NewWifiScanner returns a Windows WifiScanner backed by Wlanapi.dll.
func NewWifiScanner(logger *zap.Logger) WifiScanner {
	return &windowsWifiScanner{logger: logger}
}

// Available returns true if the WLAN API is accessible and at least one
// wireless interface exists.
func (s *windowsWifiScanner) Available() bool {
	if err := wlanapi.Load(); err != nil {
		s.logger.Debug("wlanapi.dll not available", zap.Error(err))
		return false
	}

	handle, err := wlanOpen()
	if err != nil {
		s.logger.Debug("WLAN open failed", zap.Error(err))
		return false
	}
	defer wlanClose(handle)

	guids, err := wlanEnumInterfaces(handle)
	if err != nil {
		s.logger.Debug("WLAN enum interfaces failed", zap.Error(err))
		return false
	}
	return len(guids) > 0
}

// Scan discovers nearby WiFi access points via the Windows WLAN API.
func (s *windowsWifiScanner) Scan(ctx context.Context) ([]AccessPointInfo, error) {
	handle, err := wlanOpen()
	if err != nil {
		return nil, fmt.Errorf("WLAN open: %w", err)
	}
	defer wlanClose(handle)

	guids, err := wlanEnumInterfaces(handle)
	if err != nil {
		return nil, fmt.Errorf("WLAN enum interfaces: %w", err)
	}
	if len(guids) == 0 {
		return nil, nil
	}

	// Use the first wireless interface.
	ifGUID := guids[0]

	// Trigger a scan. Ignore errors -- the BSS list may still have cached results.
	if scanErr := wlanTriggerScan(handle, ifGUID); scanErr != nil {
		s.logger.Debug("WLAN trigger scan failed, using cached results", zap.Error(scanErr))
	}

	// Brief pause to allow scan results to populate.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(150 * time.Millisecond):
	}

	entries, err := wlanGetBSSList(handle, ifGUID)
	if err != nil {
		return nil, fmt.Errorf("WLAN get BSS list: %w", err)
	}

	results := make([]AccessPointInfo, 0, len(entries))
	for _, e := range entries {
		bssid := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			e.BSSID[0], e.BSSID[1], e.BSSID[2],
			e.BSSID[3], e.BSSID[4], e.BSSID[5])

		ssidLen := binary.LittleEndian.Uint32(e.SSID[0:4])
		if ssidLen > 32 {
			ssidLen = 32
		}
		ssid := string(e.SSID[4 : 4+ssidLen])

		freqMHz := int(e.ChCenterFrequency) / 1000 // kHz to MHz

		ap := AccessPointInfo{
			BSSID:     bssid,
			SSID:      ssid,
			Frequency: freqMHz,
			Channel:   freqToChannel(freqMHz),
			Signal:    int(e.RSSI),
			Security:  inferSecurityFromCapability(e.CapabilityInfo),
		}
		results = append(results, ap)
	}

	return results, nil
}

// inferSecurityFromCapability uses the BSS capability info privacy bit and
// other available data to infer security.  The WLAN_BSS_ENTRY does not
// directly expose the auth algorithm, so we check the privacy capability
// bit (bit 4).  This is a rough heuristic -- WPA2 vs WPA3 cannot be
// distinguished from this bit alone.
func inferSecurityFromCapability(cap uint16) string {
	privacyBit := (cap >> 4) & 1
	if privacyBit == 0 {
		return "Open"
	}
	// Privacy bit is set but we cannot distinguish WPA2/WPA3 from the
	// capability field alone.  Return "WPA2" as the most common case.
	return "WPA2"
}

// wlanOpen opens a WLAN client handle.
func wlanOpen() (syscall.Handle, error) {
	var negotiated uint32
	var handle syscall.Handle
	ret, _, _ := procWlanOpenHandle.Call( //nolint:gosec // G115: WLAN API call
		uintptr(wlanClientVersion2),
		0, // reserved
		uintptr(unsafe.Pointer(&negotiated)),
		uintptr(unsafe.Pointer(&handle)),
	)
	if ret != 0 {
		return 0, fmt.Errorf("WlanOpenHandle returned %d", ret)
	}
	return handle, nil
}

// wlanClose closes a WLAN client handle.
func wlanClose(handle syscall.Handle) {
	procWlanCloseHandle.Call(uintptr(handle), 0) //nolint:gosec // G115: WLAN API call
}

// wlanEnumInterfaces returns the GUIDs of all wireless interfaces.
func wlanEnumInterfaces(handle syscall.Handle) ([][16]byte, error) {
	var listPtr unsafe.Pointer
	ret, _, _ := procWlanEnumInterfaces.Call( //nolint:gosec // G115: WLAN API call
		uintptr(handle),
		0, // reserved
		uintptr(unsafe.Pointer(&listPtr)),
	)
	if ret != 0 {
		return nil, fmt.Errorf("WlanEnumInterfaces returned %d", ret)
	}
	defer procWlanFreeMemory.Call(uintptr(listPtr)) //nolint:gosec // G115: freeing WLAN memory

	// WLAN_INTERFACE_INFO_LIST: first 4 bytes = dwNumberOfItems
	count := *(*uint32)(listPtr)
	if count == 0 {
		return nil, nil
	}

	// Interface entries start at offset 8 (4 bytes count + 4 bytes index).
	const infoSize = int(unsafe.Sizeof(wlanInterfaceInfo{}))

	guids := make([][16]byte, 0, count)
	for i := 0; i < int(count); i++ {
		entryPtr := unsafe.Add(listPtr, 8+i*infoSize)
		entry := (*wlanInterfaceInfo)(entryPtr)
		guids = append(guids, entry.InterfaceGUID)
	}
	return guids, nil
}

// wlanTriggerScan triggers a WiFi scan on the specified interface.
func wlanTriggerScan(handle syscall.Handle, ifGUID [16]byte) error {
	ret, _, _ := procWlanScan.Call( //nolint:gosec // G115: WLAN API call
		uintptr(handle),
		uintptr(unsafe.Pointer(&ifGUID)),
		0, 0, 0, // no SSID filter, no IE data
	)
	if ret != 0 {
		return fmt.Errorf("WlanScan returned %d", ret)
	}
	return nil
}

// wlanGetBSSList retrieves the list of BSS entries for the specified interface.
func wlanGetBSSList(handle syscall.Handle, ifGUID [16]byte) ([]wlanBssEntry, error) {
	var listPtr unsafe.Pointer
	ret, _, _ := procWlanGetNetworkBssList.Call( //nolint:gosec // G115: WLAN API call
		uintptr(handle),
		uintptr(unsafe.Pointer(&ifGUID)),
		0,                          // no SSID filter
		uintptr(wlanBssTypeAny),    // DOT11_BSS_TYPE
		0,                          // bSecurityEnabled = FALSE (all networks)
		0,                          // reserved
		uintptr(unsafe.Pointer(&listPtr)),
	)
	if ret != 0 {
		return nil, fmt.Errorf("WlanGetNetworkBssList returned %d", ret)
	}
	defer procWlanFreeMemory.Call(uintptr(listPtr)) //nolint:gosec // G115: freeing WLAN memory

	// WLAN_BSS_LIST: totalSize (4 bytes) + numberOfItems (4 bytes) + entries
	count := *(*uint32)(unsafe.Add(listPtr, 4))
	if count == 0 {
		return nil, nil
	}

	const entrySize = int(unsafe.Sizeof(wlanBssEntry{}))

	entries := make([]wlanBssEntry, 0, count)
	for i := 0; i < int(count); i++ {
		entryPtr := unsafe.Add(listPtr, 8+i*entrySize)
		entry := (*wlanBssEntry)(entryPtr)
		entries = append(entries, *entry)
	}
	return entries, nil
}
