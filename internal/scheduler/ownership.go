//go:build !windows

package scheduler

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// isOwnedByCurrentUser checks if a file is owned by the current user on Unix systems
func isOwnedByCurrentUser(currentUser *user.User, fileInfo os.FileInfo) bool {
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		fileUID := stat.Uid
		currentUID := currentUser.Uid

		if uidInt, err := strconv.ParseUint(currentUID, 10, 32); err == nil {
			return uint32(uidInt) == fileUID
		}
	}

	return false
}
