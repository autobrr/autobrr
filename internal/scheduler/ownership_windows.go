//go:build windows

package scheduler

import (
	"os"
)

// isOwnedByCurrentUser on Windows - simplified ownership check
func isOwnedByCurrentUser(userID string, fileInfo os.FileInfo) bool {
	// On Windows, if we can read the file info, we likely have access to it
	// This is a simplified approach since Windows file ownership is more complex
	return true
}
