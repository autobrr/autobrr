package irc

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/featureflags"

	"github.com/alphadose/haxmap"
	"github.com/rs/zerolog"
)

func TestChannel_IsValidAnnouncer(t *testing.T) {
	type fields struct {
		log        zerolog.Logger
		announcers []string
		users      []string
	}
	type args struct {
		nick  string
		users []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test1",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "announce-bot", users: []string{"announce-bot"}},
			want: true,
		},
		{
			name: "test2",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "announce-bot1", users: []string{"announce-bot1"}},
			want: false,
		},
		{
			name: "test3",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "announce-bot*"},
			want: false,
		},
		{
			name: "test3",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "mcbot"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Channel{
				log:        tt.fields.log,
				announcers: haxmap.New[string, *domain.IrcUser](),
				users:      haxmap.New[string, *domain.IrcUser](),
			}

			c.RegisterAnnouncers(tt.fields.announcers)
			c.SetUsers(tt.args.users)

			if got := c.IsValidAnnouncer(tt.args.nick); got != tt.want {
				t.Errorf("IsValidAnnouncer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChannel_IsValidAnnouncer_Exp_Flag(t *testing.T) {
	featureflags.SetEnabled(domain.IRCFuzzyAnnouncer, true)
	type fields struct {
		log        zerolog.Logger
		announcers []string
		users      []string
	}
	type args struct {
		nick  string
		users []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test1",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "announce-bot", users: []string{"announce-bot"}},
			want: true,
		},
		{
			name: "test2",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "announce-bot1", users: []string{"announce-bot1"}},
			want: true,
		},
		{
			name: "test3",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "announce-bot*"},
			want: true,
		},
		{
			name: "test3",
			fields: fields{
				announcers: []string{"announce-bot"},
			},
			args: args{nick: "mcbot"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Channel{
				log:        tt.fields.log,
				announcers: haxmap.New[string, *domain.IrcUser](),
				users:      haxmap.New[string, *domain.IrcUser](),
			}

			c.RegisterAnnouncers(tt.fields.announcers)
			c.SetUsers(tt.args.users)

			if got := c.IsValidAnnouncer(tt.args.nick); got != tt.want {
				t.Errorf("IsValidAnnouncer() = %v, want %v", got, tt.want)
			}
		})
	}
}
