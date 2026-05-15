//go:build !windows

package scheduler

import (
	"os"
	"strconv"
	"syscall"
)

// isOwnedByCurrentUser checks if a file is owned by the current user on Unix systems
func isOwnedByCurrentUser(userID string, fileInfo os.FileInfo) bool {
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		fileUID := stat.Uid

		if uidInt, err := strconv.ParseUint(userID, 10, 32); err == nil {
			return uint32(uidInt) == fileUID
		}
	}

	return false
}
