package domain

import (
	"testing"
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
