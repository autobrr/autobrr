// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package icons embeds the icons shipped with the Windows desktop build. It is
// a package so cmd/autobrr-desktop can embed files that live outside its own
// directory, which go:embed does not allow.
package icons

import _ "embed"

// TrayLight is the autobrr logo shown in the system tray.
//
//go:embed autobrr-light-32.png
var TrayLight []byte

//go:embed autobrr-white-32.png
var TrayDark []byte
