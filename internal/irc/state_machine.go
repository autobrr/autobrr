// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"fmt"
	"slices"

	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
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
	return [...]string{
		"Disconnected",
		"Connecting",
		"Connected",
		"Authenticating",
		"Authenticated",
		"JoiningChannels",
		"FullyOperational",
		"PartiallyOperational",
		"Error",
	}[s]
}

type ConnectionStateMachine struct {
	m            deadlock.RWMutex
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
		log:          handler.log.With().Str("component", "state-machine").Logger(),
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
	validTransitions := map[ConnectionState][]ConnectionState{
		StateDisconnected:         {StateConnecting, StateConnected},
		StateConnecting:           {StateConnected, StateError, StateDisconnected},
		StateConnected:            {StateAuthenticating, StateAuthenticated, StatePartiallyOperational, StateError, StateDisconnected},
		StateAuthenticating:       {StateAuthenticated, StatePartiallyOperational, StateError, StateDisconnected},
		StateAuthenticated:        {StateJoiningChannels, StateFullyOperational, StatePartiallyOperational, StateError, StateDisconnected},
		StateJoiningChannels:      {StateFullyOperational, StatePartiallyOperational, StateError, StateDisconnected},
		StateFullyOperational:     {StatePartiallyOperational, StateError, StateDisconnected},
		StatePartiallyOperational: {StateFullyOperational, StateError, StateDisconnected},
		StateError:                {StateDisconnected, StateConnecting},
	}

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

	sm.transition(to)
}

func (sm *ConnectionStateMachine) updateOperationalState() {
	enabled := 0
	monitoring := 0
	errored := 0

	sm.handler.channels.ForEach(func(name string, ch *Channel) bool {
		if !ch.Enabled {
			return true
		}

		enabled++

		if ch.Monitoring && !ch.HasConnectionErrors() {
			monitoring++
			return true
		}

		if ch.HasConnectionErrors() {
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
		if !ch.Enabled {
			return true
		}

		if !ch.Monitoring || ch.HasConnectionErrors() {
			allJoined = false
			return false
		}

		return true
	})

	return allJoined
}

func (sm *ConnectionStateMachine) onStateEntry(state ConnectionState) {
	switch state {
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

	case StatePartiallyOperational:
		sm.log.Warn().Msg("IRC connection partially operational")
		sm.cleanup()

	case StateError:
		sm.handleError()

	case StateDisconnected:
		sm.cleanup()
	default:
		sm.log.Error().Str("state", state.String()).Msg("invalid state")
	}
}

func (sm *ConnectionStateMachine) handleAuthentication() {
	sm.handler.m.RLock()
	needsAuth := sm.handler.network.Auth.Password != "" && !sm.handler.saslauthed
	sm.handler.m.RUnlock()

	if needsAuth {
		sm.log.Trace().Msg("sending NickServ authentication")
		if err := sm.handler.NickServIdentify(sm.handler.network.Auth.Password); err != nil {
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
	sm.log.Error().Str("state", sm.currentState.String()).Msg("error state reached")
	sm.cleanup()
}

// Event handlers called by IRC callbacks

func (sm *ConnectionStateMachine) OnConnecting() {
	sm.transition(StateConnecting)
}

func (sm *ConnectionStateMachine) OnConnected() {
	sm.transition(StateConnected)

	// Determine next state based on auth requirements
	sm.handler.m.RLock()
	botMode := sm.handler.network.BotMode
	needsAuth := sm.handler.network.Auth.Password != "" && !sm.handler.saslauthed
	sm.handler.m.RUnlock()

	if botMode && sm.handler.botModeSupported() {
		sm.handler.setBotMode()
		// Will transition to auth in handleMode callback
	} else if needsAuth {
		sm.transition(StateAuthenticating)
	} else {
		sm.transition(StateAuthenticated)
	}
}

func (sm *ConnectionStateMachine) OnAuthenticated() {
	sm.m.RLock()
	currentState := sm.currentState
	sm.m.RUnlock()

	if currentState == StateAuthenticating || currentState == StateConnected {
		sm.transition(StateAuthenticated)
	}
}

func (sm *ConnectionStateMachine) OnChannelJoined(channel string) {
	sm.updateOperationalState()

	if sm.allEnabledChannelsMonitoring() {
		sm.transitionIfNeeded(StateFullyOperational)
	}
}

func (sm *ConnectionStateMachine) OnBotJoined(botName string) {
	// Channel state machines handle invite workflows now.
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
	sm.transition(StateDisconnected)
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
