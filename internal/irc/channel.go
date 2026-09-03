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

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

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
// MonitoringSince, LastAnnounce, ConnectionErrors, inviteCommand) and the
// announcers set are guarded by m: they are written from the IRC-callback and
// channel-state-machine goroutines and read from the HTTP/health goroutine.
// NetworkID, Name, DefaultChannel, Messages, announceProcessor and stateMachine
// are set at construction (stateMachine before the channel is made visible) and
// are treated as immutable.
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

	// announcers is the set of registered announcer nicks (from the indexer
	// definition) that are allowed to announce in this channel. autobrr does not
	// track the channel user list or announcer presence - it only validates that
	// an announce line came from a known announcer.
	announcers       map[string]struct{}
	DefaultChannel   bool
	SkipCleanMessage bool

	Messages *MessageBuffer

	announceProcessor announce.Processor
	stateMachine      *ChannelStateMachine
}

// ChannelSnapshot is a consistent, lock-free copy of a channel's health fields.
type ChannelSnapshot struct {
	ID               int64
	Name             string
	Enabled          bool
	DefaultChannel   bool
	Password         string
	Monitoring       bool
	MonitoringSince  time.Time
	LastAnnounce     time.Time
	ConnectionErrors []string
}

func NewChannel(log zerolog.Logger, networkID int64, name string, defaultChannel bool, skipCleanMessage bool, announceProcessor announce.Processor) *Channel {
	return &Channel{
		m:                 deadlock.RWMutex{},
		log:               log.With().Str("channel", name).Logger(),
		ID:                0,
		NetworkID:         networkID,
		Name:              name,
		Enabled:           true,
		Password:          "",
		Topic:             "",
		ConnectionErrors:  make([]string, 0),
		Monitoring:        false,
		MonitoringSince:   time.Time{},
		LastAnnounce:      time.Time{},
		inviteCommand:     "",
		announcers:        make(map[string]struct{}),
		DefaultChannel:    defaultChannel,
		SkipCleanMessage:  skipCleanMessage,
		announceProcessor: announceProcessor,
		Messages:          NewMessageBuffer(1000), // make opt-in?
	}
}

// OnMsg records the message in the channel history and queues it for announce
// parsing. It returns the stored message so callers broadcast the same text
// that went into history, honoring SkipCleanMessage.
func (c *Channel) OnMsg(msg ircmsg.Message) (domain.IrcMessage, bool) {
	if len(msg.Params) < 2 {
		return domain.IrcMessage{}, false
	}

	// parse announce
	nick := msg.Nick()
	//channel := msg.Params[0]
	message := msg.Params[1]

	// optionally skip clean message
	cleanedMsg := message
	if !c.SkipCleanMessage {
		cleanedMsg = cleanMessage(message)
	}

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
		return newMsg, true
	}

	if err := c.QueueAnnounceLine(cleanedMsg); err != nil {
		return newMsg, true
	}
	c.UpdateLastAnnounce()

	c.log.Debug().Str("nick", nick).Str("msg", cleanedMsg).Msg("got message")

	return newMsg, true
}

// IsValidAnnouncer reports whether nick is a registered announcer for this
// channel. It is a plain membership check against the list from the indexer
// definition - autobrr does not track channel membership or presence.
func (c *Channel) IsValidAnnouncer(nick string) bool {
	nick = strings.ToLower(nick)

	c.m.RLock()
	defer c.m.RUnlock()

	if _, ok := c.announcers[nick]; ok {
		return true
	}

	// experimental feature to allow for fuzzy announcer matching. This is not
	// enabled by default because it will allow similar nicks to announce.
	if featureflags.IsEnabled(domain.IRCFuzzyAnnouncer) {
		for announcer := range c.announcers {
			// nick is the announcer with one extra trailing character
			if strings.HasPrefix(nick, announcer) && len(nick) == len(announcer)+1 {
				c.log.Warn().Str("nick", nick).Msg("unknown announcer, but valid variant")
				return true
			}

			// nick is a variant of the announcer with a trailing *
			if strings.HasSuffix(nick, "*") && strings.HasPrefix(nick, announcer) {
				c.log.Warn().Str("nick", nick).Msg("unknown announcer, but valid variant")
				return true
			}
		}
	}

	return false
}

func (c *Channel) SetConnectionError(err string) {
	c.m.Lock()
	defer c.m.Unlock()

	if slices.Contains(c.ConnectionErrors, err) {
		return
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

// ConnectionErrorsCopy returns a copy of the channel's connection errors.
func (c *Channel) ConnectionErrorsCopy() []string {
	c.m.RLock()
	defer c.m.RUnlock()
	return slices.Clone(c.ConnectionErrors)
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

// Snapshot returns a consistent copy of the channel's health fields.
func (c *Channel) Snapshot() ChannelSnapshot {
	c.m.RLock()
	defer c.m.RUnlock()
	return ChannelSnapshot{
		ID:               c.ID,
		Name:             c.Name,
		Enabled:          c.Enabled,
		DefaultChannel:   c.DefaultChannel,
		Password:         c.Password,
		Monitoring:       c.Monitoring,
		MonitoringSince:  c.MonitoringSince,
		LastAnnounce:     c.LastAnnounce,
		ConnectionErrors: slices.Clone(c.ConnectionErrors),
	}
}

func (c *Channel) UpdateLastAnnounce() {
	c.m.Lock()
	defer c.m.Unlock()
	c.LastAnnounce = time.Now()
}

// RegisterAnnouncers adds the given nicks to the set of announcers allowed to
// announce in this channel.
func (c *Channel) RegisterAnnouncers(announcers []string) {
	c.m.Lock()
	defer c.m.Unlock()

	for _, announcer := range announcers {
		c.announcers[strings.ToLower(announcer)] = struct{}{}
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

func (c *Channel) QueueAnnounceLine(line string) error {
	if err := c.announceProcessor.AddLineToQueue(c.Name, line); err != nil {
		c.log.Error().Err(err).Str("line", line).Msg("could not add line to queue")
		return err
	}

	return nil
}
