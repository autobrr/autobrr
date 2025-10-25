package irc

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
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
		ChannelStateInviteFailed,
		ChannelStateInviteFailedNoSuchNick,
		ChannelStateJoining,
		ChannelStateError,
		ChannelStateKicked,
		ChannelStateParted,
	},
	ChannelStateAwaitingInviteBot: {
		ChannelStateAwaitingInvite,
		ChannelStateInviteFailedNoSuchNick,
		ChannelStateMonitoring,
		ChannelStateJoining,
		ChannelStateError,
		ChannelStateKicked,
	},
	ChannelStateInviteFailed: {
		ChannelStateAwaitingInviteBot,
		ChannelStateJoining,
		ChannelStateError,
	},
	ChannelStateInviteFailedNoSuchNick: {
		ChannelStateAwaitingInviteBot,
		ChannelStateJoining,
		ChannelStateError,
	},
	ChannelStateKicked: {
		ChannelStateIdle,
		ChannelStateJoining,
		ChannelStateAwaitingInvite,
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
	},
}

type ChannelStateMachine struct {
	m       deadlock.RWMutex
	state   ChannelState
	channel *Channel
	handler *Handler
	log     zerolog.Logger

	inviteCommand   string
	lastAttempt     time.Time
	authAttempts    int
	joinAfterInvite bool
}

func NewChannelStateMachine(channel *Channel, handler *Handler, inviteCommand string) *ChannelStateMachine {
	return &ChannelStateMachine{
		state:         ChannelStateIdle,
		channel:       channel,
		handler:       handler,
		log:           handler.log.With().Str("channel", channel.Name).Str("component", "channel-state").Logger(),
		inviteCommand: strings.TrimSpace(inviteCommand),
		authAttempts:  0,
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

	sm.log.Debug().Str("from", from.String()).Str("to", to.String()).Msg("transitioning channel state")

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
	switch state {
	case ChannelStateIdle:
	case ChannelStateJoining:
		sm.runJoin()
	case ChannelStateAwaitingInvite:
		sm.runJoinFlowLocked()
	case ChannelStateAwaitingInviteBot:
		sm.handleWaitForInviteBot()
	case ChannelStateInviteFailed:
		sm.handleInviteFailed()
	case ChannelStateInviteFailedNoSuchNick:
		sm.handleNoSuchNick()
	case ChannelStateMonitoring:
		sm.handleMonitoring()
	case ChannelStateKicked:
		sm.handleKicked()
	case ChannelStateParted:

	default:
		sm.log.Error().Str("state", state.String()).Msgf("invalid state")
	}
}

func (sm *ChannelStateMachine) Start() {
	//sm.m.RLock()
	//defer sm.m.RUnlock()

	//if sm.state == ChannelStateMonitoring || sm.state == ChannelStateJoining || sm.state == ChannelStateAwaitingInvite {
	//	return
	//}

	if !sm.channel.Enabled {
		sm.log.Debug().Msg("channel disabled, skipping join workflow")
		sm.transition(ChannelStateDisabled)
		return
	}

	if sm.inviteCommand != "" {
		sm.transition(ChannelStateAwaitingInvite)
		return
	}

	sm.transition(ChannelStateJoining)
}

func (sm *ChannelStateMachine) runJoinFlowLocked() {
	if !sm.channel.Enabled {
		sm.log.Debug().Msg("channel disabled, skipping join workflow")
		return
	}

	sm.lastAttempt = time.Now()

	if sm.inviteCommand == "" {
		sm.transition(ChannelStateJoining)
		return
	}

	sm.m.Lock()
	sm.authAttempts++
	sm.state = ChannelStateAwaitingInvite
	sm.m.Unlock()

	sm.log.Debug().Str("invite_command", sm.inviteCommand).Int("attempt", sm.authAttempts).Msg("sending invite command for channel")
	if err := sm.sendInviteCommandLocked(); err != nil {
		sm.transitionErrorLocked(err)
	}
}

func (sm *ChannelStateMachine) runJoin() {
	if !sm.channel.Enabled {
		sm.log.Debug().Msg("channel disabled, skipping join workflow")
		return
	}

	sm.lastAttempt = time.Now()

	sm.m.Lock()
	joinAfterInvite := sm.joinAfterInvite
	sm.joinAfterInvite = false
	sm.m.Unlock()

	if sm.inviteCommand != "" && !joinAfterInvite {
		sm.transition(ChannelStateAwaitingInvite)
		return
	}

	sm.log.Debug().Msg("joining channel")
	if err := sm.handler.JoinChannel(sm.channel.Name, sm.channel.Password); err != nil {
		sm.transitionErrorLocked(err)
	}
}

func (sm *ChannelStateMachine) OnInvite(nick string) {
	sm.m.Lock()
	if sm.state != ChannelStateAwaitingInvite && sm.state != ChannelStateKicked {
		sm.m.Unlock()
		return
	}
	sm.joinAfterInvite = true
	sm.m.Unlock()

	sm.log.Debug().Str("from", nick).Msg("received invite, joining channel")
	sm.transition(ChannelStateJoining)
}

func (sm *ChannelStateMachine) OnInviteFailed(msg string) {
	sm.transition(ChannelStateInviteFailed)
}

func (sm *ChannelStateMachine) handleInviteFailed() {
	sm.log.Debug().Msg("invite failed")
	//sm.transition(ChannelStateInviteFailedNoSuchNick)
}

func (sm *ChannelStateMachine) handleWaitForInviteBot() {
	delay, ok := retryBackoff(sm.authAttempts)
	if !ok {
		sm.log.Warn().Int("attempt", sm.authAttempts).Msg("invite retries exhausted, marking channel as errored")
		sm.transition(ChannelStateError)
		return
	}

	sm.log.Debug().Dur("sleep", delay).Int("attempt", sm.authAttempts).Msg("waiting for invite bot before retrying")
	time.Sleep(delay)

	sm.transition(ChannelStateAwaitingInvite)
}

func (sm *ChannelStateMachine) OnNoSuchNick(nick string) {
	sm.transition(ChannelStateInviteFailedNoSuchNick)
}

func (sm *ChannelStateMachine) handleNoSuchNick() {
	sm.log.Debug().Msg("no such nick")
	// start timer to retry
	sm.transition(ChannelStateAwaitingInviteBot)
}

func (sm *ChannelStateMachine) OnJoinSuccess() {
	sm.transition(ChannelStateMonitoring)
}

func (sm *ChannelStateMachine) handleMonitoring() {
	sm.m.RLock()
	defer sm.m.RUnlock()
	if sm.state != ChannelStateMonitoring {
		return
	}
	sm.channel.SetMonitoring()
	sm.log.Debug().Msg("monitoring channel")
}

func (sm *ChannelStateMachine) OnParted() {
	sm.m.Lock()
	defer sm.m.Unlock()

	if sm.state == ChannelStateMonitoring {
		sm.state = ChannelStateIdle
	}
}

func (sm *ChannelStateMachine) OnKicked(nick, kickedBy, reason string) {
	sm.m.Lock()
	defer sm.m.Unlock()

	sm.state = ChannelStateKicked
	sm.channel.ResetMonitoring()

	msg := domain.IrcMessage{
		Network: sm.channel.NetworkID,
		Channel: sm.channel.Name,
		Type:    "KICK",
		//Nick:    kickedBy,
		Nick:    "<-*",
		Message: fmt.Sprintf("%s was kicked from %s by %s (%s)", nick, sm.channel.Name, kickedBy, reason),
		Time:    time.Now(),
	}
	sm.channel.Messages.AddMessage(msg)

	sm.handler.publishSSEMsg(msg)
}

func (sm *ChannelStateMachine) handleKicked() {
	//sm.m.Lock()
	//defer sm.m.Unlock()
	//
	//sm.state = ChannelStateKicked
	//sm.channel.ResetMonitoring()
	//
	//msg := domain.IrcMessage{
	//	Network: sm.channel.NetworkID,
	//	Channel: sm.channel.Name,
	//	Type:    "KICK",
	//	//Nick:    kickedBy,
	//	Nick:    "<-*",
	//	Message: fmt.Sprintf("%s was kicked from %s by %s (%s)", nick, sm.channel.Name, kickedBy, reason),
	//	Time:    time.Now(),
	//}
	//sm.channel.Messages.AddMessage(msg)
	//
	//sm.handler.publishSSEMsg(msg)
}

func (sm *ChannelStateMachine) OnError(reason string) {
	sm.m.Lock()
	defer sm.m.Unlock()

	sm.transitionErrorLocked(errors.New("%s", reason))
}

func (sm *ChannelStateMachine) transitionErrorLocked(err error) {
	sm.state = ChannelStateError
	sm.channel.SetConnectionError(err.Error())
	sm.log.Warn().Err(err).Msg("channel join failed")
}

func (sm *ChannelStateMachine) sendInviteCommandLocked() error {
	cmd := sm.inviteCommand
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
	sm.m.Lock()
	defer sm.m.Unlock()
	if sm.inviteCommand != inviteCommand {
		sm.inviteCommand = strings.TrimSpace(inviteCommand)
		sm.transition(ChannelStateJoining)
	}
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
		return 15 * time.Second, true
	case attempt <= firstPhaseAttempts+secondPhaseAttempts+thirdPhaseAttempts:
		return time.Minute, true
	case attempt <= firstPhaseAttempts+secondPhaseAttempts+thirdPhaseAttempts+fourthPhaseAttempts:
		return time.Hour, true
	default:
		return 0, false
	}
}
