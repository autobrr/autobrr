// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

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
	StateAwaitingBots     // Waiting for invite bots to appear
	StateSendingInvites   // Sending invite commands
	StateFullyOperational // All channels joined, all invites sent
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
		"AwaitingBots",
		"SendingInvites",
		"FullyOperational",
		"Error",
	}[s]
}

type ConnectionStateMachine struct {
	m            deadlock.RWMutex
	currentState ConnectionState
	handler      *Handler
	log          zerolog.Logger

	// State tracking
	authAttempts   int
	inviteAttempts int
	maxRetries     int

	// Bot tracking for invite commands
	requiredBots   map[string]bool // bots we need before sending invites
	botsPresent    map[string]bool // which bots are currently present
	pendingInvites []string        // commands to send

	// Timers and cancellation
	watcherDone chan struct{}
}

func NewConnectionStateMachine(handler *Handler) *ConnectionStateMachine {
	return &ConnectionStateMachine{
		currentState:   StateDisconnected,
		handler:        handler,
		log:            handler.log.With().Str("component", "state-machine").Logger(),
		maxRetries:     5,
		requiredBots:   make(map[string]bool),
		botsPresent:    make(map[string]bool),
		pendingInvites: make([]string, 0),
		watcherDone:    make(chan struct{}),
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
		StateDisconnected:     {StateConnecting, StateConnected},
		StateConnecting:       {StateConnected, StateError, StateDisconnected},
		StateConnected:        {StateAuthenticating, StateAuthenticated, StateDisconnected},
		StateAuthenticating:   {StateAuthenticated, StateError, StateDisconnected},
		StateAuthenticated:    {StateJoiningChannels, StateFullyOperational, StateDisconnected},
		StateJoiningChannels:  {StateAwaitingBots, StateFullyOperational, StateDisconnected},
		StateAwaitingBots:     {StateSendingInvites, StateFullyOperational, StateError, StateDisconnected},
		StateSendingInvites:   {StateFullyOperational, StateAwaitingBots, StateError, StateDisconnected},
		StateFullyOperational: {StateDisconnected},
		StateError:            {StateDisconnected, StateConnecting},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	return slices.Contains(allowed, to)
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
		sm.m.Lock()
		sm.inviteAttempts = 0
		sm.m.Unlock()
		sm.handleJoinChannels()

	case StateJoiningChannels:
		// Channels are joining, wait for NAMES replies

	case StateAwaitingBots:
		sm.prepareInviteCommands()

	case StateSendingInvites:
		sm.sendInviteCommands()

	case StateFullyOperational:
		sm.log.Info().Msg("IRC connection fully operational")
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

func (sm *ConnectionStateMachine) prepareInviteCommands() {
	if sm.handler.network.InviteCommand == "" {
		// No invites needed, go straight to operational
		sm.log.Debug().Msg("no invite commands, going operational")
		sm.transition(StateFullyOperational)
		return
	}

	commands, err := parseInviteCommands(sm.handler.network.InviteCommand)
	if err != nil {
		sm.log.Error().Err(err).Msg("failed to parse invite commands")
		sm.transition(StateFullyOperational) // Continue without invites
		return
	}

	sm.m.Lock()
	sm.pendingInvites = commands
	sm.requiredBots = make(map[string]bool)
	sm.botsPresent = make(map[string]bool)

	// Extract bot names from commands
	for _, cmd := range commands {
		parts := strings.SplitN(strings.TrimSpace(cmd), " ", 2)
		if len(parts) > 0 {
			botName := strings.ToLower(parts[0])
			sm.requiredBots[botName] = true
			sm.botsPresent[botName] = false
		}
	}
	sm.m.Unlock()

	sm.log.Debug().Int("bot_count", len(sm.requiredBots)).Msg("waiting for invite bots")

	// Check if bots are already present
	if sm.checkBotPresence() {
		sm.log.Debug().Msg("all bots already present")
		sm.transition(StateSendingInvites)
	} else {
		// Start watching for bots
		go sm.watchForBots()
	}
}

func (sm *ConnectionStateMachine) checkBotPresence() bool {
	sm.m.Lock()
	defer sm.m.Unlock()

	for botName := range sm.requiredBots {
		if bot, ok := sm.handler.bots.Get(botName); ok {
			sm.botsPresent[botName] = bot.Present && bot.State == domain.IrcUserStatePresent
		}
	}

	// Check if all required bots are present
	allPresent := true
	for botName, required := range sm.requiredBots {
		if required && !sm.botsPresent[botName] {
			allPresent = false
			sm.log.Debug().Str("bot", botName).Msg("waiting for bot to appear")
		}
	}

	return allPresent
}

func (sm *ConnectionStateMachine) watchForBots() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(60 * time.Second) // Give up after 60 seconds

	for {
		select {
		case <-ticker.C:
			sm.m.RLock()
			if sm.currentState != StateAwaitingBots {
				sm.m.RUnlock()
				return
			}
			sm.m.RUnlock()

			if sm.checkBotPresence() {
				sm.log.Info().Msg("all required bots present, sending invites")
				sm.transition(StateSendingInvites)
				return
			}

		case <-timeout:
			sm.log.Warn().Msg("timeout waiting for invite bots, continuing anyway")
			sm.handler.addConnectError("invite bots did not appear within timeout")
			sm.transition(StateFullyOperational)
			return

		case <-sm.watcherDone:
			return
		}
	}
}

func (sm *ConnectionStateMachine) sendInviteCommands() {
	sm.m.Lock()
	commands := sm.pendingInvites
	sm.m.Unlock()

	success := true

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		sm.log.Debug().Msgf("sending invite command: %s", cmd)
		params := strings.SplitN(cmd, " ", 2)

		if err := sm.handler.Send("PRIVMSG", params...); err != nil {
			sm.log.Error().Err(err).Msgf("error sending invite command: %s", cmd)
			success = false
		}

		time.Sleep(1 * time.Second)
	}

	if success {
		sm.transition(StateFullyOperational)
	} else {
		sm.m.Lock()
		sm.inviteAttempts++
		attempts := sm.inviteAttempts
		sm.m.Unlock()

		if attempts < sm.maxRetries {
			sm.log.Warn().Msgf("invite commands failed, retry %d/%d", attempts, sm.maxRetries)
			time.Sleep(10 * time.Second)
			sm.transition(StateAwaitingBots) // Go back and check bots again
		} else {
			sm.log.Error().Msg("max invite retries reached")
			sm.handler.addConnectError("failed to send invite commands after max retries")
			sm.transition(StateFullyOperational) // Give up but stay connected
		}
	}
}

func (sm *ConnectionStateMachine) cleanup() {
	// Signal watcher to stop if running
	select {
	case sm.watcherDone <- struct{}{}:
	default:
	}

	sm.m.Lock()
	sm.requiredBots = make(map[string]bool)
	sm.botsPresent = make(map[string]bool)
	sm.pendingInvites = nil
	sm.m.Unlock()
}

func (sm *ConnectionStateMachine) handleError() {
	sm.log.Error().Str("state", sm.currentState.String()).Msg("error state reached")
	// Could implement retry logic here
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
	//sm.m.RLock()
	//currentState := sm.currentState
	//sm.m.RUnlock()

	//if currentState != StateJoiningChannels {
	//	return
	//}

	// Check if all channels are joined
	allJoined := true
	sm.handler.channels.ForEach(func(s string, ch *Channel) bool {
		if ch.Enabled && !ch.Monitoring {
			allJoined = false
			return false // stop iteration
		}
		return true
	})

	if allJoined {
		// Decide next state based on whether we need invites
		if sm.handler.network.InviteCommand != "" {
			sm.transition(StateAwaitingBots)
		} else {
			sm.transition(StateFullyOperational)
		}
	}
}

func (sm *ConnectionStateMachine) OnBotJoined(botName string) {
	sm.m.Lock()
	if sm.currentState == StateAwaitingBots {
		if _, required := sm.requiredBots[botName]; required {
			sm.botsPresent[botName] = true
			sm.log.Debug().Str("bot", botName).Msg("required bot appeared")
		}
	}
	sm.m.Unlock()

	// checkBotPresence will be called by watchForBots ticker
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
