package irc

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type ChannelState int

const (
	ChannelStateIdle ChannelState = iota
	ChannelStateAwaitingInvite
	ChannelStateAwaitingInviteBot
	ChannelStateInviteFailed
	ChannelStateInviteFailedNoSuchNick
	ChannelStateJoining
	ChannelStateMonitoring
	ChannelStateKicked
	ChannelStateParted
	ChannelStateDisabled
	ChannelStateError
)

// recovery tuning. Delays are fields on the state machine (defaulting to these)
// so tests can shorten them; the caps/counts are fixed.
const (
	defaultInviteResponseTimeout = 30 * time.Second
	// defaultInviteResponseGrace is how long we wait, after a present invite bot
	// answers our request with a plain message, for a JOIN before concluding the
	// bot refused us. Some trackers' bots acknowledge positively ("attempting to
	// join you") and then have the server force-join us near-instantly, so we must
	// not treat that first message as a rejection - the JOIN, not the message, is
	// authoritative. Long enough to cover a force-join round trip, short enough to
	// surface a genuine rejection promptly.
	defaultInviteResponseGrace = 5 * time.Second
	defaultJoinConfirmTimeout  = 45 * time.Second
	defaultErrorRetryBaseDelay = 15 * time.Minute

	maxErrorRetries    = 8
	maxErrorRetryDelay = 6 * time.Hour
)

func (s ChannelState) String() string {
	switch s {
	case ChannelStateIdle:
		return "Idle"
	case ChannelStateAwaitingInvite:
		return "AwaitingInvite"
	case ChannelStateAwaitingInviteBot:
		return "AwaitingInviteBot"
	case ChannelStateInviteFailed:
		return "InviteFailed"
	case ChannelStateInviteFailedNoSuchNick:
		return "InviteFailedNoSuchNick"
	case ChannelStateJoining:
		return "Joining"
	case ChannelStateMonitoring:
		return "Monitoring"
	case ChannelStateKicked:
		return "Kicked"
	case ChannelStateParted:
		return "Parted"
	case ChannelStateDisabled:
		return "Disabled"
	case ChannelStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

var validChannelTransitions = map[ChannelState][]ChannelState{
	ChannelStateIdle: {
		ChannelStateJoining,
		ChannelStateAwaitingInvite,
		ChannelStateMonitoring,
		ChannelStateError,
		ChannelStateKicked,
		ChannelStateParted,
	},
	ChannelStateJoining: {
		ChannelStateMonitoring,
		ChannelStateAwaitingInvite,
		ChannelStateError,
		ChannelStateKicked,
		ChannelStateParted,
	},
	ChannelStateAwaitingInvite: {
		ChannelStateMonitoring,
		ChannelStateAwaitingInviteBot,
		ChannelStateInviteFailedNoSuchNick,
		ChannelStateJoining,
		ChannelStateError,
		ChannelStateKicked,
		ChannelStateParted,
	},
	ChannelStateInviteFailed: {
		// a rejection is a definitive "no": the channel parks here (no auto-retry)
		// and recovers only via an explicit INVITE, a config change or a reconnect
		ChannelStateJoining,
		ChannelStateMonitoring, // a late JOIN confirmation (force-join) should still recover
	},
	ChannelStateAwaitingInviteBot: {
		ChannelStateAwaitingInvite,
		ChannelStateInviteFailedNoSuchNick,
		ChannelStateMonitoring,
		ChannelStateJoining,
		ChannelStateError,
		ChannelStateKicked,
	},
	ChannelStateInviteFailedNoSuchNick: {
		ChannelStateAwaitingInviteBot,
		ChannelStateJoining,
		ChannelStateError,
	},
	ChannelStateKicked: {
		ChannelStateIdle,
		ChannelStateJoining,        // only via an explicit invite; never auto-rejoin
		ChannelStateAwaitingInvite, // idem, for invite channels
		ChannelStateMonitoring,     // a late JOIN confirmation should still recover
	},
	ChannelStateParted: {
		ChannelStateIdle,
		ChannelStateJoining,
	},
	ChannelStateDisabled: {
		ChannelStateIdle,
	},
	ChannelStateError: {
		ChannelStateIdle,
		ChannelStateJoining,        // auto-retry / recover (direct-join channels)
		ChannelStateAwaitingInvite, // auto-retry / recover (invite channels)
		ChannelStateMonitoring,     // a successful JOIN should recover directly
	},
}

type ChannelStateMachine struct {
	m       sync.RWMutex
	state   ChannelState
	channel *Channel
	handler *Handler
	log     zerolog.Logger

	inviteCommand   string
	lastAttempt     time.Time
	authAttempts    int
	joinAfterInvite bool

	// recovery bookkeeping (guarded by m)
	inviteGen     int // bumped on each AwaitingInvite entry; guards stale invite-timeout fires
	joinGen       int // bumped on each Joining entry; guards stale join-timeout fires
	errorAttempts int

	// invite-response bookkeeping (guarded by m). A present invite bot that answers
	// our request but has not (yet) joined us is only treated as a rejection once
	// the grace timer elapses without a JOIN; see OnInviteBotResponse.
	inviteResponded    bool
	inviteRejectReason string

	// tunable delays, immutable after construction (tests may set them before use)
	inviteTimeout       time.Duration
	inviteResponseGrace time.Duration
	joinTimeout         time.Duration
	errorBaseDelay      time.Duration
}

func NewChannelStateMachine(channel *Channel, handler *Handler, inviteCommand string) *ChannelStateMachine {
	return &ChannelStateMachine{
		state:               ChannelStateIdle,
		channel:             channel,
		handler:             handler,
		log:                 handler.log.With().Str("channel", channel.Name).Str("component", "channel-state").Logger(),
		inviteCommand:       strings.TrimSpace(inviteCommand),
		authAttempts:        0,
		inviteTimeout:       defaultInviteResponseTimeout,
		inviteResponseGrace: defaultInviteResponseGrace,
		joinTimeout:         defaultJoinConfirmTimeout,
		errorBaseDelay:      defaultErrorRetryBaseDelay,
	}
}

func (sm *ChannelStateMachine) transition(to ChannelState) error {
	sm.m.Lock()
	defer sm.m.Unlock()

	from := sm.state

	if !sm.isValidTransition(from, to) {
		sm.log.Error().Str("from", from.String()).Str("to", to.String()).Msg("invalid state transition")
		return fmt.Errorf("invalid state transition from %s to %s", from, to)
	}

	sm.log.Trace().Str("from", from.String()).Str("to", to.String()).Msg("transitioning channel state")

	sm.state = to

	go sm.onStateEntry(to)

	return nil
}

func (sm *ChannelStateMachine) isValidTransition(from, to ChannelState) bool {
	allowed, ok := validChannelTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}

func (sm *ChannelStateMachine) onStateEntry(state ChannelState) {
	// Monitoring applies its channel-level effects (clears errors, sets the
	// monitoring flag) BEFORE broadcasting, so the STATE event - and the network
	// health it carries - reflect the settled state rather than the pre-join one.
	if state == ChannelStateMonitoring {
		sm.handleMonitoring()
	}

	sm.broadcastStateChange(state)

	switch state {
	case ChannelStateIdle:
	case ChannelStateJoining:
		sm.runJoin()
	case ChannelStateAwaitingInvite:
		sm.handleAwaitingInvite()
	case ChannelStateAwaitingInviteBot:
		sm.handleWaitForInviteBot()
	case ChannelStateInviteFailed:
		// entered by direct assignment in onInviteResponseTimeout; parks with no entry action
	case ChannelStateInviteFailedNoSuchNick:
		sm.handleNoSuchNick()
	case ChannelStateMonitoring:
		// applied above, before the broadcast
	case ChannelStateKicked:
		sm.handleKicked()
	case ChannelStateParted:
	case ChannelStateDisabled:
	case ChannelStateError:
		// entry is handled directly by enterError (which can be reached from
		// states such as Monitoring that have no transition() path)
	default:
		sm.log.Error().Str("state", state.String()).Msg("invalid state")
	}
}

func (sm *ChannelStateMachine) Start() {
	if sm.CurrentState() == ChannelStateMonitoring {
		return
	}

	if !sm.channel.IsEnabled() {
		sm.log.Debug().Msg("channel disabled, skipping join workflow")
		sm.transition(ChannelStateDisabled)
		return
	}

	sm.m.RLock()
	hasInvite := sm.inviteCommand != ""
	sm.m.RUnlock()

	if hasInvite {
		sm.transition(ChannelStateAwaitingInvite)
		return
	}

	sm.transition(ChannelStateJoining)
}

// Reset returns the state machine to its initial Idle state and clears the
// invite/backoff bookkeeping. It is called when the connection drops so that a
// channel can run its join workflow again on the next (re)connect.
//
// ChannelStateMonitoring has no outgoing transitions in validChannelTransitions,
// so a channel that was being monitored when the connection dropped would
// otherwise be stuck: Start() -> transition(Monitoring -> Joining) is rejected
// as invalid and no JOIN is ever re-sent. The state is assigned directly (rather
// than via transition()) precisely because Monitoring has no valid path back.
func (sm *ChannelStateMachine) Reset() {
	sm.m.Lock()
	sm.state = ChannelStateIdle
	sm.authAttempts = 0
	sm.errorAttempts = 0
	sm.inviteGen++ // invalidate any pending invite-timeout / grace callback
	sm.joinGen++   // invalidate any pending join-timeout callback
	sm.joinAfterInvite = false
	sm.inviteResponded = false
	sm.inviteRejectReason = ""
	sm.m.Unlock()

	// broadcast outside the lock so the UI reflects the drop immediately
	// instead of showing a stale Monitoring pill until the channel rejoins.
	sm.broadcastStateChange(ChannelStateIdle)
}

// handleAwaitingInvite sends the invite command and arms a timeout so that a
// silent invite bot (no INVITE and no ERR_NOSUCHNICK) still falls through to the
// backoff/retry loop instead of stalling in AwaitingInvite forever.
func (sm *ChannelStateMachine) handleAwaitingInvite() {
	if !sm.channel.IsEnabled() {
		sm.log.Debug().Msg("channel disabled, skipping join workflow")
		return
	}

	sm.m.Lock()
	inviteCommand := sm.inviteCommand
	if inviteCommand == "" {
		sm.m.Unlock()
		sm.transition(ChannelStateJoining)
		return
	}
	sm.lastAttempt = time.Now()
	sm.authAttempts++
	sm.inviteGen++
	// fresh attempt: forget any bot response recorded for a previous one
	sm.inviteResponded = false
	sm.inviteRejectReason = ""
	gen := sm.inviteGen
	attempt := sm.authAttempts
	timeout := sm.inviteTimeout
	sm.m.Unlock()

	sm.log.Debug().Str("invite_command", inviteCommand).Int("attempt", attempt).Msg("sending invite command for channel")

	if err := sm.sendInviteCommand(inviteCommand); err != nil {
		sm.enterError(err.Error())
		return
	}

	time.AfterFunc(timeout, func() { sm.onInviteTimeout(gen) })
}

// onInviteTimeout fires when we have waited too long for a response to the invite
// command. It only acts if the channel is still waiting for the same invite
// attempt (gen), otherwise an INVITE/NOSUCHNICK/reset already moved us on. If the
// bot HAS answered (inviteResponded), it defers entirely to the grace timer
// (onInviteResponseTimeout): the bot is present, so this is a rejection to be
// parked, not an absent bot to retry - retrying here would re-send the invite to a
// bot that already answered and discard its reason. This matters when the bot
// replies in the last inviteResponseGrace seconds of the timeout window, where
// this timer would otherwise fire first.
func (sm *ChannelStateMachine) onInviteTimeout(gen int) {
	sm.m.Lock()
	if sm.state != ChannelStateAwaitingInvite || sm.inviteGen != gen || sm.inviteResponded {
		sm.m.Unlock()
		return
	}
	// Flip to the backoff state under the same lock that validated we are still on
	// this invite attempt (AwaitingInvite -> AwaitingInviteBot is a valid
	// transition). Doing this atomically - rather than releasing the lock and
	// calling transition() - means a concurrent onInviteResponseTimeout, which
	// parks by direct assignment (and bumps inviteGen), cannot be overtaken and
	// leave the channel retrying a bot that has already refused us.
	sm.state = ChannelStateAwaitingInviteBot
	sm.m.Unlock()

	sm.log.Debug().Msg("no invite response received, retrying after backoff")
	go sm.onStateEntry(ChannelStateAwaitingInviteBot)
}

func (sm *ChannelStateMachine) runJoin() {
	if !sm.channel.IsEnabled() {
		sm.log.Debug().Msg("channel disabled, skipping join workflow")
		return
	}

	sm.m.Lock()
	sm.lastAttempt = time.Now()
	joinAfterInvite := sm.joinAfterInvite
	sm.joinAfterInvite = false
	inviteCommand := sm.inviteCommand
	sm.joinGen++
	gen := sm.joinGen
	timeout := sm.joinTimeout
	sm.m.Unlock()

	if inviteCommand != "" && !joinAfterInvite {
		sm.transition(ChannelStateAwaitingInvite)
		return
	}

	sm.log.Debug().Msg("joining channel")
	if err := sm.handler.JoinChannel(sm.channel.Name, sm.channel.GetPassword()); err != nil {
		sm.enterError(err.Error())
		return
	}

	// if RPL_ENDOFNAMES never arrives (join silently rejected: banned, full,
	// bad key, or a slow/broken server), fall into the bounded error-retry loop
	// instead of stalling in Joining forever.
	time.AfterFunc(timeout, func() { sm.onJoinTimeout(gen) })
}

// onJoinTimeout fires when a JOIN was sent but never confirmed. It only acts if
// the channel is still on the same join attempt (a later JOIN, a successful
// RPL_ENDOFNAMES, a kick or a reset all supersede it).
func (sm *ChannelStateMachine) onJoinTimeout(gen int) {
	sm.m.Lock()
	stale := sm.state != ChannelStateJoining || sm.joinGen != gen
	sm.m.Unlock()
	if stale {
		return
	}

	sm.log.Debug().Msg("join not confirmed in time, treating as failed")
	sm.enterError("join not confirmed")
}

func (sm *ChannelStateMachine) OnInvite(nick string) {
	sm.m.Lock()
	switch sm.state {
	case ChannelStateMonitoring, ChannelStateJoining, ChannelStateDisabled, ChannelStateParted:
		// already joined/joining, or intentionally not in the channel
		sm.m.Unlock()
		return
	}
	sm.joinAfterInvite = true
	sm.m.Unlock()

	sm.log.Debug().Str("from", nick).Msg("received invite, joining channel")
	sm.transition(ChannelStateJoining)
}

func (sm *ChannelStateMachine) handleWaitForInviteBot() {
	sm.m.RLock()
	attempts := sm.authAttempts
	sm.m.RUnlock()

	delay, ok := retryBackoff(attempts)
	if !ok {
		sm.log.Debug().Int("attempt", attempts).Msg("invite retries exhausted, marking channel as errored")
		sm.enterError("invite retries exhausted")
		return
	}

	sm.log.Debug().Dur("sleep", delay).Int("attempt", attempts).Msg("waiting for invite bot before retrying")
	time.Sleep(delay)

	// the invite may have arrived (or the channel been reset) while we slept
	if sm.CurrentState() != ChannelStateAwaitingInviteBot {
		return
	}

	sm.transition(ChannelStateAwaitingInvite)
}

// OnInviteBotResponse is called when a present invite bot answers our invite
// command with a plain NOTICE/DM instead of an actual INVITE. This is deliberately
// NOT treated as an immediate failure: several trackers
// acknowledge positively - "Attempting to join you to #chan" - and then have the
// server force-join us near-instantly, so the JOIN, not the message, is the
// authoritative signal of success. We record the bot's reason and arm a short
// grace timer; whichever happens first decides the outcome:
//   - a JOIN confirmation moves us to Monitoring (the grace fire then no-ops), or
//   - the grace elapses with no join -> the bot answered but never let us in, a
//     definitive rejection: park in InviteFailed with the reason (see
//     onInviteResponseTimeout).
//
// It only acts while we are actively awaiting an invite. An *absent* bot never
// calls this, so its silent invite timeout still routes to the backoff/retry loop
// (the bot may simply not be connected yet).
func (sm *ChannelStateMachine) OnInviteBotResponse(reason string) {
	sm.m.Lock()
	if sm.state != ChannelStateAwaitingInvite {
		sm.m.Unlock()
		return
	}
	// keep the latest message as the reason we would surface on failure
	sm.inviteRejectReason = reason
	if sm.inviteResponded {
		// grace already armed for this attempt; a repeat NOTICE just updates the reason
		sm.m.Unlock()
		return
	}
	sm.inviteResponded = true
	gen := sm.inviteGen
	grace := sm.inviteResponseGrace
	sm.m.Unlock()

	sm.log.Debug().Str("reason", reason).Msg("invite bot responded; waiting for join confirmation before deciding outcome")
	time.AfterFunc(grace, func() { sm.onInviteResponseTimeout(gen) })
}

// onInviteResponseTimeout fires a short grace after a present invite bot answered
// our request. If we are still awaiting an invite on the same attempt (no JOIN,
// reset or new attempt in the meantime), the bot answered but refused us: this is
// a definitive rejection, so park in InviteFailed with the bot's reason rather
// than retrying (which would spam a bot that already said no). Parking by direct
// assignment under one lock - and bumping inviteGen - keeps it atomic w.r.t. the
// still-pending silent invite timeout, which then no-ops.
func (sm *ChannelStateMachine) onInviteResponseTimeout(gen int) {
	sm.m.Lock()
	if sm.state != ChannelStateAwaitingInvite || sm.inviteGen != gen {
		sm.m.Unlock()
		return
	}
	reason := sm.inviteRejectReason
	if reason == "" {
		reason = "invite bot responded but did not grant access"
	}
	sm.state = ChannelStateInviteFailed
	sm.inviteGen++
	// Set the connection error while STILL holding sm.m. A racing late force-join
	// (OnJoinSuccess -> transition(InviteFailed -> Monitoring)) must acquire sm.m in
	// transition() only after we release it here, so its handleMonitoring - which
	// clears this error - is guaranteed to run AFTER SetConnectionError rather than
	// racing it (which would otherwise leave the channel Monitoring but flagged
	// errored until the next reconnect). Deadlock-safe: the lock order is only ever
	// sm.m -> channel.m (SetConnectionError takes channel.m and never re-enters sm;
	// no path takes channel.m then sm.m - Channel.SetInviteCommand releases channel.m
	// before entering the state machine).
	sm.channel.SetConnectionError(reason)
	sm.m.Unlock()

	sm.log.Warn().Str("reason", reason).Msg("invite request rejected by bot; parking channel until credentials are fixed or an invite arrives")
	sm.broadcastStateChange(ChannelStateInviteFailed)
}

func (sm *ChannelStateMachine) OnNoSuchNick(nick string) {
	sm.transition(ChannelStateInviteFailedNoSuchNick)
}

func (sm *ChannelStateMachine) handleNoSuchNick() {
	sm.log.Debug().Msg("no such nick")
	// route into the backoff/retry loop
	sm.transition(ChannelStateAwaitingInviteBot)
}

func (sm *ChannelStateMachine) OnJoinSuccess() {
	if sm.CurrentState() == ChannelStateMonitoring {
		return
	}
	sm.transition(ChannelStateMonitoring)
}

func (sm *ChannelStateMachine) handleMonitoring() {
	sm.m.Lock()
	if sm.state != ChannelStateMonitoring {
		sm.m.Unlock()
		return
	}
	// a successful join clears the retry/backoff bookkeeping
	sm.authAttempts = 0
	sm.errorAttempts = 0
	sm.inviteResponded = false
	sm.inviteRejectReason = ""
	sm.m.Unlock()

	sm.channel.SetMonitoring()
	sm.log.Debug().Msg("monitoring channel")
	// onStateEntry broadcasts the Monitoring state after this runs, so the event
	// reflects the cleared errors / set monitoring flag

	// Re-evaluate the network's operational state now that this channel's monitoring
	// effects (flag set, errors cleared) have settled. handler.handleJoined calls
	// OnChannelJoined inline right after OnJoinSuccess - i.e. BEFORE this async entry
	// action runs - so on the stale pre-monitoring snapshot the connection SM can
	// latch StateError (any Error/InviteFailed -> Monitoring recovery, e.g. a late
	// force-join). This second notification lets it settle on the joined state.
	if sm.handler != nil && sm.handler.stateMachine != nil {
		sm.handler.stateMachine.OnChannelJoined(sm.channel.Name)
	}
}

// OnParted marks a monitored channel as parted when we leave it. The state is
// assigned directly because Monitoring has no transition() path out.
func (sm *ChannelStateMachine) OnParted() {
	sm.m.Lock()
	if sm.state != ChannelStateMonitoring {
		sm.m.Unlock()
		return
	}
	sm.state = ChannelStateParted
	sm.m.Unlock()

	sm.broadcastStateChange(ChannelStateParted)
}

// OnKicked marks the channel as kicked and, deliberately, does NOT schedule any rejoining.
// A kicked channel stays Kicked until it is cleared by a
// reconnect (Reset), a manual restart, or an explicit INVITE (OnInvite) - i.e.
// only when we are actually welcomed back, never on our own initiative.
func (sm *ChannelStateMachine) OnKicked(nick, kickedBy, reason string) {
	sm.m.Lock()
	sm.state = ChannelStateKicked
	sm.m.Unlock()

	sm.channel.ResetMonitoring()

	msg := domain.IrcMessage{
		Network: sm.channel.NetworkID,
		Channel: sm.channel.Name,
		Type:    "KICK",
		Nick:    "<-*",
		Message: fmt.Sprintf("%s was kicked from %s by %s (%s)", nick, sm.channel.Name, kickedBy, reason),
		Time:    time.Now(),
	}
	sm.channel.Messages.AddMessage(msg)

	sm.handler.broadcastMessage(msg)
	sm.broadcastStateChange(ChannelStateKicked)

	sm.log.Info().Str("by", kickedBy).Str("reason", reason).Msg("kicked from channel; not auto-rejoining")
}

func (sm *ChannelStateMachine) handleKicked() {
	// OnKicked sets the state directly (Monitoring has no transition to Kicked)
	// and intentionally schedules no rejoin, so there is nothing to do on entry.
}

func (sm *ChannelStateMachine) OnError(reason string) {
	sm.enterError(reason)
}

// enterError moves the channel into the Error state and schedules an automatic
// recovery attempt. Error is entered by direct assignment (not transition())
// because it can be reached from states such as Monitoring that have no
// transitions defined.
func (sm *ChannelStateMachine) enterError(reason string) {
	sm.m.Lock()
	sm.state = ChannelStateError
	sm.errorAttempts++
	attempt := sm.errorAttempts
	sm.m.Unlock()

	sm.channel.SetConnectionError(reason)
	sm.log.Warn().Str("reason", reason).Int("attempt", attempt).Msg("channel entered error state")
	sm.broadcastStateChange(ChannelStateError)

	go sm.scheduleErrorRetry(attempt)
}

// scheduleErrorRetry waits out a growing backoff and then restarts the join
// workflow. Only the most recent error attempt drives recovery, and only while
// the channel is still errored; after maxErrorRetries it gives up.
func (sm *ChannelStateMachine) scheduleErrorRetry(attempt int) {
	delay, ok := sm.errorRetryDelay(attempt)
	if !ok {
		sm.log.Warn().Int("attempt", attempt).Msg("giving up channel recovery after repeated errors")
		return
	}

	sm.log.Debug().Dur("delay", delay).Int("attempt", attempt).Msg("scheduling channel recovery after error")
	time.Sleep(delay)

	sm.m.Lock()
	stale := sm.state != ChannelStateError || sm.errorAttempts != attempt
	inviteCommand := sm.inviteCommand
	sm.m.Unlock()
	if stale {
		return
	}

	if inviteCommand != "" {
		sm.transition(ChannelStateAwaitingInvite)
		return
	}
	sm.transition(ChannelStateJoining)
}

// errorRetryDelay returns how long to wait before retrying from Error and
// whether we should still try. The delay grows exponentially up to a cap.
func (sm *ChannelStateMachine) errorRetryDelay(attempt int) (time.Duration, bool) {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > maxErrorRetries {
		return 0, false
	}
	delay := sm.errorBaseDelay << (attempt - 1)
	if delay <= 0 || delay > maxErrorRetryDelay {
		delay = maxErrorRetryDelay
	}
	return delay, true
}

func (sm *ChannelStateMachine) sendInviteCommand(cmd string) error {
	if cmd == "" {
		return errors.New("invite command missing")
	}

	params := strings.SplitN(cmd, " ", 2)

	if len(params) < 2 {
		return errors.New("invalid invite command")
	}

	if err := sm.handler.Send("PRIVMSG", params...); err != nil {
		return errors.Wrap(err, "failed to send invite command")
	}

	time.Sleep(time.Second)

	return nil
}

func (sm *ChannelStateMachine) CurrentState() ChannelState {
	sm.m.RLock()
	defer sm.m.RUnlock()
	return sm.state
}

func (sm *ChannelStateMachine) SetInviteCommand(inviteCommand string) {
	inviteCommand = strings.TrimSpace(inviteCommand)

	sm.m.Lock()
	changed := sm.inviteCommand != inviteCommand
	if changed {
		sm.inviteCommand = inviteCommand
	}
	sm.m.Unlock()

	// transition outside the lock; transition() acquires sm.m itself
	if changed {
		sm.transition(ChannelStateJoining)
	}
}

// broadcastStateChange sends a STATE event via SSE
func (sm *ChannelStateMachine) broadcastStateChange(newState ChannelState) {
	msg := map[string]any{
		"network":           sm.channel.NetworkID,
		"channel":           sm.channel.Name,
		"type":              "STATE",
		"state":             newState.String(),
		"connection_errors": sm.channel.ConnectionErrorsCopy(),
		"healthy":           sm.handler.computeHealthy(),
		"time":              time.Now(),
	}

	sm.handler.broadcastEvent("STATE", msg)
}

// retryBackoff returns the duration to wait before retrying a failed invite attempt.
// The duration is calculated based on the attempt number and duration intervals.
//   - the first 2 minutes are 15 seconds
//   - the next 30 minutes are 30 seconds
//   - the next 60 minutes are 60 seconds,
//   - and the next 5 days are 1 hour.
func retryBackoff(attempt int) (time.Duration, bool) {
	if attempt <= 0 {
		attempt = 1
	}

	const (
		firstPhaseAttempts  = 8   // 2 minutes @ 15s intervals
		secondPhaseAttempts = 60  // next 30 minutes @ 30s intervals
		thirdPhaseAttempts  = 60  // next 60 minutes @ 60s intervals
		fourthPhaseAttempts = 120 // next 5 days @ 1h intervals
	)

	switch {
	case attempt <= firstPhaseAttempts:
		return 15 * time.Second, true
	case attempt <= firstPhaseAttempts+secondPhaseAttempts:
		return 30 * time.Second, true
	case attempt <= firstPhaseAttempts+secondPhaseAttempts+thirdPhaseAttempts:
		return time.Minute, true
	case attempt <= firstPhaseAttempts+secondPhaseAttempts+thirdPhaseAttempts+fourthPhaseAttempts:
		return time.Hour, true
	default:
		return 0, false
	}
}
