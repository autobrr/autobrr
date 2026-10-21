// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build darwin || linux

package main

// showError is a no-op outside Windows, where the error already reaches the
// terminal and the log.
func showError(title, message string) {}

// onSessionEnd is a no-op outside Windows, where logout and shutdown send
// SIGTERM, which ShutdownContext already handles.
func onSessionEnd(fn func()) error {
	return nil
}
