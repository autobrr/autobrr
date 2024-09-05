package irc

import (
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/announce"

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

	Members            map[string]struct{}
	validAnnouncers    map[string]struct{}
	DefaultChannel     bool
	AnnouncerInChannel bool

	announceProcessor announce.Processor
}

func (c *Channel) OnMsg(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}

	// parse announce
	nick := msg.Nick()
	channel := msg.Params[0]
	message := msg.Params[1]

	// clean message
	cleanedMsg := cleanMessage(message)

	// publish to SSE stream
	//h.publishSSEMsg(domain.IrcMessage{Channel: channel, Nick: nick, Message: cleanedMsg, Time: time.Now()})

	// check if message is from a valid channel, if not return
	//if validChannel := h.isValidChannel(channel); !validChannel {
	//	return
	//}

	// check if message is from announce bot, if not return
	if !c.IsValidAnnouncer(nick) {
		c.log.Trace().Str("channel", channel).Str("nick", nick).Str("msg", cleanedMsg).Msg("not a valid announcer, ignoring")

		return
	}

	if err := c.QueueAnnounceLine(cleanedMsg); err != nil {
		return
	}
	c.LastAnnounce = time.Now()

	c.log.Debug().Str("channel", channel).Str("nick", nick).Msg(cleanedMsg)
}

// IsValidAnnouncer check if announcer is one from the list in the definition
func (c *Channel) IsValidAnnouncer(nick string) bool {
	//c.m.RLock()
	//defer c.m.RUnlock()
	//
	_, ok := c.validAnnouncers[strings.ToLower(nick)]
	return ok
}

func (c *Channel) SetAnnouncers(announcers []string) {
	for _, announcer := range announcers {
		c.validAnnouncers[announcer] = struct{}{}
	}
}

func (c *Channel) QueueAnnounceLine(line string) error {
	if err := c.announceProcessor.AddLineToQueue(c.Name, line); err != nil {
		c.log.Error().Stack().Err(err).Msgf("could not add line %s to queue", line)
		return err
	}

	return nil
}
