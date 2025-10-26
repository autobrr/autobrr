package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_ParseMode(t *testing.T) {
	type fields struct {
		Nick string
		Mode string
	}
	type args struct {
		nick string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantOk bool
	}{
		{
			name: "test1",
			fields: fields{
				Nick: "admin",
				Mode: "@",
			},
			args: args{
				nick: "@admin",
			},
			wantOk: true,
		},
		{
			name: "test1",
			fields: fields{
				Nick: "admin",
				Mode: "",
			},
			args: args{
				nick: "admin",
			},
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &IrcUser{
				Nick: tt.fields.Nick,
				Mode: tt.fields.Mode,
			}
			if ok := u.ParseMode(tt.args.nick); ok != tt.wantOk {
				t.Errorf("ParseMode() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestIrcUser_ParseMode(t *testing.T) {
	type fields struct {
		Nick    string
		Mode    string
		Present bool
		State   IrcUserState
	}
	type args struct {
		nick string
	}
	type want struct {
		ok   bool
		mode string
		nick string
	}
	tests := []struct {
		name string
		//fields fields
		args args
		want want
	}{
		{
			name: "good nick",
			//fields: fields{
			//	Nick: "admin",
			//	Mode: "@",
			//},
			args: args{
				nick: "@admin",
			},
			want: want{
				ok:   true,
				mode: "@",
				nick: "admin",
			},
		},
		{
			name: "bad nick",
			//fields: fields{
			//	Nick: "admin",
			//	Mode: "",
			//},
			args: args{
				nick: "admin@",
			},
			want: want{
				ok:   false,
				mode: "",
				nick: "",
			},
		},
		{
			name: "short nick",
			//fields: fields{
			//	Nick: "a",
			//	Mode: "@",
			//},
			args: args{
				nick: "@a",
			},
			want: want{
				ok:   false,
				mode: "@",
				nick: "a",
			},
		},
		{
			name: "short nick",
			//fields: fields{
			//	Nick: "a",
			//	Mode: "@",
			//},
			args: args{
				nick: "@a@",
			},
			want: want{
				ok:   false,
				mode: "",
				nick: "a",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &IrcUser{
				//Nick:    tt.fields.Nick,
				//Mode:    tt.fields.Mode,
				//Present: tt.fields.Present,
				//State:   tt.fields.State,
			}
			ok := u.ParseMode(tt.args.nick)
			assert.Equal(t, tt.want.ok, ok)
			assert.Equal(t, tt.want.mode, u.Mode)
			assert.Equal(t, tt.want.nick, u.Nick)
		})
	}
}
