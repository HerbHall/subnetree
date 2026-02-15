//go:build windows

package tier

import (
	"syscall"
	"unsafe"
)

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

func getSystemRAMBytes() uint64 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms)) //nolint:gosec // G103: unsafe needed for Windows syscall

	ret, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms))) //nolint:gosec // G103: unsafe needed for Windows syscall
	if ret == 0 {
		return 0 // fallback
	}

	return ms.TotalPhys
}
