//go:build !windows

package scheduler

import (
	"os"
	"syscall"
)

// isOwnedByCurrentUser checks if a file is owned by the current user on Unix systems
func isOwnedByCurrentUser(uid int, fileInfo os.FileInfo) bool {
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}

	return uint32(uid) == stat.Uid
}
