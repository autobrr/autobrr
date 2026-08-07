// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
)

// hasConnectError reports whether any recorded network-level error contains substr.
func hasConnectError(h *Handler, substr string) bool {
	h.m.RLock()
	defer h.m.RUnlock()

	for _, e := range h.connectionErrors {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

// nickServNotice builds the NOTICE a services package sends us.
func nickServNotice(nick, text string) ircmsg.Message {
	return ircmsg.Message{
		Source:  "NickServ!services@services.example.test",
		Command: "NOTICE",
		Params:  []string{nick, text},
	}
}

// withNickServAuth configures a handler for NICKSERV auth with a bot nick that
// differs from the account, i.e. the setup in issue #2528.
func withNickServAuth(h *Handler, account string) {
	h.network.Nick = "user_bot"
	h.network.Auth = domain.IRCAuth{
		Mechanism: domain.IRCAuthMechanismNickServ,
		Account:   account,
		Password:  "hunter2",
	}
}

// TestAuthenticateGatesNickServOnMechanism verifies IDENTIFY is gated on the
// auth mechanism, not on password presence alone: mechanism NONE with a
// leftover password used to send IDENTIFY on every connect, spamming networks
// without services (the 2026-08 TorrentLeech ban wave). authenticate() marks
// the handler authenticated immediately when it decides not to IDENTIFY, while
// the IDENTIFY path waits for NickServ's reply, so h.authenticated doubles as
// the observable outcome.
func TestAuthenticateGatesNickServOnMechanism(t *testing.T) {
	tests := []struct {
		name         string
		auth         domain.IRCAuth
		saslauthed   bool
		wantIdentify bool
	}{
		{
			name:         "none with leftover password",
			auth:         domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNone, Password: "hunter2"},
			wantIdentify: false,
		},
		{
			name:         "nickserv",
			auth:         domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNickServ, Password: "hunter2"},
			wantIdentify: true,
		},
		{
			name:         "sasl plain falls back when sasl did not complete",
			auth:         domain.IRCAuth{Mechanism: domain.IRCAuthMechanismSASLPlain, Account: "test_bot", Password: "hunter2"},
			wantIdentify: true,
		},
		{
			name:         "sasl plain already authenticated over sasl",
			auth:         domain.IRCAuth{Mechanism: domain.IRCAuthMechanismSASLPlain, Account: "test_bot", Password: "hunter2"},
			saslauthed:   true,
			wantIdentify: false,
		},
		{
			name:         "empty mechanism keeps historical password-only behavior",
			auth:         domain.IRCAuth{Password: "hunter2"},
			wantIdentify: true,
		},
		{
			name:         "nickserv without password",
			auth:         domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNickServ},
			wantIdentify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			h.network.Auth = tt.auth
			h.saslauthed = tt.saslauthed

			h.authenticate()

			if identified := !h.authenticated; identified != tt.wantIdentify {
				t.Errorf("authenticate() took the IDENTIFY path = %v, want %v", identified, tt.wantIdentify)
			}
		})
	}
}

// TestOnConnectedSkipsAuthenticatingWhenMechanismNone covers the state machine
// entry point of the same gate: with mechanism NONE the connection must go
// straight to joining channels instead of parking in Authenticating to wait for
// a NickServ reply that never comes.
func TestOnConnectedSkipsAuthenticatingWhenMechanismNone(t *testing.T) {
	h, _ := newTestHandler()
	h.network.Auth = domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNone, Password: "hunter2"}

	h.stateMachine.OnConnected()

	if state := h.stateMachine.GetState(); state == StateAuthenticating {
		t.Error("expected mechanism NONE to skip the Authenticating state")
	}

	h2, _ := newTestHandler()
	h2.network.Auth = domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNickServ, Password: "hunter2"}

	h2.stateMachine.OnConnected()

	if state := h2.stateMachine.GetState(); state != StateAuthenticating {
		t.Errorf("expected mechanism NICKSERV to authenticate, got state %s", state)
	}
}

// TestNickServNoticesIgnoredWhenServicesDisabled is the regression test for the
// escalation bypass: the IDENTIFY escalation is driven straight from a NOTICE,
// so gating only the connect-time paths still let a nick calling itself NickServ
// pull the stored password out of a network that opted out of services - on the
// services-less networks where mechanism NONE is used, that nick is free for
// anyone to take. A bogus rejection must not stop the network either.
func TestNickServNoticesIgnoredWhenServicesDisabled(t *testing.T) {
	auth := func(m domain.IRCAuthMechanism) domain.IRCAuth {
		return domain.IRCAuth{Mechanism: m, Account: "test_bot", Password: "hunter2"}
	}

	tests := []struct {
		name   string
		auth   domain.IRCAuth
		notice string
	}{
		{"escalation bait on none", auth(domain.IRCAuthMechanismNone), "Nick user_bot isn't registered."},
		{"stop bait on none", auth(domain.IRCAuthMechanismNone), "Password incorrect."},
		{"account bait on none", auth(domain.IRCAuthMechanismNone), "Nick user_bot isn't registered."},
		{"escalation bait without a password", domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNickServ, Account: "test_bot"}, "Nick user_bot isn't registered."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			h.network.Nick = "user_bot"
			h.network.Auth = tt.auth

			h.handleNickServ(nickServNotice("user_bot", tt.notice))

			if h.identifyAttempt != identifyFormBare {
				t.Errorf("credential leak: escalated to %q on a network that does not use services", h.identifyCommand())
			}

			if h.identifyEscalated {
				t.Error("escalation must not be armed when services auth is disabled")
			}

			if h.Stopped() {
				t.Error("an unsolicited nickserv notice must not stop the network")
			}

			if len(h.connectionErrors) != 0 {
				t.Errorf("expected no connection errors, got %v", h.connectionErrors)
			}
		})
	}
}

// TestCanEscalateIdentifyRequiresServicesAuth pins the predicate itself, so the
// gate survives a future caller that reaches it by another route.
func TestCanEscalateIdentifyRequiresServicesAuth(t *testing.T) {
	h, _ := newTestHandler()
	h.network.Nick = "user_bot"
	h.network.Auth = domain.IRCAuth{
		Mechanism: domain.IRCAuthMechanismNone,
		Account:   "test_bot",
		Password:  "hunter2",
	}

	if h.canEscalateIdentify("user_bot") {
		t.Error("canEscalateIdentify must be false when the mechanism is NONE")
	}
}

// TestNoticeAllowsIdentifyEscalation covers the verbatim replies of each
// services package. The "no" cases matter most: escalating on a bad password
// would loop, and escalating on success would be pointless.
func TestNoticeAllowsIdentifyEscalation(t *testing.T) {
	tests := []struct {
		name   string
		notice string
		want   bool
	}{
		{"anope unknown nick", "Nick user_bot isn't registered.", true},
		{"anope 1.8 unknown nick", "Your nick isn't registered.", true},
		{"atheme unknown nick", "\x02user_bot\x02 is not a registered nickname.", true},
		{"atheme insufficient params", "Insufficient parameters for \x02IDENTIFY\x02.", true},
		{"atheme syntax line", "Syntax: IDENTIFY <account> <password>", true},
		{"ergo bad param count", "Invalid parameters. For usage, do /msg NickServ HELP IDENTIFY", true},
		{"anope more info", "\x02/msg NickServ HELP IDENTIFY\x02 for more information.", true},

		{"anope success", "Password accepted - you are now recognized.", false},
		{"atheme success", "You are now identified for \x02test_bot\x02.", false},
		{"anope bad password", "Password incorrect.", false},
		{"atheme bad password", "Invalid password for \x02test_bot\x02.", false},
		{"nick protection warning", "This nickname is registered and protected.", false},
		{"unrelated chatter", "Your vhost has been activated.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noticeAllowsIdentifyEscalation(tt.notice); got != tt.want {
				t.Errorf("noticeAllowsIdentifyEscalation(%q) = %v, want %v", tt.notice, got, tt.want)
			}
		})
	}
}

// TestCanEscalateIdentify covers the state guards that keep escalation to a
// single forward-only attempt per connection.
func TestCanEscalateIdentify(t *testing.T) {
	tests := []struct {
		name        string
		account     string
		currentNick string
		setup       func(h *Handler)
		want        bool
	}{
		{
			name:        "account differs from nick",
			account:     "test_bot",
			currentNick: "user_bot",
			want:        true,
		},
		{
			name:        "account equals nick",
			account:     "user_bot",
			currentNick: "user_bot",
			want:        false,
		},
		{
			name:        "account equals nick case insensitively",
			account:     "User_Bot",
			currentNick: "user_bot",
			want:        false,
		},
		{
			name:        "no account configured",
			account:     "",
			currentNick: "user_bot",
			want:        false,
		},
		{
			name:        "no password configured",
			account:     "test_bot",
			currentNick: "user_bot",
			setup:       func(h *Handler) { h.network.Auth.Password = "" },
			want:        false,
		},
		{
			name:        "already escalated",
			account:     "test_bot",
			currentNick: "user_bot",
			setup:       func(h *Handler) { h.identifyEscalated = true },
			want:        false,
		},
		{
			name:        "already on the account form",
			account:     "test_bot",
			currentNick: "user_bot",
			setup:       func(h *Handler) { h.identifyAttempt = identifyFormAccount },
			want:        false,
		},
		{
			name:        "already authenticated",
			account:     "test_bot",
			currentNick: "user_bot",
			setup:       func(h *Handler) { h.authenticated = true },
			want:        false,
		},
		{
			name:        "authenticated over sasl",
			account:     "test_bot",
			currentNick: "user_bot",
			setup:       func(h *Handler) { h.saslauthed = true },
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, tt.account)
			if tt.setup != nil {
				tt.setup(h)
			}

			if got := h.canEscalateIdentify(tt.currentNick); got != tt.want {
				t.Errorf("canEscalateIdentify(%q) = %v, want %v", tt.currentNick, got, tt.want)
			}
		})
	}
}

// TestIdentifyEscalationLifecycle verifies the ladder is forward-only within a
// connection and re-armed by a reconnect.
func TestIdentifyEscalationLifecycle(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	if !h.canEscalateIdentify("user_bot") {
		t.Fatal("expected escalation to be available on a fresh connection")
	}

	h.escalateIdentify()

	if h.identifyAttempt != identifyFormAccount {
		t.Error("expected identifyAttempt to be identifyFormAccount after escalation")
	}

	if h.canEscalateIdentify("user_bot") {
		t.Error("expected escalation to be spent after one attempt")
	}

	// a reconnect starts over: the nick we get back may differ
	h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"})

	if h.identifyAttempt != identifyFormBare {
		t.Error("expected identifyAttempt to reset to identifyFormBare on disconnect")
	}

	if !h.canEscalateIdentify("user_bot") {
		t.Error("expected escalation to be re-armed after a disconnect")
	}
}

// TestIdentifyFormIsStickyPerNetwork verifies a successful escalation is
// remembered: reconnecting must not re-send a bare IDENTIFY already known to
// fail on this network.
func TestIdentifyFormIsStickyPerNetwork(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Nick user_bot isn't registered."))
	h.handleNickServ(nickServNotice("user_bot", "Password accepted - you are now recognized."))

	if !h.authenticated {
		t.Fatal("expected the escalated identify to authenticate")
	}

	h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"})

	if h.identifyAttempt != identifyFormAccount {
		t.Error("expected the reconnect to start on the account-qualified form")
	}

	if got := h.identifyCommand(); got != "IDENTIFY test_bot hunter2" {
		t.Errorf("identifyCommand() = %q, want the account-qualified form", got)
	}
}

// TestIdentifyFormStaysBareWhenBareWorks keeps the default path untouched: a
// network that authenticates bare must never drift onto the account form.
func TestIdentifyFormStaysBareWhenBareWorks(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Password accepted - you are now recognized."))
	h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"})

	if h.identifyAttempt != identifyFormBare {
		t.Error("expected a network that authenticates bare to stay on the bare form")
	}
}

// TestIdentifyFormUnlearnedAfterRejection covers the heal path: once a
// remembered account form is itself rejected, the next connect starts over at
// bare, so a later nick grouping recovers without user action.
func TestIdentifyFormUnlearnedAfterRejection(t *testing.T) {
	tests := []struct {
		name   string
		notice string
	}{
		{"password rejected", "Password incorrect."},
		{"account unknown", "Nick test_bot isn't registered."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, "test_bot")
			h.identifyFormLearned = identifyFormAccount
			h.identifyAttempt = identifyFormAccount

			h.handleNickServ(nickServNotice("user_bot", tt.notice))

			if h.identifyFormLearned != identifyFormBare {
				t.Error("expected the rejected account form to be forgotten")
			}
		})
	}
}

// TestIdentifyFormClearedOnAuthChange verifies the learned form does not outlive
// an edit to the credentials it was learned for.
func TestIdentifyFormClearedOnAuthChange(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(n *domain.IrcNetwork)
		cleared bool
	}{
		{
			name:    "account changed",
			mutate:  func(n *domain.IrcNetwork) { n.Auth.Account = "someone_else" },
			cleared: true,
		},
		{
			name:    "password changed",
			mutate:  func(n *domain.IrcNetwork) { n.Auth.Password = "correcthorse" },
			cleared: true,
		},
		{
			name:    "mechanism changed",
			mutate:  func(n *domain.IrcNetwork) { n.Auth.Mechanism = domain.IRCAuthMechanismSASLPlain },
			cleared: true,
		},
		{
			name:    "unrelated field changed",
			mutate:  func(n *domain.IrcNetwork) { n.Name = "Renamed" },
			cleared: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, "test_bot")
			h.identifyFormLearned = identifyFormAccount

			updated := *h.network
			tt.mutate(&updated)

			h.UpdateNetwork(&updated)

			if cleared := h.identifyFormLearned == identifyFormBare; cleared != tt.cleared {
				t.Errorf("learned form cleared = %v, want %v", cleared, tt.cleared)
			}
		})
	}
}

// TestHandleNickServEscalatesBeforeStopping is the regression test for #2528:
// "Nick <bot> isn't registered." used to stop the network outright, because the
// account-does-not-exist branch returned before the account-qualified retry was
// ever reached.
func TestHandleNickServEscalatesBeforeStopping(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Nick user_bot isn't registered."))

	if h.identifyAttempt != identifyFormAccount {
		t.Error("expected the notice to escalate to the account-qualified form")
	}

	if h.Stopped() {
		t.Error("expected the network to stay up while the account form is retried")
	}

	if len(h.connectionErrors) != 0 {
		t.Errorf("expected no connection errors during escalation, got %v", h.connectionErrors)
	}
}

// TestHandleNickServStopsWhenAccountUnknownAfterEscalation verifies the ladder
// terminates: once the account form has been tried, the same notice is fatal.
func TestHandleNickServStopsWhenAccountUnknownAfterEscalation(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Nick user_bot isn't registered."))
	h.handleNickServ(nickServNotice("user_bot", "Nick test_bot isn't registered."))

	if !h.Stopped() {
		t.Error("expected the network to stop once the account form also failed")
	}

	if !hasConnectError(h, "account does not exist") {
		t.Errorf("expected an account-does-not-exist error, got %v", h.connectionErrors)
	}
}

// TestHandleNickServBadCredentialsAfterEscalation covers the Anope 1.8 hazard:
// services that ignore the account argument read it as the password and answer
// "Password incorrect.". The network still stops, but the error must not tell
// the user their credentials are simply wrong.
func TestHandleNickServBadCredentialsAfterEscalation(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Your nick isn't registered."))
	h.handleNickServ(nickServNotice("user_bot", "Password incorrect."))

	if !h.Stopped() {
		t.Fatal("expected the network to stop on a rejected password")
	}

	if !hasConnectError(h, "account-qualified") {
		t.Errorf("expected the ambiguous account-qualified error, got %v", h.connectionErrors)
	}
}

// TestHandleNickServBadCredentialsBare keeps the pre-existing message for a
// plain wrong password, where there is no ambiguity to report.
func TestHandleNickServBadCredentialsBare(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Password incorrect."))

	if !h.Stopped() {
		t.Fatal("expected the network to stop on a rejected password")
	}

	if !hasConnectError(h, "Bad account credentials") {
		t.Errorf("expected the bad-credentials error, got %v", h.connectionErrors)
	}
}

// TestHandleNickServNoEscalationWithoutAccount preserves today's behaviour for
// networks where the nick is the account: nothing to escalate to, so an unknown
// nick stays terminal.
func TestHandleNickServNoEscalationWithoutAccount(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "")

	h.handleNickServ(nickServNotice("user_bot", "Nick user_bot isn't registered."))

	if h.identifyAttempt != identifyFormBare {
		t.Error("expected no escalation without a configured account")
	}

	if !h.Stopped() {
		t.Error("expected the network to stop when there is no account to retry with")
	}
}

// TestHandleNickServShortParams guards the Params[1] index against a malformed
// or truncated NOTICE.
func TestHandleNickServShortParams(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(ircmsg.Message{
		Source:  "NickServ!services@services.example.test",
		Command: "NOTICE",
		Params:  []string{"user_bot"},
	})
}

// TestNickServIdentifyCommand verifies the wire form for each ladder state.
// PRIVMSG carries the whole command as a single trailing parameter: splitting it
// across parameters puts the password past the message body, where every ircd
// discards it.
func TestNickServIdentifyCommand(t *testing.T) {
	tests := []struct {
		name    string
		account string
		form    identifyForm
		want    string
	}{
		{"bare", "test_bot", identifyFormBare, "IDENTIFY hunter2"},
		{"account qualified", "test_bot", identifyFormAccount, "IDENTIFY test_bot hunter2"},
		{"account qualified without an account falls back to bare", "", identifyFormAccount, "IDENTIFY hunter2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, tt.account)
			h.identifyAttempt = tt.form

			if got := h.identifyCommand(); got != tt.want {
				t.Errorf("identifyCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestHandleLoggedIn covers RPL_LOGGEDIN, whose source is the server rather than
// a nick, so the target has to come from the parameters.
func TestHandleLoggedIn(t *testing.T) {
	tests := []struct {
		name   string
		params []string
		want   bool
	}{
		{
			name:   "our nick",
			params: []string{"", "user_bot!user_bot@host", "test_bot", "You are now logged in as test_bot"},
			want:   true,
		},
		{
			name:   "another user",
			params: []string{"someone_else", "someone_else!u@h", "someone", "You are now logged in as someone"},
			want:   false,
		},
		{
			name:   "truncated",
			params: []string{"user_bot"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, "test_bot")

			// CurrentNick is "" without a live client, so address the message to it
			params := append([]string(nil), tt.params...)

			h.handleLoggedIn(ircmsg.Message{
				Source:  "irc.example.test",
				Command: "900",
				Params:  params,
			})

			if h.authenticated != tt.want {
				t.Errorf("authenticated = %v, want %v", h.authenticated, tt.want)
			}
		})
	}
}
