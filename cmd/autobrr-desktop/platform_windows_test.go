// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// A named mutex reports ERROR_ALREADY_EXISTS to its own process too, so a
// second lockInstance call stands in for a second instance.
func TestLockInstance(t *testing.T) {
	dir := t.TempDir()

	unlock, locked, err := lockInstance(dir)
	if err != nil || !locked {
		t.Fatalf("first lock: locked=%v err=%v", locked, err)
	}

	for _, same := range []string{dir, strings.ToUpper(dir), dir + `\`, filepath.Join(dir, "sub", "..")} {
		_, locked, err := lockInstance(same)
		if err != nil || locked {
			t.Errorf("lockInstance(%q): locked=%v err=%v, want already running", same, locked, err)
		}
	}

	otherUnlock, locked, err := lockInstance(t.TempDir())
	if err != nil || !locked {
		t.Errorf("other config dir: locked=%v err=%v, want its own lock", locked, err)
	} else {
		otherUnlock()
	}

	// a restart unlocks before the new process locks
	unlock()

	unlock, locked, err = lockInstance(dir)
	if err != nil || !locked {
		t.Fatalf("lock after unlock: locked=%v err=%v", locked, err)
	}
	unlock()
}

func TestLoginCommand(t *testing.T) {
	got := loginCommand(`C:\Program Files\autobrr\autobrr-desktop.exe`, `C:\Users\a b\AppData\Roaming\autobrr`)
	want := `"C:\Program Files\autobrr\autobrr-desktop.exe" --config "C:\Users\a b\AppData\Roaming\autobrr"`
	if got != want {
		t.Fatalf("loginCommand = %s, want %s", got, want)
	}

	args, err := windows.DecomposeCommandLine(got)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 3 || args[2] != `C:\Users\a b\AppData\Roaming\autobrr` {
		t.Fatalf("command line splits into %q", args)
	}
}

// Writes real entries under HKCU, named per process and removed afterwards.
func TestStartOnLogin(t *testing.T) {
	name := fmt.Sprintf("autobrr-test-%d", os.Getpid())
	command := loginCommand(`C:\autobrr\autobrr-desktop.exe`, `C:\autobrr\config`)

	t.Cleanup(func() {
		_ = setStartOnLogin(name, command, false)
	})

	assertStarts := func(want bool) {
		t.Helper()

		got, err := startsOnLogin(name, command)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("startsOnLogin = %v, want %v", got, want)
		}
	}

	assertStarts(false)

	if err := setStartOnLogin(name, command, true); err != nil {
		t.Fatal(err)
	}
	assertStarts(true)

	other := loginCommand(`D:\old\autobrr-desktop.exe`, `C:\autobrr\config`)
	if got, _ := startsOnLogin(name, other); got {
		t.Fatal("an entry for another exe counts as on")
	}

	approved, _, err := registry.CreateKey(registry.CURRENT_USER, startupApprovedKey, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer approved.Close()

	if err := approved.SetBinaryValue(name, []byte{3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	assertStarts(false)

	if err := setStartOnLogin(name, command, true); err != nil {
		t.Fatal(err)
	}
	assertStarts(true)

	if err := setStartOnLogin(name, command, false); err != nil {
		t.Fatal(err)
	}
	assertStarts(false)

	if err := setStartOnLogin(name, command, false); err != nil {
		t.Fatalf("disabling twice: %v", err)
	}
}

// Writes a real value under HKCU, named per process and removed afterwards.
func TestDesktopSetting(t *testing.T) {
	name := fmt.Sprintf("autobrr-test-%d", os.Getpid())

	t.Cleanup(func() {
		if k, err := registry.OpenKey(registry.CURRENT_USER, desktopSettingsKey, registry.SET_VALUE); err == nil {
			_ = k.DeleteValue(name)
			k.Close()
		}
	})

	assertSetting := func(want bool) {
		t.Helper()

		got, err := desktopSetting(name)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("desktopSetting = %v, want %v", got, want)
		}
	}

	assertSetting(false)

	for _, on := range []bool{true, false, true} {
		if err := setDesktopSetting(name, on); err != nil {
			t.Fatal(err)
		}
		assertSetting(on)
	}
}

func TestLockUpdate(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "autobrr-desktop.exe")

	unlock, locked, err := lockUpdate(exe)
	if err != nil || !locked {
		t.Fatalf("first lock: locked=%v err=%v", locked, err)
	}

	if _, locked, err := lockUpdate(strings.ToUpper(exe)); err != nil || locked {
		t.Fatalf("second lock: locked=%v err=%v, want held", locked, err)
	}

	// the instance lock for a config dir at the same path is a different mutex
	instanceUnlock, locked, err := lockInstance(exe)
	if err != nil || !locked {
		t.Fatalf("instance lock: locked=%v err=%v", locked, err)
	}
	instanceUnlock()

	unlock()

	unlock, locked, err = lockUpdate(exe)
	if err != nil || !locked {
		t.Fatalf("lock after unlock: locked=%v err=%v", locked, err)
	}
	unlock()
}
