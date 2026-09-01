// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"fmt"
	"slices"
	"sync"

	"github.com/rs/zerolog"
)

// ConnectionState represents the current state of an IRC connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateAuthenticating
	StateAuthenticated
	StateJoiningChannels
	StateFullyOperational // All channels joined, all invites sent
	StatePartiallyOperational
	StateError
)

func (s ConnectionState) String() string {
	names := [...]string{
		"Disconnected",
		"Connecting",
		"Connected",
		"Authenticating",
		"Authenticated",
		"JoiningChannels",
		"FullyOperational",
		"PartiallyOperational",
		"Error",
	}

	if s < 0 || int(s) >= len(names) {
		return fmt.Sprintf("ConnectionState(%d)", int(s))
	}

	return names[s]
}

var validTransitions = map[ConnectionState][]ConnectionState{
	// StateError is reachable from Disconnected because the irc-go client auto-
	// reconnects from inside Loop() without going back through Handler.Run() (so
	// OnConnecting is never called and the SM stays Disconnected during the new
	// registration). A fatal in-band failure on that reconnect - e.g. a 465 G-Line -
	// calls OnError while still Disconnected; allowing the transition surfaces the
	// reason in real time instead of logging a spurious invalid-transition. Safe per
	// the StateError invariant below: every OnError caller also Stop()s.
	StateDisconnected:         {StateConnecting, StateConnected, StateError},
	StateConnecting:           {StateConnected, StateError, StateDisconnected},
	StateConnected:            {StateAuthenticating, StateAuthenticated, StatePartiallyOperational, StateError, StateDisconnected},
	StateAuthenticating:       {StateAuthenticated, StatePartiallyOperational, StateError, StateDisconnected},
	StateAuthenticated:        {StateJoiningChannels, StateFullyOperational, StatePartiallyOperational, StateError, StateDisconnected},
	StateJoiningChannels:      {StateFullyOperational, StatePartiallyOperational, StateError, StateDisconnected},
	StateFullyOperational:     {StatePartiallyOperational, StateError, StateDisconnected},
	StatePartiallyOperational: {StateFullyOperational, StateError, StateDisconnected},
	// Error is recoverable without a reconnect: when a channel that had failed
	// rejoins (updateOperationalState re-runs on OnChannelJoined), the network must
	// be able to climb back to operational. These channel-driven events only fire on
	// a live connection, so a genuine connection-level error still ends in a
	// disconnect/reconnect rather than a false recovery.
	//
	// INVARIANT this relies on: every ConnectionStateMachine.OnError caller must tear
	// the connection down (Handler.Stop()) so no channel JOIN can arrive afterwards
	// and falsely climb the network back out of a real connection-level error. All
	// current callers do (they are pre-authentication auth failures + Stop()). Do not
	// add an OnError path that leaves the socket live without also gating this climb.
	StateError: {StateDisconnected, StateConnecting, StateFullyOperational, StatePartiallyOperational},
}

type ConnectionStateMachine struct {
	m            sync.RWMutex
	currentState ConnectionState
	handler      *Handler
	log          zerolog.Logger

	// State tracking
	authAttempts int
}

func NewConnectionStateMachine(handler *Handler) *ConnectionStateMachine {
	return &ConnectionStateMachine{
		currentState: StateDisconnected,
		handler:      handler,
		log:          handler.log.With().Str("component", "handler-state").Logger(),
	}
}

// State transitions
func (sm *ConnectionStateMachine) transition(to ConnectionState) error {
	sm.m.Lock()
	defer sm.m.Unlock()

	from := sm.currentState

	// Validate transition
	if !sm.isValidTransition(from, to) {
		sm.log.Error().Str("from", from.String()).Str("to", to.String()).Msg("invalid state transition")
		return fmt.Errorf("invalid state transition from %s to %s", from, to)
	}

	sm.log.Debug().
		Str("from", from.String()).
		Str("to", to.String()).
		Msg("state transition")

	sm.currentState = to

	// Execute state entry actions
	go sm.onStateEntry(to)

	return nil
}

func (sm *ConnectionStateMachine) isValidTransition(from, to ConnectionState) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	return slices.Contains(allowed, to)
}

func (sm *ConnectionStateMachine) transitionIfNeeded(to ConnectionState) {
	sm.m.RLock()
	current := sm.currentState
	sm.m.RUnlock()

	if current == to {
		return
	}

	_ = sm.transition(to) // #nosec G104

}

func (sm *ConnectionStateMachine) updateOperationalState() {
	enabled := 0
	monitoring := 0
	errored := 0

	sm.handler.channels.ForEach(func(name string, ch *Channel) bool {
		snap := ch.Snapshot()
		// only announce (default) channels drive the operational state; user-added
		// extras are surfaced per-channel and must not flip the network unhealthy
		if !snap.Enabled || !snap.DefaultChannel {
			return true
		}

		enabled++

		if snap.Monitoring && len(snap.ConnectionErrors) == 0 {
			monitoring++
			return true
		}

		if len(snap.ConnectionErrors) > 0 {
			errored++
		}

		return true
	})

	if enabled == 0 {
		sm.transitionIfNeeded(StateFullyOperational)
		return
	}

	pending := enabled - monitoring - errored

	if pending > 0 {
		// Still waiting for additional channels to join
		return
	}

	if monitoring == enabled {
		sm.transitionIfNeeded(StateFullyOperational)
		return
	}

	if monitoring > 0 {
		sm.transitionIfNeeded(StatePartiallyOperational)
		return
	}

	// All enabled channels failed
	sm.transitionIfNeeded(StateError)
}

func (sm *ConnectionStateMachine) allEnabledChannelsMonitoring() bool {
	allJoined := true

	sm.handler.channels.ForEach(func(name string, ch *Channel) bool {
		snap := ch.Snapshot()
		if !snap.Enabled || !snap.DefaultChannel {
			return true
		}

		if !snap.Monitoring || len(snap.ConnectionErrors) > 0 {
			allJoined = false
			return false
		}

		return true
	})

	return allJoined
}

func (sm *ConnectionStateMachine) onStateEntry(state ConnectionState) {
	switch state {
	case StateConnecting:
		// the connect attempt itself is driven by the handler; no entry action

	case StateConnected:
		sm.m.Lock()
		sm.authAttempts = 0
		sm.m.Unlock()
		sm.handler.setConnectionStatus()

	case StateAuthenticating:
		sm.handleAuthentication()

	case StateAuthenticated:
		if err := sm.transition(StateJoiningChannels); err != nil {
			sm.log.Error().Err(err).Msg("failed to transition to joining channels")
		}
		return

	case StateJoiningChannels:
		sm.handleJoinChannels()
		// Channels are joining, wait for NAMES replies

	case StateFullyOperational:
		sm.log.Info().Msg("IRC connection fully operational")
		sm.cleanup()
		// a live operational connection means any prior network-wide error (ban,
		// auth failure) is stale - drop it so it doesn't stick, especially on
		// no-auth networks that never reach setAuthenticated
		sm.handler.clearConnectErrors()
		sm.handler.broadcastHealth()

	case StatePartiallyOperational:
		sm.log.Warn().Msg("IRC connection partially operational")
		sm.cleanup()
		// the network-level connection succeeded here too; remaining failures are
		// per-channel, so clear stale network-wide errors
		sm.handler.clearConnectErrors()
		sm.handler.broadcastHealth()

	case StateError:
		sm.handleError()
		// surface the network-level failure reason to the UI in real time; the
		// reason was recorded (addConnectError) before OnError drove us here
		sm.handler.broadcastHealth()

	case StateDisconnected:
		sm.cleanup()
		sm.handler.broadcastHealth()
	default:
		sm.log.Error().Str("state", state.String()).Msg("invalid state")
	}
}

func (sm *ConnectionStateMachine) handleAuthentication() {
	sm.handler.m.RLock()
	needsAuth := sm.handler.network.Auth.NickServEnabled() && !sm.handler.saslauthed
	sm.handler.m.RUnlock()

	if needsAuth {
		sm.log.Trace().Msg("sending NickServ authentication")
		if err := sm.handler.NickServIdentify(); err != nil {
			sm.log.Error().Err(err).Msg("failed to send NickServ identify")
		}
		// Wait for handleNickServ callback to call OnAuthenticated()
	} else {
		sm.OnAuthenticated()
	}
}

func (sm *ConnectionStateMachine) handleJoinChannels() {
	sm.log.Debug().Msg("joining channels")
	sm.handler.JoinChannels()
	// Wait for handleJoined callbacks to call OnChannelJoined()
}

func (sm *ConnectionStateMachine) cleanup() {
}

func (sm *ConnectionStateMachine) handleError() {
	// runs from onStateEntry in its own goroutine, concurrently with transition()
	// writing currentState, so read it under the lock rather than racing the write
	sm.m.RLock()
	state := sm.currentState
	sm.m.RUnlock()

	sm.log.Error().Str("state", state.String()).Msg("error state reached")
	sm.cleanup()
}

// Event handlers called by IRC callbacks

func (sm *ConnectionStateMachine) OnConnecting() {
	_ = sm.transition(StateConnecting) // #nosec G104

}

func (sm *ConnectionStateMachine) OnConnected() {
	_ = sm.transition(StateConnected) // #nosec G104

	// Determine next state based on auth requirements
	sm.handler.m.RLock()
	botMode := sm.handler.network.BotMode
	needsAuth := sm.handler.network.Auth.NickServEnabled() && !sm.handler.saslauthed
	sm.handler.m.RUnlock()

	if botMode && sm.handler.botModeSupported() {
		sm.handler.setBotMode()
		// Will transition to auth in handleMode callback
	} else if needsAuth {
		_ = sm.transition(StateAuthenticating) // #nosec G104

	} else {
		_ = sm.transition(StateAuthenticated) // #nosec G104

	}
}

func (sm *ConnectionStateMachine) OnAuthenticated() {
	sm.m.RLock()
	currentState := sm.currentState
	sm.m.RUnlock()

	if currentState == StateAuthenticating || currentState == StateConnected {
		_ = sm.transition(StateAuthenticated) // #nosec G104

	}
}

func (sm *ConnectionStateMachine) OnChannelJoined(channel string) {
	sm.updateOperationalState()

	if sm.allEnabledChannelsMonitoring() {
		sm.transitionIfNeeded(StateFullyOperational)
	}
}

func (sm *ConnectionStateMachine) OnChannelError(channel, reason string) {
	sm.log.Error().
		Str("channel", channel).
		Str("reason", reason).
		Msg("channel reported connection issue")

	sm.updateOperationalState()
}

func (sm *ConnectionStateMachine) OnError(reason string) {
	sm.m.Lock()
	if sm.currentState == StateError {
		sm.m.Unlock()
		return
	}
	current := sm.currentState
	sm.m.Unlock()

	sm.log.Error().Str("from", current.String()).Str("reason", reason).Msg("transitioning to error state")
	if err := sm.transition(StateError); err != nil {
		sm.log.Error().Err(err).Str("reason", reason).Msg("failed to transition to error state")
	}
}

func (sm *ConnectionStateMachine) OnDisconnected() {
	_ = sm.transition(StateDisconnected) // #nosec G104

}

func (sm *ConnectionStateMachine) GetState() ConnectionState {
	sm.m.RLock()
	defer sm.m.RUnlock()
	return sm.currentState
}

func (sm *ConnectionStateMachine) IsOperational() bool {
	return sm.GetState() == StateFullyOperational
}

func (sm *ConnectionStateMachine) IsHealthy() bool {
	state := sm.GetState()
	return state == StateFullyOperational || state == StateJoiningChannels
}
