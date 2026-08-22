// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package harness

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AnnounceChannel is the channel test/definitions/mock.yaml tells autobrr to
// monitor on the mock indexer's network.
const AnnounceChannel = "#announces"

// ircNetwork is the part of the /api/irc response the harness cares about.
type ircNetwork struct {
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	Connected        bool   `json:"connected"`
	ConnectionErrors []string
	Channels         []struct {
		Name       string `json:"name"`
		Monitoring bool   `json:"monitoring"`
		State      string `json:"state"`
	} `json:"channels"`
}

// WaitForChannelMonitoring blocks until autobrr reports it is monitoring
// channel, which means it has connected, registered and joined.
//
// Announcing before that point is the classic e2e race: the line goes out, no
// one is in the channel to hear it, and the test fails somewhere far away from
// the cause. Asking autobrr for its own view of the connection is both faster
// and more honest than sleeping.
func (a *App) WaitForChannelMonitoring(channel string, timeout time.Duration) {
	a.t.Helper()

	deadline := time.Now().Add(timeout)

	var last string

	for time.Now().Before(deadline) {
		networks, err := a.ircNetworks()
		if err != nil {
			last = err.Error()
		} else {
			for _, network := range networks {
				for _, ch := range network.Channels {
					if !strings.EqualFold(ch.Name, channel) {
						continue
					}

					if ch.Monitoring {
						return
					}

					last = fmt.Sprintf("channel %s state %q, network connected=%t errors=%v",
						ch.Name, ch.State, network.Connected, network.ConnectionErrors)
				}
			}

			if last == "" {
				last = fmt.Sprintf("channel %s not present in %d networks", channel, len(networks))
			}
		}

		time.Sleep(250 * time.Millisecond)
	}

	a.t.Fatalf("timed out waiting for autobrr to monitor %s: %s\n%s", channel, last, a.Logs())
}

func (a *App) ircNetworks() ([]ircNetwork, error) {
	resp, err := a.client.Get(a.BaseURL + "/api/irc")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /api/irc returned %d", resp.StatusCode)
	}

	var networks []ircNetwork
	if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
		return nil, fmt.Errorf("could not decode irc networks: %w", err)
	}

	return networks, nil
}
