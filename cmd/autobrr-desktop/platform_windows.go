// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

package main

import (
	"unsafe"

	"github.com/autobrr/autobrr/pkg/errors"

	"golang.org/x/sys/windows"
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
	wmQueryEndSession = 0x0011
	wmEndSession      = 0x0016
)

// showError is the only way a GUI build can report a failed start, it has no
// console to print to.
func showError(title, message string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(message)

	_, _, _ = procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbIconError)
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
