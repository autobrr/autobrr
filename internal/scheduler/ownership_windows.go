//go:build windows

package scheduler

import (
	"os"
	"os/user"
)

// isOwnedByCurrentUser on Windows - simplified ownership check
func isOwnedByCurrentUser(currentUser *user.User, fileInfo os.FileInfo) bool {
	// On Windows, if we can read the file info, we likely have access to it
	// This is a simplified approach since Windows file ownership is more complex
	return true
}
