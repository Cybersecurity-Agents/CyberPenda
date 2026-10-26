//go:build !linux

package blackboardv2

import "os"

func evidenceChangeTime(_ os.FileInfo) (int64, int64, bool) { return 0, 0, false }
