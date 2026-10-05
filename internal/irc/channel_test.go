package irc

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/featureflags"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAnnouncerChannel(announcers []string) *Channel {
	c := &Channel{
		log:        zerolog.Nop(),
		announcers: make(map[string]struct{}),
	}
	c.RegisterAnnouncers(announcers)
	return c
}

func TestChannel_IsValidAnnouncer(t *testing.T) {
	tests := []struct {
		name       string
		announcers []string
		nick       string
		want       bool
	}{
		{name: "exact match", announcers: []string{"announce-bot"}, nick: "announce-bot", want: true},
		{name: "one char off", announcers: []string{"announce-bot"}, nick: "announce-bot1", want: false},
		{name: "star variant", announcers: []string{"announce-bot"}, nick: "announce-bot*", want: false},
		{name: "unrelated", announcers: []string{"announce-bot"}, nick: "mcbot", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newAnnouncerChannel(tt.announcers)
			assert.Equal(t, tt.want, c.IsValidAnnouncer(tt.nick))
		})
	}
}

func TestChannel_IsValidAnnouncer_Exp_Flag(t *testing.T) {
	featureflags.SetEnabled(domain.IRCFuzzyAnnouncer, true)
	// reset the global flag so it does not leak into other tests / -count>1 runs
	t.Cleanup(func() { featureflags.SetEnabled(domain.IRCFuzzyAnnouncer, false) })

	tests := []struct {
		name       string
		announcers []string
		nick       string
		want       bool
	}{
		{name: "exact match", announcers: []string{"announce-bot"}, nick: "announce-bot", want: true},
		{name: "one char off matches fuzzy", announcers: []string{"announce-bot"}, nick: "announce-bot1", want: true},
		{name: "star variant matches fuzzy", announcers: []string{"announce-bot"}, nick: "announce-bot*", want: true},
		{name: "unrelated still fails", announcers: []string{"announce-bot"}, nick: "mcbot", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newAnnouncerChannel(tt.announcers)
			assert.Equal(t, tt.want, c.IsValidAnnouncer(tt.nick))
		})
	}
}

func TestChannel_OnMsg_SkipCleanMessage(t *testing.T) {
	const raw = "\x02New Torrent\x02: \x0304Some.Release-GRP\x03"

	tests := []struct {
		name string
		skip bool
		want string
	}{
		{name: "cleans by default", skip: false, want: "New Torrent: Some.Release-GRP"},
		{name: "keeps raw when skipping", skip: true, want: raw},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChannel(zerolog.Nop(), 1, "#chan", true, tt.skip, nil)

			msg := ircmsg.Message{
				Source:  "announce-bot!bot@host",
				Command: "PRIVMSG",
				Params:  []string{"#chan", raw},
			}

			got, ok := c.OnMsg(msg)
			require.True(t, ok, "OnMsg() returned ok = false")
			assert.Equal(t, tt.want, got.Message)

			// the broadcast message must match what went into history
			history := c.Messages.GetMessages()
			require.Len(t, history, 1)
			assert.Equal(t, got.Message, history[0].Message)
		})
	}
}

func TestChannel_QueueAnnounceLine_NoProcessor(t *testing.T) {
	c := NewChannel(zerolog.Nop(), 1, "#nordicbytes", false, false, nil)

	err := c.QueueAnnounceLine("[N]-[TV]-[WEB-DL]-[1080p]-[5.77 GiB]-[Some.Show.S01.1080p.WEB-DL.H.264-GROUP]")
	assert.ErrorIs(t, err, domain.ErrIRCChannelNoAnnounceProcessor)
}
