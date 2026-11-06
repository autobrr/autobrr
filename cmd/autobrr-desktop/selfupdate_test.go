// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pkgversion "github.com/autobrr/autobrr/pkg/version"
)

const (
	testArchive  = "autobrr-desktop_1.91.0_windows_x86_64.zip"
	testChecksum = "autobrr_1.91.0_checksums.txt"
)

func zipWith(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// releaseServer serves assets by name and returns a release that lists them.
func releaseServer(t *testing.T, assets map[string][]byte) *pkgversion.Release {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := assets[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	rel := &pkgversion.Release{TagName: "v1.91.0"}
	for name := range assets {
		rel.Assets = append(rel.Assets, pkgversion.Asset{Name: name, BrowserDownloadUrl: srv.URL + "/" + name})
	}

	return rel
}

func installedExe(t *testing.T) string {
	t.Helper()

	exe := filepath.Join(t.TempDir(), updateExeName)
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	return exe
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", filepath.Base(path), got, want)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists, want it gone (err=%v)", filepath.Base(path), err)
	}
}

func TestSelfUpdaterInstall(t *testing.T) {
	archive := zipWith(t, map[string]string{"LICENSE": "license", "README.md": "readme", updateExeName: "new"})
	checksums := fmt.Sprintf("%s  autobrr_1.91.0_windows_x86_64.zip\n%s  %s\n", sha([]byte("other")), sha(archive), testArchive)

	rel := releaseServer(t, map[string][]byte{testArchive: archive, testChecksum: []byte(checksums)})
	exe := installedExe(t)

	// a backup from an earlier update is replaced
	if err := os.WriteFile(exe+".old", []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := newSelfUpdater(exe).install(context.Background(), rel); err != nil {
		t.Fatal(err)
	}

	assertFile(t, exe, "new")
	assertFile(t, exe+".old", "old")
	assertMissing(t, exe+".new")
}

func TestSelfUpdaterInstallLeavesExeOnFailure(t *testing.T) {
	archive := zipWith(t, map[string]string{updateExeName: "new"})

	tests := []struct {
		name    string
		assets  map[string][]byte
		wantErr string
	}{
		{
			name:    "checksum mismatch",
			assets:  map[string][]byte{testArchive: archive, testChecksum: []byte(sha([]byte("tampered")) + "  " + testArchive + "\n")},
			wantErr: "checksum mismatch",
		},
		{
			name:    "archive not in checksums",
			assets:  map[string][]byte{testArchive: archive, testChecksum: []byte(sha(archive) + "  something-else.zip\n")},
			wantErr: "no checksum for " + testArchive,
		},
		{
			name:    "no desktop archive",
			assets:  map[string][]byte{testChecksum: []byte(sha(archive) + "  " + testArchive + "\n")},
			wantErr: "has no asset " + testArchive,
		},
		{
			name:    "no checksums",
			assets:  map[string][]byte{testArchive: archive},
			wantErr: "has no asset " + testChecksum,
		},
		{
			name: "exe missing from archive",
			assets: func() map[string][]byte {
				other := zipWith(t, map[string]string{"autobrr.exe": "server"})
				return map[string][]byte{testArchive: other, testChecksum: []byte(sha(other) + "  " + testArchive + "\n")}
			}(),
			wantErr: "no " + updateExeName + " in archive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel := releaseServer(t, tt.assets)
			exe := installedExe(t)

			err := newSelfUpdater(exe).install(context.Background(), rel)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("install error = %v, want %q", err, tt.wantErr)
			}

			assertFile(t, exe, "old")
			assertMissing(t, exe+".new")
			assertMissing(t, exe+".old")
		})
	}
}

func TestSelfUpdaterSwapRestoresExe(t *testing.T) {
	exe := installedExe(t)

	u := newSelfUpdater(exe)
	u.rename = func(oldpath, newpath string) error {
		if oldpath == exe+".new" {
			return os.ErrPermission
		}
		return os.Rename(oldpath, newpath)
	}

	if err := u.swap([]byte("new")); err == nil {
		t.Fatal("swap succeeded, want an error")
	}

	assertFile(t, exe, "old")
	assertMissing(t, exe+".new")
	assertMissing(t, exe+".old")
}

func TestSelfUpdaterSwapAfterClose(t *testing.T) {
	exe := installedExe(t)

	u := newSelfUpdater(exe)
	u.close()

	if err := u.swap([]byte("new")); !errors.Is(err, errShuttingDown) {
		t.Fatalf("swap error = %v, want errShuttingDown", err)
	}

	assertFile(t, exe, "old")
	assertMissing(t, exe+".new")
}
