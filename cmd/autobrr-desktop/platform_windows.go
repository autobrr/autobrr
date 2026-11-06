// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/autobrr/autobrr/pkg/errors"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procMessageBoxW      = user32.NewProc("MessageBoxW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
)

const (
	mbIconError       = 0x10
	mbIconInformation = 0x40
	wmQueryEndSession = 0x0011
	wmEndSession      = 0x0016
)

// showError is the only way a GUI build can report a failed start, it has no
// console to print to.
func showError(title, message string) {
	showMessage(title, message, mbIconError)
}

func showInfo(title, message string) {
	showMessage(title, message, mbIconInformation)
}

func showMessage(title, message string, icon uintptr) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(message)

	_, _, _ = procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), icon)
}

// lockInstance reports false when another instance already runs with
// configDir. The named mutex is scoped to the config directory, so instances
// with separate configs can still run side by side, and to the login session
// through the Local namespace. Windows releases it when the process exits, so
// a crash cannot leave it stale.
func lockInstance(configDir string) (bool, error) {
	dir, err := filepath.Abs(configDir)
	if err != nil {
		return false, errors.Wrap(err, "could not resolve config directory")
	}

	// mutex names cannot contain backslashes, and Windows paths are case-insensitive
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(dir))))

	name, err := windows.UTF16PtrFromString(`Local\autobrr-` + hex.EncodeToString(sum[:16]))
	if err != nil {
		return false, err
	}

	// the handle is deliberately never closed: holding it until exit is the lock
	if _, err := windows.CreateMutex(nil, false, name); err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			return false, nil
		}

		return false, errors.Wrap(err, "could not create instance mutex")
	}

	return true, nil
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       windows.Handle
	cursor     windows.Handle
	background windows.Handle
	menuName   *uint16
	className  *uint16
	iconSm     windows.Handle
}

// onSessionEnd calls fn when Windows logs the user off or shuts down. A GUI
// process gets no console control events, so os/signal never fires; the only
// notice is WM_ENDSESSION on a top-level window, and Windows ends the process
// as soon as the handler returns, so fn has to block until shutdown is done.
//
// The window must be created on the thread that runs the tray message loop,
// which dispatches its messages. It is hidden rather than message-only,
// because message-only windows do not receive the session broadcast.
func onSessionEnd(fn func()) error {
	className, err := windows.UTF16PtrFromString("autobrr-session")
	if err != nil {
		return err
	}

	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return errors.Wrap(err, "could not get module handle")
	}

	wc := wndClassEx{
		wndProc: windows.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
			switch msg {
			case wmQueryEndSession:
				return 1
			case wmEndSession:
				if wParam != 0 {
					fn()
				}

				return 0
			}

			ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)

			return ret
		}),
		instance:  instance,
		className: className,
	}
	wc.size = uint32(unsafe.Sizeof(wc))

	if ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); ret == 0 {
		return errors.Wrap(err, "could not register session window class")
	}

	if hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, 0, 0, uintptr(instance), 0); hwnd == 0 {
		return errors.Wrap(err, "could not create session window")
	}

	return nil
}

const (
	runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

	// Settings > Apps > Startup and Task Manager disable a Run entry here,
	// without touching the Run value itself
	startupApprovedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
)

// loginCommand is the command line Windows runs at login. The config directory
// is always passed, so the entry keeps working if the default ever changes.
func loginCommand(exe, configDir string) string {
	return windows.EscapeArg(exe) + " --config " + windows.EscapeArg(configDir)
}

// startsOnLogin reports whether the Run entry name launches command and is not
// disabled in Windows settings. An entry for another path, such as an older
// copy of the exe, counts as off, so enabling it again points it here.
func startsOnLogin(name, command string) (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}

		return false, errors.Wrap(err, "could not open run key")
	}
	defer k.Close()

	value, _, err := k.GetStringValue(name)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}

		return false, errors.Wrap(err, "could not read run entry")
	}

	if !strings.EqualFold(value, command) {
		return false, nil
	}

	return !disabledAtStartup(name), nil
}

func disabledAtStartup(name string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, startupApprovedKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	data, _, err := k.GetBinaryValue(name)
	if err != nil || len(data) == 0 {
		return false
	}

	// Windows writes 02 or 06 to enable the entry and 03 or 07 to disable it
	return data[0]&1 == 1
}

// setStartOnLogin adds or removes the Run entry name. It also clears a
// disabled state left in Windows settings, which would otherwise keep a new
// entry from running.
func setStartOnLogin(name, command string, enabled bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return errors.Wrap(err, "could not open run key")
	}
	defer k.Close()

	if enabled {
		if err := k.SetStringValue(name, command); err != nil {
			return errors.Wrap(err, "could not write run entry")
		}
	} else if err := k.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		return errors.Wrap(err, "could not delete run entry")
	}

	approved, err := registry.OpenKey(registry.CURRENT_USER, startupApprovedKey, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}

		return errors.Wrap(err, "could not open startup approved key")
	}
	defer approved.Close()

	if err := approved.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		return errors.Wrap(err, "could not clear startup approved entry")
	}

	return nil
}
