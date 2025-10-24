package irc

import (
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

type ChannelState int

const (
	ChannelStateIdle ChannelState = iota
	ChannelStateAwaitingInvite
	ChannelStateJoining
	ChannelStateMonitoring
	ChannelStateError
)

func (s ChannelState) String() string {
	switch s {
	case ChannelStateIdle:
		return "Idle"
	case ChannelStateAwaitingInvite:
		return "AwaitingInvite"
	case ChannelStateJoining:
		return "Joining"
	case ChannelStateMonitoring:
		return "Monitoring"
	case ChannelStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

type ChannelStateMachine struct {
	m       deadlock.RWMutex
	state   ChannelState
	channel *Channel
	handler *Handler
	log     zerolog.Logger

	inviteCommand string
	lastAttempt   time.Time
}

func NewChannelStateMachine(channel *Channel, handler *Handler, inviteCommand string) *ChannelStateMachine {
	return &ChannelStateMachine{
		state:         ChannelStateIdle,
		channel:       channel,
		handler:       handler,
		log:           handler.log.With().Str("channel", channel.Name).Str("component", "channel-state-machine").Logger(),
		inviteCommand: strings.TrimSpace(inviteCommand),
	}
}

func (sm *ChannelStateMachine) Start() {
	sm.m.Lock()
	defer sm.m.Unlock()

	if sm.state == ChannelStateMonitoring || sm.state == ChannelStateJoining || sm.state == ChannelStateAwaitingInvite {
		return
	}

	sm.runJoinFlowLocked()
}

func (sm *ChannelStateMachine) runJoinFlowLocked() {
	if !sm.channel.Enabled {
		sm.log.Debug().Msg("channel disabled, skipping join workflow")
		return
	}

	sm.lastAttempt = time.Now()

	if sm.inviteCommand != "" {
		sm.state = ChannelStateAwaitingInvite
		sm.log.Debug().Str("invite_command", sm.inviteCommand).Msg("sending invite command for channel")
		if err := sm.sendInviteCommandLocked(); err != nil {
			sm.transitionErrorLocked(err)
		}
		return
	}

	sm.state = ChannelStateJoining
	sm.log.Debug().Msg("joining channel")
	if err := sm.handler.JoinChannel(sm.channel.Name, sm.channel.Password); err != nil {
		sm.transitionErrorLocked(err)
	}
}

func (sm *ChannelStateMachine) OnInvite(nick string) {
	sm.m.Lock()
	defer sm.m.Unlock()

	if sm.state != ChannelStateAwaitingInvite {
		return
	}

	sm.log.Debug().Str("from", nick).Msg("received invite, joining channel")
	sm.state = ChannelStateJoining
	if err := sm.handler.JoinChannel(sm.channel.Name, sm.channel.Password); err != nil {
		sm.transitionErrorLocked(err)
	}
}

func (sm *ChannelStateMachine) OnJoinSuccess() {
	sm.m.Lock()
	defer sm.m.Unlock()

	sm.state = ChannelStateMonitoring
	sm.channel.ClearConnectionErrors()
}

func (sm *ChannelStateMachine) OnParted() {
	sm.m.Lock()
	defer sm.m.Unlock()

	if sm.state == ChannelStateMonitoring {
		sm.state = ChannelStateIdle
	}
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
	sm.inviteCommand = strings.TrimSpace(inviteCommand)
}
