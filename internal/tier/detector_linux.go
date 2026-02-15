//go:build linux

package tier

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func getSystemRAMBytes() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0
			}
			kb, parseErr := strconv.ParseUint(fields[1], 10, 64)
			if parseErr != nil {
				return 0
			}
			return kb * 1024 // Convert kB to bytes
		}
	}

	return 0
}
