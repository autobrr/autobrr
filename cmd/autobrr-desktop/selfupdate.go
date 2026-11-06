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
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
	pkgversion "github.com/autobrr/autobrr/pkg/version"
)

const (
	// names from the archives and checksum sections of .goreleaser.yml
	updateArchiveName  = "autobrr-desktop_%s_windows_x86_64.zip"
	updateChecksumName = "autobrr_%s_checksums.txt"
	updateExeName      = "autobrr-desktop.exe"

	maxUpdateDownload = 256 << 20
)

// errShuttingDown is returned by an install that starts after close.
var errShuttingDown = errors.New("autobrr is shutting down")

// selfUpdater replaces exe with the one from a release's desktop zip.
type selfUpdater struct {
	client *http.Client
	exe    string

	// rename is os.Rename, replaced in tests to fail the second rename of a swap
	rename func(oldpath, newpath string) error

	// held during a swap; close takes it, so the process never exits between
	// the two renames, which would leave no exe at the installed path
	mu     sync.Mutex
	closed bool
}

func newSelfUpdater(exe string) *selfUpdater {
	return &selfUpdater{
		client: &http.Client{
			Timeout:   5 * time.Minute,
			Transport: sharedhttp.Transport,
		},
		exe:    exe,
		rename: os.Rename,
	}
}

// install downloads the desktop zip of rel, checks it against the release
// checksums and swaps its exe in. The running exe is not touched until the new
// one is verified and written next to it.
func (u *selfUpdater) install(ctx context.Context, rel *pkgversion.Release) error {
	ver := strings.TrimPrefix(rel.TagName, "v")
	archiveName := fmt.Sprintf(updateArchiveName, ver)
	checksumName := fmt.Sprintf(updateChecksumName, ver)

	archiveURL, err := assetURL(rel, archiveName)
	if err != nil {
		return err
	}

	checksumURL, err := assetURL(rel, checksumName)
	if err != nil {
		return err
	}

	checksums, err := u.download(ctx, checksumURL)
	if err != nil {
		return errors.Wrap(err, "could not download %s", checksumName)
	}

	want, err := checksumFor(checksums, archiveName)
	if err != nil {
		return err
	}

	archive, err := u.download(ctx, archiveURL)
	if err != nil {
		return errors.Wrap(err, "could not download %s", archiveName)
	}

	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return errors.New("checksum mismatch for %s: got %s, want %s", archiveName, got, want)
	}

	exe, err := extractExe(archive)
	if err != nil {
		return errors.Wrap(err, "could not read %s", archiveName)
	}

	return u.swap(exe)
}

func assetURL(rel *pkgversion.Release, name string) (string, error) {
	for _, asset := range rel.Assets {
		if asset.Name == name {
			return asset.BrowserDownloadUrl, nil
		}
	}

	return "", errors.New("release %s has no asset %s", rel.TagName, name)
}

func (u *selfUpdater) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer sharedhttp.DrainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxUpdateDownload+1))
	if err != nil {
		return nil, err
	}

	if len(data) > maxUpdateDownload {
		return nil, errors.New("download is larger than %d bytes", maxUpdateDownload)
	}

	return data, nil
}

// checksumFor reads the sha256sum format that GoReleaser writes: a hash and a
// file name per line.
func checksumFor(checksums []byte, name string) (string, error) {
	for line := range strings.Lines(string(checksums)) {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return strings.ToLower(fields[0]), nil
		}
	}

	return "", errors.New("no checksum for %s", name)
}

func extractExe(archive []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}

	for _, f := range r.File {
		if f.Name != updateExeName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		data, err := io.ReadAll(io.LimitReader(rc, maxUpdateDownload+1))
		if err != nil {
			return nil, err
		}

		if len(data) > maxUpdateDownload {
			return nil, errors.New("%s is larger than %d bytes", updateExeName, maxUpdateDownload)
		}

		return data, nil
	}

	return nil, errors.New("no %s in archive", updateExeName)
}

// close waits for a running swap and makes later ones fail with
// errShuttingDown. The shutdown path calls it before the process exits.
func (u *selfUpdater) close() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.closed = true
}

// swap moves exe aside instead of overwriting it, because Windows lets a
// running exe be renamed but not written. The previous version stays as
// <exe>.old for a manual rollback until the next update replaces it.
func (u *selfUpdater) swap(exe []byte) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.closed {
		return errShuttingDown
	}

	// instances with separate configs share the exe and its .new and .old files
	unlock, locked, err := lockUpdate(u.exe)
	if err != nil {
		return errors.Wrap(err, "could not lock exe for update")
	}
	if !locked {
		return errors.New("another autobrr is updating %s", u.exe)
	}
	defer unlock()

	next := u.exe + ".new"
	old := u.exe + ".old"

	if err := os.WriteFile(next, exe, 0o755); err != nil {
		return errors.Wrap(err, "could not write new exe")
	}
	defer func() {
		if err != nil {
			_ = os.Remove(next)
		}
	}()

	if err := os.Remove(old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "could not remove previous backup")
	}

	if err := u.rename(u.exe, old); err != nil {
		return errors.Wrap(err, "could not move current exe aside")
	}

	if err := u.rename(next, u.exe); err != nil {
		if rollbackErr := u.rename(old, u.exe); rollbackErr != nil {
			return errors.Wrap(err, "could not move new exe in place, and restoring %s failed: %v", old, rollbackErr)
		}

		return errors.Wrap(err, "could not move new exe in place")
	}

	return nil
}
