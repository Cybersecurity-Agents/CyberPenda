//go:build linux

package blackboardv2

import (
	"os"
	"syscall"
)

func evidenceChangeTime(info os.FileInfo) (int64, int64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec), true
}
