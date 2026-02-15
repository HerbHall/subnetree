//go:build windows

package metrics

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"
)

var (
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modIPHelper = windows.NewLazySystemDLL("iphlpapi.dll")

	procGetSystemTimes      = modKernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = modKernel32.NewProc("GlobalMemoryStatusEx")
	procGetLogicalDrives    = modKernel32.NewProc("GetLogicalDrives")
	procGetIfTable          = modIPHelper.NewProc("GetIfTable")
)

// windowsCollector gathers system metrics using Windows syscalls.
type windowsCollector struct {
	logger *zap.Logger
}

// Compile-time guard.
var _ Collector = (*windowsCollector)(nil)

func newPlatformCollector(logger *zap.Logger) Collector {
	return &windowsCollector{logger: logger}
}

func (c *windowsCollector) Collect(ctx context.Context) (*scoutpb.SystemMetrics, error) {
	m := &scoutpb.SystemMetrics{}

	cpu, err := c.collectCPU(ctx)
	if err != nil {
		c.logger.Debug("cpu metrics failed", zap.Error(err))
	} else {
		m.CpuPercent = cpu
	}

	memPercent, memUsed, memTotal, err := c.collectMemory()
	if err != nil {
		c.logger.Debug("memory metrics failed", zap.Error(err))
	} else {
		m.MemoryPercent = memPercent
		m.MemoryUsedBytes = memUsed
		m.MemoryTotalBytes = memTotal
	}

	disks, err := c.collectDisks()
	if err != nil {
		c.logger.Debug("disk metrics failed", zap.Error(err))
	} else {
		m.Disks = disks
	}

	networks, err := c.collectNetwork()
	if err != nil {
		c.logger.Debug("network metrics failed", zap.Error(err))
	} else {
		m.Networks = networks
	}

	// Collect Docker container stats (graceful degradation if Docker unavailable).
	containerStats := collectDockerStats(ctx, c.logger)
	if len(containerStats) > 0 {
		m.ContainerStats = containerStats
		c.logger.Debug("collected docker container stats", zap.Int("count", len(containerStats)))
	}

	return m, nil
}

// filetime holds a Windows FILETIME (100-nanosecond intervals since 1601-01-01).
type filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

func (ft filetime) toUint64() uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

func (c *windowsCollector) collectCPU(ctx context.Context) (float64, error) {
	var idle1, kernel1, user1 filetime
	ret, _, err := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle1)),
		uintptr(unsafe.Pointer(&kernel1)),
		uintptr(unsafe.Pointer(&user1)),
	)
	if ret == 0 {
		return 0, fmt.Errorf("GetSystemTimes (first call): %w", err)
	}

	// Wait for a short interval to compute delta.
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(200 * time.Millisecond):
	}

	var idle2, kernel2, user2 filetime
	ret, _, err = procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle2)),
		uintptr(unsafe.Pointer(&kernel2)),
		uintptr(unsafe.Pointer(&user2)),
	)
	if ret == 0 {
		return 0, fmt.Errorf("GetSystemTimes (second call): %w", err)
	}

	idleDelta := idle2.toUint64() - idle1.toUint64()
	kernelDelta := kernel2.toUint64() - kernel1.toUint64()
	userDelta := user2.toUint64() - user1.toUint64()

	// Kernel time includes idle time.
	totalDelta := kernelDelta + userDelta
	if totalDelta == 0 {
		return 0, nil
	}

	// CPU usage = (total - idle) / total * 100.
	cpuPercent := float64(totalDelta-idleDelta) / float64(totalDelta) * 100.0
	if cpuPercent < 0 {
		cpuPercent = 0
	}
	if cpuPercent > 100 {
		cpuPercent = 100
	}
	return cpuPercent, nil
}

// memoryStatusEx matches the MEMORYSTATUSEX struct layout.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func (c *windowsCollector) collectMemory() (percent, used, total float64, err error) {
	var memStatus memoryStatusEx
	memStatus.Length = uint32(unsafe.Sizeof(memStatus))

	ret, _, callErr := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return 0, 0, 0, fmt.Errorf("GlobalMemoryStatusEx: %w", callErr)
	}

	total = float64(memStatus.TotalPhys)
	used = float64(memStatus.TotalPhys - memStatus.AvailPhys)
	if total > 0 {
		percent = used / total * 100.0
	}
	return percent, used, total, nil
}

func (c *windowsCollector) collectDisks() ([]*scoutpb.DiskMetric, error) {
	ret, _, err := procGetLogicalDrives.Call()
	if ret == 0 {
		return nil, fmt.Errorf("GetLogicalDrives: %w", err)
	}

	bitmask := uint32(ret)
	var disks []*scoutpb.DiskMetric

	for i := uint(0); i < 26; i++ {
		if bitmask&(1<<i) == 0 {
			continue
		}

		driveLetter := string(rune('A' + i))
		rootPath := driveLetter + `:\`
		rootPathUTF16, utfErr := windows.UTF16PtrFromString(rootPath)
		if utfErr != nil {
			continue
		}

		var freeBytesAvailable, totalBytes, totalFreeBytes uint64
		gdfErr := windows.GetDiskFreeSpaceEx(
			rootPathUTF16,
			&freeBytesAvailable,
			&totalBytes,
			&totalFreeBytes,
		)
		if gdfErr != nil {
			// Skip drives that can't be queried (CD-ROM, disconnected network drives).
			continue
		}

		if totalBytes == 0 {
			continue
		}

		disks = append(disks, &scoutpb.DiskMetric{
			MountPoint: rootPath,
			TotalBytes: float64(totalBytes),
			UsedBytes:  float64(totalBytes - totalFreeBytes),
			FreeBytes:  float64(totalFreeBytes),
		})
	}

	return disks, nil
}

// mibIfRow matches the MIB_IFROW struct (relevant fields).
// Full struct size is 860 bytes on Windows.
type mibIfRow struct {
	Name            [256]uint16
	Index           uint32
	Type            uint32
	Mtu             uint32
	Speed           uint32
	PhysAddrLen     uint32
	PhysAddr        [8]byte // MAXLEN_PHYSADDR
	AdminStatus     uint32
	OperStatus      uint32
	LastChange      uint32
	InOctets        uint32
	InUcastPkts     uint32
	InNUcastPkts    uint32
	InDiscards      uint32
	InErrors        uint32
	InUnknownProtos uint32
	OutOctets       uint32
	OutUcastPkts    uint32
	OutNUcastPkts   uint32
	OutDiscards     uint32
	OutErrors       uint32
	OutQLen         uint32
	DescrLen        uint32
	Descr           [256]byte
}

// mibIfTable matches the MIB_IFTABLE struct header.
type mibIfTable struct {
	NumEntries uint32
	// Followed by NumEntries mibIfRow entries.
}

func (c *windowsCollector) collectNetwork() ([]*scoutpb.NetworkMetric, error) {
	// First call to get required buffer size.
	const errInsufficientBuffer = 122
	var bufSize uint32
	ret, _, _ := procGetIfTable.Call(0, uintptr(unsafe.Pointer(&bufSize)), 0)
	if ret != errInsufficientBuffer {
		return nil, fmt.Errorf("GetIfTable size query returned %d", ret)
	}

	buf := make([]byte, bufSize)
	ret, _, err := procGetIfTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufSize)),
		0,
	)
	if ret != 0 {
		return nil, fmt.Errorf("GetIfTable: %w (ret=%d)", err, ret)
	}

	table := (*mibIfTable)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(mibIfRow{})
	var networks []*scoutpb.NetworkMetric

	for i := uint32(0); i < table.NumEntries; i++ {
		offset := unsafe.Sizeof(mibIfTable{}) + uintptr(i)*rowSize
		if offset+rowSize > uintptr(len(buf)) {
			break
		}
		row := (*mibIfRow)(unsafe.Pointer(&buf[offset]))

		// Only include operational interfaces.
		if row.OperStatus != 1 { // MIB_IF_OPER_STATUS_OPERATIONAL
			continue
		}

		ifName := windows.UTF16ToString(row.Name[:])
		if ifName == "" {
			ifName = fmt.Sprintf("interface_%d", row.Index)
		}

		networks = append(networks, &scoutpb.NetworkMetric{
			InterfaceName: ifName,
			BytesSent:     float64(row.OutOctets),
			BytesRecv:     float64(row.InOctets),
		})
	}

	return networks, nil
}
