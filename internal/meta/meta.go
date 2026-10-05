// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package meta

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	UserAgent string `json:"user_agent"`
}

const (
	devVersion      = "dev"
	unknownValue    = "unknown"
	commitSHALength = 7
)

var (
	version = devVersion
	commit  = unknownValue
	date    = unknownValue
)

// Set records the build info that packagers inject into package main with
// -X main.version, main.commit and main.date. That ldflags contract is shared with
// goreleaser and the distro recipes, so the targets stay in main and are forwarded here.
func Set(v, c, d string) {
	if v != "" {
		version = v
	}
	if c != "" {
		commit = c
	}
	if d != "" {
		date = d
	}
}

func GetVersion() string {
	if version != devVersion {
		if !strings.HasPrefix(version, "v") {
			return "v" + version
		}

		return version
	}

	// `go install module@version` builds from the module cache without ldflags, so the
	// module version is the only source. Builds from a checkout carry vcs.* settings and
	// a pseudo-version, and stay "dev" since the logger and update checker key off it.
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return devVersion
	}

	for _, setting := range info.Settings {
		if strings.HasPrefix(setting.Key, "vcs.") {
			return devVersion
		}
	}

	return info.Main.Version
}

// GetCommit returns the short SHA, or the value as given when it is not a hex SHA
// (Alpine sets alpine-r<pkgrel>, Homebrew sets its tap owner).
func GetCommit() string {
	if len(commit) > commitSHALength && isHex(commit) {
		return commit[:commitSHALength]
	}

	return commit
}

func isHex(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}

	return true
}

func GetDate() string {
	if date != "" && date != unknownValue {
		if t, err := time.Parse(time.RFC3339, date); err == nil {
			return t.Format("2006-01-02")
		}
	}

	return time.Now().Format("2006-01-02")
}

func GetUserAgent() string {
	return fmt.Sprintf("autobrr/%s (%s/%s)", GetVersion(), runtime.GOOS, runtime.GOARCH)
}

func GetMetaStr() string {
	v := GetVersion()
	d := GetDate()
	c := GetCommit()

	if c == unknownValue {
		return fmt.Sprintf("%s (Built on %s)", v, d)
	}

	return fmt.Sprintf("%s (Built on %s from Git SHA %s)", v, d, c)
}

func GetMetaInfo() Info {
	return Info{
		Version: GetVersion(),
		Date:    GetDate(),
		Commit:  GetCommit(),
	}
}
