// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "time"

// AlertKind identifies the condition an Alert reports.
type AlertKind string

const (
	AlertKindAppUpdate        AlertKind = "APP_UPDATE"
	AlertKindIRCUnhealthy     AlertKind = "IRC_UNHEALTHY"
	AlertKindListRefreshError AlertKind = "LIST_REFRESH_ERROR"
)

func (k AlertKind) String() string {
	return string(k)
}

// AlertSeverity decides how prominently the web UI shows an Alert.
type AlertSeverity string

const (
	AlertSeverityInfo  AlertSeverity = "INFO"
	AlertSeverityError AlertSeverity = "ERROR"
)

func (s AlertSeverity) String() string {
	return string(s)
}

// Alert is a condition that holds right now and needs the user's attention. Unlike an
// inbox message it is not stored: it is derived from live state and goes away once the
// condition clears.
type Alert struct {
	Kind      AlertKind     `json:"kind"`
	Severity  AlertSeverity `json:"severity"`
	SubjectID int64         `json:"subject_id,omitempty"`
	Subject   string        `json:"subject"`
	Message   string        `json:"message,omitempty"`
	URL       string        `json:"url,omitempty"`
	Since     time.Time     `json:"since,omitzero"`
}
