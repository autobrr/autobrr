// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build darwin || linux

package main

// showError is a no-op outside Windows, where the error already reaches the
// terminal and the log.
func showError(title, message string) {}

// showInfo is a no-op outside Windows, see showError.
func showInfo(title, message string) {}

// lockInstance always succeeds outside Windows, where the desktop build is
// only used for development.
func lockInstance(configDir string) (unlock func(), locked bool, err error) {
	return func() {}, true, nil
}

// askYesNo answers no outside Windows, which has no dialog to ask with.
func askYesNo(title, message string) bool {
	return false
}

// onSessionEnd is a no-op outside Windows, where logout and shutdown send
// SIGTERM, which ShutdownContext already handles.
func onSessionEnd(fn func()) error {
	return nil
}

// loginCommand, startsOnLogin and setStartOnLogin back the Windows-only
// "Start on login" menu item, which the tray does not add elsewhere.
func loginCommand(exe, configDir string) string {
	return ""
}

func startsOnLogin(name, command string) (bool, error) {
	return false, nil
}

func setStartOnLogin(name, command string, enabled bool) error {
	return nil
}

// desktopSetting and setDesktopSetting back Windows-only tray settings.
func desktopSetting(name string) (bool, error) {
	return false, nil
}

func setDesktopSetting(name string, on bool) error {
	return nil
}
