// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "testing"

func TestIRCAuthValidate(t *testing.T) {
	tests := []struct {
		name    string
		auth    IRCAuth
		wantErr bool
	}{
		{name: "none", auth: IRCAuth{Mechanism: IRCAuthMechanismNone}},
		{name: "none retains credentials", auth: IRCAuth{Mechanism: IRCAuthMechanismNone, Account: "old", Password: "old"}},
		{name: "sasl", auth: IRCAuth{Mechanism: IRCAuthMechanismSASLPlain, Account: "user", Password: "secret"}},
		{name: "legacy sasl missing account", auth: IRCAuth{Mechanism: IRCAuthMechanismSASLPlain, Password: "secret"}},
		{name: "legacy sasl missing password", auth: IRCAuth{Mechanism: IRCAuthMechanismSASLPlain, Account: "user"}},
		{name: "nickserv with account", auth: IRCAuth{Mechanism: IRCAuthMechanismNickServ, Account: "user", Password: "secret"}},
		{name: "nickserv password only", auth: IRCAuth{Mechanism: IRCAuthMechanismNickServ, Password: "secret"}},
		{name: "legacy nickserv missing password", auth: IRCAuth{Mechanism: IRCAuthMechanismNickServ, Account: "user"}},
		{name: "legacy empty mechanism", auth: IRCAuth{}},
		{name: "legacy password only", auth: IRCAuth{Password: "secret"}},
		{name: "unknown mechanism", auth: IRCAuth{Mechanism: "TYPO", Password: "secret"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.auth.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIRCAuthNickServEnabled(t *testing.T) {
	tests := []struct {
		name string
		auth IRCAuth
		want bool
	}{
		{name: "none with leftover credentials", auth: IRCAuth{Mechanism: IRCAuthMechanismNone, Password: "secret"}},
		{name: "nickserv password only", auth: IRCAuth{Mechanism: IRCAuthMechanismNickServ, Password: "secret"}, want: true},
		{name: "nickserv without password", auth: IRCAuth{Mechanism: IRCAuthMechanismNickServ}},
		{name: "sasl fallback", auth: IRCAuth{Mechanism: IRCAuthMechanismSASLPlain, Account: "user", Password: "secret"}, want: true},
		{name: "incomplete sasl", auth: IRCAuth{Mechanism: IRCAuthMechanismSASLPlain, Password: "secret"}},
		{name: "legacy empty mechanism", auth: IRCAuth{Password: "secret"}, want: true},
		{name: "unknown mechanism", auth: IRCAuth{Mechanism: "TYPO", Password: "secret"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.auth.NickServEnabled(); got != tt.want {
				t.Fatalf("NickServEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
