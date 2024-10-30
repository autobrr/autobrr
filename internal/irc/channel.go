// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/alphadose/haxmap"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

type Channel struct {
	m   deadlock.RWMutex
	log zerolog.Logger

	ID              int64 `json:"id"`
	Name            string
	Enabled         bool `json:"enabled"`
	Password        string
	Topic           string
	Monitoring      bool
	MonitoringSince time.Time
	LastAnnounce    time.Time

	users              *haxmap.Map[string, *domain.IrcUser]
	announcers         *haxmap.Map[string, *domain.IrcUser]
	DefaultChannel     bool
	AnnouncerInChannel bool

	announceProcessor announce.Processor
}

func NewChannel(log zerolog.Logger, name string, defaultChannel bool, announceProcessor announce.Processor) *Channel {
	return &Channel{
		m:                  deadlock.RWMutex{},
		log:                log.With().Str("channel", name).Logger(),
		ID:                 0,
		Name:               name,
		Enabled:            true,
		Password:           "",
		Topic:              "",
		Monitoring:         false,
		MonitoringSince:    time.Time{},
		LastAnnounce:       time.Time{},
		users:              haxmap.New[string, *domain.IrcUser](),
		announcers:         haxmap.New[string, *domain.IrcUser](),
		DefaultChannel:     defaultChannel,
		AnnouncerInChannel: false,
		announceProcessor:  announceProcessor,
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

	// check if message is from announce bot, if not return
	if !c.IsValidAnnouncer(nick) {
		c.log.Trace().Str("nick", nick).Str("msg", cleanedMsg).Msg("not a valid announcer, ignoring")

		return
	}

	if err := c.QueueAnnounceLine(cleanedMsg); err != nil {
		return
	}
	c.UpdateLastAnnounce()

	c.log.Debug().Str("nick", nick).Msg(cleanedMsg)
}

// IsValidAnnouncer check if announcer is one from the list in the definition
func (c *Channel) IsValidAnnouncer(nick string) bool {
	nick = strings.ToLower(nick)

	announcer, ok := c.announcers.Get(nick)
	if ok {
		if !announcer.Present && announcer.State == domain.IrcUserStateUninitialized {
			c.log.Trace().Str("nick", nick).Msg("announcer not present and uninitialized, setting to present")
			announcer.Present = true
			announcer.State = domain.IrcUserStatePresent
			c.announcers.Set(nick, announcer)
			return true
		}

		if !announcer.Present {
			c.log.Warn().Str("nick", nick).Msg("announcer not present")
			return false
		}

		// announcer found and is present
		return true

	}

	found := false

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

	return found
}

func (c *Channel) SetMonitoring() {
	c.Monitoring = true
	c.MonitoringSince = time.Now()
}

func (c *Channel) ResetMonitoring() {
	c.Monitoring = false
	c.MonitoringSince = time.Time{}

	//c.announceProcessor = nil
}

func (c *Channel) UpdateLastAnnounce() {
	c.LastAnnounce = time.Now()
}

func (c *Channel) RegisterAnnouncers(announcers []string) {
	for _, announcer := range announcers {
		announcer = strings.ToLower(announcer)

		c.announcers.Set(announcer, &domain.IrcUser{
			Nick:    announcer,
			Present: false,
			State:   domain.IrcUserStateUninitialized,
		})
	}
}

func (c *Channel) SetTopic(topic string) {
	c.Topic = topic
}

// SetUsers sets user and announcers on channel
func (c *Channel) SetUsers(users []string) {
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
