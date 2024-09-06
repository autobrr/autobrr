package irc

import (
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/announce"

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

	Members            map[string]struct{}
	users              *haxmap.Map[string, *User]
	announcers         *haxmap.Map[string, *Announcer]
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
		Members:            make(map[string]struct{}),
		users:              haxmap.New[string, *User](),
		announcers:         haxmap.New[string, *Announcer](),
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

	// publish to SSE stream
	//h.publishSSEMsg(domain.IrcMessage{Channel: channel, Nick: nick, Message: cleanedMsg, Time: time.Now()})

	// check if message is from a valid channel, if not return
	//if validChannel := h.isValidChannel(channel); !validChannel {
	//	return
	//}

	// check if message is from announce bot, if not return
	if !c.IsValidAnnouncer(nick) {
		c.log.Trace().Str("nick", nick).Str("msg", cleanedMsg).Msg("not a valid announcer, ignoring")

		return
	}

	if err := c.QueueAnnounceLine(cleanedMsg); err != nil {
		return
	}
	//c.LastAnnounce = time.Now()
	c.SetLastAnnounce()

	c.log.Debug().Str("nick", nick).Msg(cleanedMsg)
}

// IsValidAnnouncer check if announcer is one from the list in the definition
func (c *Channel) IsValidAnnouncer(nick string) bool {
	_, ok := c.announcers.Get(strings.ToLower(nick))
	return ok
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

func (c *Channel) SetLastAnnounce() {
	c.LastAnnounce = time.Now()
}

func (c *Channel) SetAnnouncers(announcers []string) {
	for _, announcer := range announcers {
		c.announcers.Set(announcer, &Announcer{
			Nick:      announcer,
			InChannel: false,
		})
	}
}

func (c *Channel) SetTopic(topic string) {
	c.Topic = topic
}

func (c *Channel) SetUsers(users []string) {
	for _, nick := range users {
		// check if user is expected announcer/bot and add to announcers
		if announcer, ok := c.announcers.Get(nick); ok {
			announcer.InChannel = true

			c.announcers.Set(nick, announcer)
		}

		c.users.Set(nick, &User{
			Nick: nick,
		})
	}
}

func (c *Channel) RemoveUser(nick string) {
	// check if user is announcer/bot and remove from announcers
	if announcer, ok := c.announcers.Get(nick); ok {
		announcer.InChannel = false
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

type Announcer struct {
	Nick      string `json:"nick"`
	InChannel bool   `json:"in_channel"`
}

type User struct {
	Nick string
	Mode string
}
