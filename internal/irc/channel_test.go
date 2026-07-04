package irc

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/featureflags"

	"github.com/rs/zerolog"
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
			if got := c.IsValidAnnouncer(tt.nick); got != tt.want {
				t.Errorf("IsValidAnnouncer(%q) = %v, want %v", tt.nick, got, tt.want)
			}
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
			if got := c.IsValidAnnouncer(tt.nick); got != tt.want {
				t.Errorf("IsValidAnnouncer(%q) = %v, want %v", tt.nick, got, tt.want)
			}
		})
	}
}
