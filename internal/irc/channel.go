// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/featureflags"

	"github.com/alphadose/haxmap"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

type Msg struct {
	ID   int64
	Msg  string
	Time time.Time
	Type string
	From struct {
		Nick string
		Mode string
	}
}

const MaxChannelMessages = 1000

type MessageBuffer struct {
	mu          sync.Mutex
	maxMessages int
	messages    []domain.IrcMessage
}

func NewMessageBuffer(maxMessages int) *MessageBuffer {
	b := &MessageBuffer{
		maxMessages: MaxChannelMessages,
		messages:    make([]domain.IrcMessage, 0),
	}

	if maxMessages > 0 {
		b.maxMessages = maxMessages
	}

	return b
}

// GetMessages returns a copy of the buffered messages. A copy is returned (not
// the backing slice) so callers can read it while new messages are appended.
func (b *MessageBuffer) GetMessages() []domain.IrcMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.messages)
}

func (b *MessageBuffer) ClearMessages() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = make([]domain.IrcMessage, 0)
}

func (b *MessageBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}

func (b *MessageBuffer) AddMessage(msg domain.IrcMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If we're at capacity, remove the oldest message (shift left)
	if len(b.messages) >= b.maxMessages {
		b.messages = append(b.messages[1:], msg)
		return
	}

	b.messages = append(b.messages, msg)
}

// Channel holds the runtime state of a monitored IRC channel.
//
// Mutable scalar/slice fields (ID, Enabled, Password, Topic, Monitoring,
// MonitoringSince, LastAnnounce, ConnectionErrors, inviteCommand) and the fields
// of the announcer *domain.IrcUser values are guarded by m: they are written
// from the IRC-callback and channel-state-machine goroutines and read from the
// HTTP/health goroutine. NetworkID, Name, DefaultChannel, the haxmaps, Messages,
// announceProcessor and stateMachine are set at construction (stateMachine
// before the channel is made visible) and are treated as immutable.
type Channel struct {
	m   deadlock.RWMutex
	log zerolog.Logger

	ID               int64 `json:"id"`
	NetworkID        int64 `json:"network_id"`
	Name             string
	Enabled          bool `json:"enabled"`
	Password         string
	Topic            string
	ConnectionErrors []string
	Monitoring       bool
	MonitoringSince  time.Time
	LastAnnounce     time.Time
	inviteCommand    string

	users              *haxmap.Map[string, *domain.IrcUser]
	announcers         *haxmap.Map[string, *domain.IrcUser]
	DefaultChannel     bool
	AnnouncerInChannel bool

	Messages *MessageBuffer

	announceProcessor announce.Processor
	stateMachine      *ChannelStateMachine
}

// ChannelSnapshot is a consistent, lock-free copy of a channel's health fields.
type ChannelSnapshot struct {
	ID               int64
	Name             string
	Enabled          bool
	Password         string
	Monitoring       bool
	MonitoringSince  time.Time
	LastAnnounce     time.Time
	ConnectionErrors []string
}

func NewChannel(log zerolog.Logger, networkID int64, name string, defaultChannel bool, announceProcessor announce.Processor) *Channel {
	return &Channel{
		m:                  deadlock.RWMutex{},
		log:                log.With().Str("channel", name).Logger(),
		ID:                 0,
		NetworkID:          networkID,
		Name:               name,
		Enabled:            true,
		Password:           "",
		Topic:              "",
		ConnectionErrors:   make([]string, 0),
		Monitoring:         false,
		MonitoringSince:    time.Time{},
		LastAnnounce:       time.Time{},
		inviteCommand:      "",
		users:              haxmap.New[string, *domain.IrcUser](),
		announcers:         haxmap.New[string, *domain.IrcUser](),
		DefaultChannel:     defaultChannel,
		AnnouncerInChannel: false,
		announceProcessor:  announceProcessor,
		Messages:           NewMessageBuffer(1000), // make opt-in?
	}
}

func (c *Channel) OnMsg(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}

	// parse announce
	nick := msg.Nick()
	//channel := msg.Params[0]
	message := msg.Params[1]

	// clean message
	cleanedMsg := cleanMessage(message)

	// Add message to history, maintaining maximum size
	newMsg := domain.IrcMessage{
		Network: c.NetworkID,
		Channel: c.Name,
		Nick:    nick,
		Message: cleanedMsg,
		Time:    time.Now(),
	}

	c.Messages.AddMessage(newMsg)

	// check if the message is from announce bot, if not return
	if !c.IsValidAnnouncer(nick) {
		c.log.Trace().Str("nick", nick).Str("msg", cleanedMsg).Msg("not a valid announcer, ignoring")

		return
	}

	if err := c.QueueAnnounceLine(cleanedMsg); err != nil {
		return
	}
	c.UpdateLastAnnounce()

	c.log.Debug().Str("nick", nick).Str("msg", cleanedMsg).Msg("got message")
}

// IsValidAnnouncer check if announcer is one from the list in the definition
func (c *Channel) IsValidAnnouncer(nick string) bool {
	nick = strings.ToLower(nick)

	c.m.Lock()
	defer c.m.Unlock()

	announcer, ok := c.announcers.Get(nick)
	if ok {
		if announcer.Present && announcer.State == domain.IrcUserStatePresent {
			// announcer found and is present
			return true
		}

		if !announcer.Present && announcer.State == domain.IrcUserStateUninitialized {
			c.log.Trace().Str("nick", nick).Msg("announcer not present and uninitialized, setting to present")
			announcer.Present = true
			announcer.State = domain.IrcUserStatePresent
			c.announcers.Set(nick, announcer)
			return true
		}

		//if !announcer.Present {
		//	c.log.Warn().Str("nick", nick).Msg("announcer not present")
		//	return false
		//}

		return true
	}

	found := false

	// experimental feature to allow for fuzzy announcer matching. This is not enabled by default because it will allow similar nicks to announce
	if featureflags.IsEnabled(domain.IRCFuzzyAnnouncer) {
		foundFunc := func(s string, user *domain.IrcUser) bool {
			// if nick is not an expected announcer lets check for variants
			if strings.HasPrefix(nick, user.Nick) && len(nick) == len(user.Nick)+1 {
				found = true

				c.log.Warn().Str("nick", nick).Msg("unknown announcer, but valid variant")

				return false // exit foreach on match
			}

			// check if nick is a variant of announcer with * in front
			if strings.HasSuffix(nick, "*") && strings.HasPrefix(nick, user.Nick) {
				found = true

				c.log.Warn().Str("nick", nick).Msg("unknown announcer, but valid variant")

				return false // exit foreach on match
			}

			return true
		}

		c.announcers.ForEach(foundFunc)
	}

	return found
}

func (c *Channel) SetConnectionError(err string) {
	c.m.Lock()
	defer c.m.Unlock()

	for _, existing := range c.ConnectionErrors {
		if existing == err {
			return
		}
	}
	c.ConnectionErrors = append(c.ConnectionErrors, err)
}

func (c *Channel) ClearConnectionErrors() {
	c.m.Lock()
	defer c.m.Unlock()
	c.clearConnectionErrorsLocked()
}

func (c *Channel) clearConnectionErrorsLocked() {
	c.ConnectionErrors = make([]string, 0)
}

func (c *Channel) HasConnectionErrors() bool {
	c.m.RLock()
	defer c.m.RUnlock()
	return len(c.ConnectionErrors) > 0
}

func (c *Channel) SetMonitoring() {
	c.m.Lock()
	defer c.m.Unlock()
	c.Monitoring = true
	c.MonitoringSince = time.Now()
	c.clearConnectionErrorsLocked()
}

func (c *Channel) Reset() {
	c.m.Lock()
	c.Monitoring = false
	c.MonitoringSince = time.Time{}
	c.clearConnectionErrorsLocked()
	c.m.Unlock()

	c.Messages.ClearMessages()
}

func (c *Channel) ResetMonitoring() {
	c.m.Lock()
	defer c.m.Unlock()
	c.Monitoring = false
	c.MonitoringSince = time.Time{}
	c.clearConnectionErrorsLocked()
}

// IsMonitoring reports whether the channel is currently being monitored.
func (c *Channel) IsMonitoring() bool {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.Monitoring
}

// IsEnabled reports whether the channel is enabled.
func (c *Channel) IsEnabled() bool {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.Enabled
}

// GetPassword returns the channel password.
func (c *Channel) GetPassword() string {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.Password
}

// Configure updates the persisted identity/config of the channel.
func (c *Channel) Configure(id int64, enabled bool, password string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.ID = id
	c.Enabled = enabled
	c.Password = password
}

// UpdateIdentity sets the persisted id and enabled flag, preserving other config.
func (c *Channel) UpdateIdentity(id int64, enabled bool) {
	c.m.Lock()
	defer c.m.Unlock()
	c.ID = id
	c.Enabled = enabled
}

// Snapshot returns a consistent copy of the channel's health fields.
func (c *Channel) Snapshot() ChannelSnapshot {
	c.m.RLock()
	defer c.m.RUnlock()
	return ChannelSnapshot{
		ID:               c.ID,
		Name:             c.Name,
		Enabled:          c.Enabled,
		Password:         c.Password,
		Monitoring:       c.Monitoring,
		MonitoringSince:  c.MonitoringSince,
		LastAnnounce:     c.LastAnnounce,
		ConnectionErrors: slices.Clone(c.ConnectionErrors),
	}
}

// Announcers returns a copy of the channel's announcers for reporting.
func (c *Channel) Announcers() []domain.IrcUser {
	c.m.RLock()
	defer c.m.RUnlock()

	out := make([]domain.IrcUser, 0, c.announcers.Len())
	c.announcers.ForEach(func(_ string, u *domain.IrcUser) bool {
		out = append(out, *u)
		return true
	})
	return out
}

func (c *Channel) UpdateLastAnnounce() {
	c.m.Lock()
	defer c.m.Unlock()
	c.LastAnnounce = time.Now()
}

func (c *Channel) RegisterAnnouncers(announcers []string) {
	c.m.Lock()
	defer c.m.Unlock()

	for _, announcer := range announcers {
		announcer = strings.ToLower(announcer)

		c.announcers.Set(announcer, &domain.IrcUser{
			Nick:    announcer,
			Present: false,
			State:   domain.IrcUserStateUninitialized,
		})
	}
}

func (c *Channel) SetInviteCommand(cmd string) {
	c.m.Lock()
	c.inviteCommand = cmd
	c.m.Unlock()

	if c.stateMachine != nil {
		c.stateMachine.SetInviteCommand(cmd)
	}
}

func (c *Channel) InviteCommand() string {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.inviteCommand
}

func (c *Channel) SetStateMachine(sm *ChannelStateMachine) {
	c.stateMachine = sm
}

func (c *Channel) StateMachine() *ChannelStateMachine {
	return c.stateMachine
}

func (c *Channel) SetTopic(topic string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.Topic = topic
}

// SetUsers sets user and announcers on channel
func (c *Channel) SetUsers(users []string) {
	c.m.Lock()
	defer c.m.Unlock()

	for _, nick := range users {
		nick = strings.ToLower(nick)

		u := &domain.IrcUser{Nick: nick}

		// announcers usually have one of these as user mode, but not always
		if strings.ContainsAny(nick, "~!@+&") {
			c.log.Trace().Msgf("usermode %s", nick)

			if ok := u.ParseMode(nick); !ok {
				c.log.Error().Msgf("could not parse mode for nick %s", nick)
				continue
			}

			// we only set special users
			c.users.Set(nick, u)
		}

		// check if user is expected announcer/bot and add to announcers
		if announcer, ok := c.announcers.Get(u.Nick); ok {
			announcer.Present = true
			announcer.State = domain.IrcUserStatePresent
			announcer.Mode = u.Mode

			c.announcers.Set(u.Nick, announcer)
		}

		// we are not interested in all users otherwise we would add them here
		//c.users.Set(nick, u)
	}
}

// RemoveUser remove user and handle announcer status if valid
func (c *Channel) RemoveUser(nick string) {
	nick = strings.ToLower(nick)

	c.m.Lock()
	defer c.m.Unlock()

	// check if user is announcer/bot and remove from announcers
	if announcer, ok := c.announcers.Get(nick); ok {
		announcer.Present = false
		announcer.State = domain.IrcUserStateNotPresent
		c.announcers.Set(nick, announcer)
	}

	c.users.Del(nick)
}

func (c *Channel) QueueAnnounceLine(line string) error {
	if err := c.announceProcessor.AddLineToQueue(c.Name, line); err != nil {
		c.log.Error().Err(err).Msgf("could not add line %s to queue", line)
		return err
	}

	return nil
}
