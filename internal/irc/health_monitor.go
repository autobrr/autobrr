// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
)

const (
	healthCheckInterval = time.Minute

	// unhealthyGracePeriod outlasts a normal reconnect and rejoin, so routine
	// disconnects and netsplits never reach the user as an unhealthy notification.
	unhealthyGracePeriod = 15 * time.Minute
)

// networkHealth is a point-in-time health reading of one network.
type networkHealth struct {
	// tracked is false while the network is stopped without a recorded failure:
	// a restart or shutdown in progress, which says nothing about its health.
	tracked bool
	healthy bool
	// stopped is set when autobrr stopped the network on a fatal failure (ban,
	// auth failure, TLS error, flapping). It will not retry on its own.
	stopped bool
	reasons []string
}

// unhealthyEpisode tracks one continuous unhealthy stretch of a network.
type unhealthyEpisode struct {
	// handler ties the episode to one handler: disabling and re-enabling a network
	// between checks swaps the handler under the same id, which must start fresh.
	handler  *Handler
	since    time.Time
	notified bool
	reasons  []string
}

// reportedNetwork is a network in the published unhealthy snapshot. It keeps the handler so
// a reader can drop networks disabled or deleted since the last check.
type reportedNetwork struct {
	handler *Handler
	network domain.IrcUnhealthyNetwork
}

type networkHealthReport struct {
	name    string
	reasons []string
}

// healthStatus reads the network health for the unhealthy notification check.
func (h *Handler) healthStatus() networkHealth {
	h.m.RLock()
	stopped := h.clientState == ircStopped
	connectionErrors := slices.Clone(h.connectionErrors)
	h.m.RUnlock()

	// every fatal failure records its reason before it stops the network
	if stopped {
		return networkHealth{tracked: len(connectionErrors) > 0, stopped: true, reasons: connectionErrors}
	}

	if h.computeHealthy() {
		return networkHealth{tracked: true, healthy: true}
	}

	// Channel errors are cleared on disconnect, so any left belong to this
	// connection. They also explain Error and PartiallyOperational, which a failed
	// announce channel join drives the connection into.
	connectionHealthy := h.stateMachine.IsHealthy()
	reasons := connectionErrors

	for _, channel := range h.channels.Iterator() {
		snap := channel.Snapshot()
		if !snap.Enabled || !snap.DefaultChannel {
			continue
		}

		if len(snap.ConnectionErrors) > 0 {
			reasons = append(reasons, fmt.Sprintf("%s: %s", snap.Name, strings.Join(snap.ConnectionErrors, "; ")))
			continue
		}

		if snap.Monitoring || !connectionHealthy {
			continue
		}

		state := "not monitoring"
		if sm := channel.StateMachine(); sm != nil {
			state = sm.CurrentState().String()
		}

		reasons = append(reasons, fmt.Sprintf("%s: %s", snap.Name, state))
	}

	if len(reasons) == 0 {
		reasons = []string{fmt.Sprintf("connection state: %s", h.stateMachine.GetState())}
	}

	return networkHealth{tracked: true, reasons: reasons}
}

func (s *Service) monitorHealth(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.checkNetworkHealth(ctx, time.Now())
		}
	}
}

// checkNetworkHealth emits one IRCUnhealthy event for every network that went
// unhealthy since the last check, and one IRCHealthy event for those that
// recovered after being reported. A network stopped on a fatal failure is
// reported right away; any other unhealthy network only once it has been
// unhealthy for unhealthyGracePeriod.
func (s *Service) checkNetworkHealth(ctx context.Context, now time.Time) {
	var unhealthy, recovered []networkHealthReport

	seen := make(map[int64]struct{})

	for id, handler := range s.networkHandlers.Iterator() {
		seen[id] = struct{}{}

		status := handler.healthStatus()
		if !status.tracked {
			continue
		}

		name := handler.GetNetwork().Name
		episode, found := s.unhealthyNetworks[id]
		if found && episode.handler != handler {
			delete(s.unhealthyNetworks, id)
			found = false
		}

		if status.healthy {
			if found {
				delete(s.unhealthyNetworks, id)

				if episode.notified {
					recovered = append(recovered, networkHealthReport{name: name})
				}
			}
			continue
		}

		if !found {
			episode = &unhealthyEpisode{handler: handler, since: now}
			s.unhealthyNetworks[id] = episode
		}

		episode.reasons = status.reasons

		if episode.notified || (!status.stopped && now.Sub(episode.since) < unhealthyGracePeriod) {
			continue
		}

		episode.notified = true
		unhealthy = append(unhealthy, networkHealthReport{name: name, reasons: status.reasons})
	}

	// a disabled or deleted network has no handler; drop its episode without a recovery
	for id := range s.unhealthyNetworks {
		if _, ok := seen[id]; !ok {
			delete(s.unhealthyNetworks, id)
		}
	}

	// published before the events go out, so a subscriber reading it sees this check
	s.publishUnhealthySnapshot()

	if len(unhealthy) > 0 {
		lines := make([]string, 0, len(unhealthy))
		for _, report := range sortReports(unhealthy) {
			lines = append(lines, fmt.Sprintf("%s: %s", report.name, strings.Join(report.reasons, "; ")))
		}

		s.log.Warn().Strs("networks", reportNames(unhealthy)).Msg("irc networks unhealthy")

		s.eventBus.EmitIRC(ctx, events.IRCEvent{
			Type:    events.IRCUnhealthy,
			Network: strings.Join(reportNames(unhealthy), ", "),
			State:   string(events.IRCUnhealthy),
			Message: strings.Join(lines, "\n"),
		})
	}

	if len(recovered) > 0 {
		names := strings.Join(reportNames(sortReports(recovered)), ", ")

		s.log.Info().Str("networks", names).Msg("irc networks healthy again")

		s.eventBus.EmitIRC(ctx, events.IRCEvent{
			Type:    events.IRCHealthy,
			Network: names,
			State:   string(events.IRCHealthy),
			Message: names,
		})
	}
}

func (s *Service) publishUnhealthySnapshot() {
	snapshot := make([]reportedNetwork, 0, len(s.unhealthyNetworks))
	for id, episode := range s.unhealthyNetworks {
		if !episode.notified {
			continue
		}

		snapshot = append(snapshot, reportedNetwork{
			handler: episode.handler,
			network: domain.IrcUnhealthyNetwork{
				ID:      id,
				Name:    episode.handler.GetNetwork().Name,
				Reasons: episode.reasons,
				Since:   episode.since,
			},
		})
	}

	slices.SortFunc(snapshot, func(a, b reportedNetwork) int {
		return cmp.Compare(a.network.Name, b.network.Name)
	})

	s.unhealthySnapshot.Store(&snapshot)
}

// UnhealthyNetworks returns the networks reported unhealthy by the last health check that
// have not recovered, so it agrees with the IRC unhealthy notification.
func (s *Service) UnhealthyNetworks() []domain.IrcUnhealthyNetwork {
	snapshot := s.unhealthySnapshot.Load()
	if snapshot == nil {
		return nil
	}

	networks := make([]domain.IrcUnhealthyNetwork, 0, len(*snapshot))
	for _, reported := range *snapshot {
		if handler, ok := s.networkHandlers.Get(reported.network.ID); !ok || handler != reported.handler {
			continue
		}

		networks = append(networks, reported.network)
	}

	return networks
}

func sortReports(reports []networkHealthReport) []networkHealthReport {
	slices.SortFunc(reports, func(a, b networkHealthReport) int {
		return cmp.Compare(a.name, b.name)
	})

	return reports
}

func reportNames(reports []networkHealthReport) []string {
	names := make([]string, 0, len(reports))
	for _, report := range reports {
		names = append(names, report.name)
	}

	return names
}
