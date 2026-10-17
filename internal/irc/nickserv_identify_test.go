// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// noSuchNick builds the ERR_NOSUCHNICK an ircd answers a message to an absent
// nick with.
func noSuchNick(nick, target string) ircmsg.Message {
	return ircmsg.Message{
		Source:  "irc.example.test",
		Command: "401",
		Params:  []string{nick, target, "No such nick/channel"},
	}
}

// withNickServAuth configures a handler waiting for a NICKSERV reply with a bot
// nick that differs from the account, i.e. the setup in issue #2528.
func withNickServAuth(h *Handler, account string) {
	h.network.Nick = "user_bot"
	h.network.Auth = domain.IRCAuth{
		Mechanism: domain.IRCAuthMechanismNickServ,
		Account:   account,
		Password:  "hunter2",
	}
	h.identifyOutstanding = true
}

func TestAuthenticateGatesNickServOnMechanism(t *testing.T) {
	tests := []struct {
		name              string
		auth              domain.IRCAuth
		saslauthed        bool
		wantAuthenticated bool
	}{
		{
			name:              "none ignores leftover password",
			auth:              domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNone, Password: "hunter2"},
			wantAuthenticated: true,
		},
		{
			name: "nickserv password only identifies",
			auth: domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNickServ, Password: "hunter2"},
		},
		{
			name: "sasl falls back when negotiation did not complete",
			auth: domain.IRCAuth{Mechanism: domain.IRCAuthMechanismSASLPlain, Account: "test_bot", Password: "hunter2"},
		},
		{
			name:              "sasl success skips nickserv",
			auth:              domain.IRCAuth{Mechanism: domain.IRCAuthMechanismSASLPlain, Account: "test_bot", Password: "hunter2"},
			saslauthed:        true,
			wantAuthenticated: true,
		},
		{
			name:              "legacy empty mechanism keeps password behavior",
			auth:              domain.IRCAuth{Password: "hunter2"},
			wantAuthenticated: false,
		},
		{
			name:              "unknown mechanism never identifies",
			auth:              domain.IRCAuth{Mechanism: "TYPO", Password: "hunter2"},
			wantAuthenticated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			h.network.Auth = tt.auth
			h.saslauthed = tt.saslauthed

			h.authenticate()

			require.Equal(t, tt.wantAuthenticated, h.authenticated)
			if tt.wantAuthenticated {
				require.False(t, h.identifyOutstanding, "authentication that skips NickServ must not leave an IDENTIFY outstanding")
			}
		})
	}
}

func TestHandleNickServRequiresOutstandingIdentify(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Handler)
	}{
		{name: "none", setup: func(h *Handler) { h.network.Auth.Mechanism = domain.IRCAuthMechanismNone }},
		{name: "not outstanding", setup: func(h *Handler) { h.identifyOutstanding = false }},
		{name: "already authenticated", setup: func(h *Handler) { h.authenticated = true }},
		{name: "authenticated over sasl", setup: func(h *Handler) { h.saslauthed = true }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, "test_bot")
			tt.setup(h)

			h.handleNickServ(nickServNotice("user_bot", "Password incorrect."))

			require.False(t, h.Stopped(), "unsolicited NickServ notice stopped the network")
			require.Empty(t, h.connectionErrors, "unsolicited NickServ notice added errors")
		})
	}
}

func TestOnConnectedSkipsNickServForNone(t *testing.T) {
	h, _ := newTestHandler()
	h.network.Auth = domain.IRCAuth{
		Mechanism: domain.IRCAuthMechanismNone,
		Password:  "leftover",
	}

	h.stateMachine.OnConnected()

	require.NotEqual(t, StateAuthenticating, h.stateMachine.GetState(), "mechanism NONE entered the NickServ authentication state")
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
			assert.Equal(t, tt.want, noticeAllowsIdentifyEscalation(tt.notice))
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

			assert.Equal(t, tt.want, h.canEscalateIdentify(tt.currentNick))
		})
	}
}

// TestIdentifyEscalationLifecycle verifies the ladder is forward-only within a
// connection and re-armed by a reconnect.
func TestIdentifyEscalationLifecycle(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	require.True(t, h.canEscalateIdentify("user_bot"), "expected escalation to be available on a fresh connection")

	h.escalateIdentify()

	assert.Equal(t, identifyFormAccount, h.identifyAttempt, "expected identifyAttempt to be identifyFormAccount after escalation")

	assert.False(t, h.canEscalateIdentify("user_bot"), "expected escalation to be spent after one attempt")

	// a reconnect starts over: the nick we get back may differ
	h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"})

	assert.Equal(t, identifyFormBare, h.identifyAttempt, "expected identifyAttempt to reset to identifyFormBare on disconnect")

	assert.True(t, h.canEscalateIdentify("user_bot"), "expected escalation to be re-armed after a disconnect")
}

// TestIdentifyFormIsStickyPerNetwork verifies a successful escalation is
// remembered: reconnecting must not re-send a bare IDENTIFY already known to
// fail on this network.
func TestIdentifyFormIsStickyPerNetwork(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Nick user_bot isn't registered."))
	h.identifyOutstanding = true
	h.handleNickServ(nickServNotice("user_bot", "Password accepted - you are now recognized."))

	require.True(t, h.authenticated, "expected the escalated identify to authenticate")

	h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"})

	assert.Equal(t, identifyFormAccount, h.identifyAttempt, "expected the reconnect to start on the account-qualified form")

	assert.Equal(t, "IDENTIFY test_bot hunter2", h.identifyCommand())
}

// TestIdentifyFormStaysBareWhenBareWorks keeps the default path untouched: a
// network that authenticates bare must never drift onto the account form.
func TestIdentifyFormStaysBareWhenBareWorks(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Password accepted - you are now recognized."))
	h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"})

	assert.Equal(t, identifyFormBare, h.identifyAttempt, "expected a network that authenticates bare to stay on the bare form")
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

			assert.Equal(t, identifyFormBare, h.identifyFormLearned, "expected the rejected account form to be forgotten")
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

			assert.Equal(t, tt.cleared, h.identifyFormLearned == identifyFormBare, "learned form cleared")
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

	assert.Equal(t, identifyFormAccount, h.identifyAttempt, "expected the notice to escalate to the account-qualified form")

	assert.False(t, h.Stopped(), "expected the network to stay up while the account form is retried")

	assert.Empty(t, h.connectionErrors, "expected no connection errors during escalation")
}

// TestHandleNickServStopsWhenAccountUnknownAfterEscalation verifies the ladder
// terminates: once the account form has been tried, the same notice is fatal.
func TestHandleNickServStopsWhenAccountUnknownAfterEscalation(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Nick user_bot isn't registered."))
	h.identifyOutstanding = true
	h.handleNickServ(nickServNotice("user_bot", "Nick test_bot isn't registered."))

	assert.True(t, h.Stopped(), "expected the network to stop once the account form also failed")

	assert.Truef(t, hasConnectError(h, "account does not exist"), "expected an account-does-not-exist error, got %v", h.connectionErrors)
}

// TestHandleNickServBadCredentialsAfterEscalation covers the Anope 1.8 hazard:
// services that ignore the account argument read it as the password and answer
// "Password incorrect.". The network still stops, but the error must not tell
// the user their credentials are simply wrong.
func TestHandleNickServBadCredentialsAfterEscalation(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Your nick isn't registered."))
	h.identifyOutstanding = true
	h.handleNickServ(nickServNotice("user_bot", "Password incorrect."))

	require.True(t, h.Stopped(), "expected the network to stop on a rejected password")

	assert.Truef(t, hasConnectError(h, "account-qualified"), "expected the ambiguous account-qualified error, got %v", h.connectionErrors)
}

// TestHandleNickServBadCredentialsBare keeps the pre-existing message for a
// plain wrong password, where there is no ambiguity to report.
func TestHandleNickServBadCredentialsBare(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleNickServ(nickServNotice("user_bot", "Password incorrect."))

	require.True(t, h.Stopped(), "expected the network to stop on a rejected password")

	assert.Truef(t, hasConnectError(h, "Bad account credentials"), "expected the bad-credentials error, got %v", h.connectionErrors)
}

// TestHandleNickServNoEscalationWithoutAccount preserves today's behaviour for
// networks where the nick is the account: nothing to escalate to, so an unknown
// nick stays terminal.
func TestHandleNickServNoEscalationWithoutAccount(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "")

	h.handleNickServ(nickServNotice("user_bot", "Nick user_bot isn't registered."))

	assert.Equal(t, identifyFormBare, h.identifyAttempt, "expected no escalation without a configured account")

	assert.True(t, h.Stopped(), "expected the network to stop when there is no account to retry with")
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

			assert.Equal(t, tt.want, h.identifyCommand())
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

			assert.Equal(t, tt.want, h.authenticated)
		})
	}
}

// TestNickServAbsentSchedulesIdentifyRetry is the regression test for #2770: a
// 401 for NickServ left the IDENTIFY outstanding forever, so the connection never
// authenticated, even after services came back.
func TestNickServAbsentSchedulesIdentifyRetry(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")
	h.stateMachine.currentState = StateAuthenticating
	gen := h.identifyRetryGen

	h.handleErrNoSuchNick(noSuchNick("user_bot", "NickServ"))

	assert.False(t, h.identifyOutstanding, "expected the unanswerable IDENTIFY to stop blocking authentication")
	assert.Equal(t, 1, h.identifyRetries)
	assert.NotEqual(t, gen, h.identifyRetryGen, "expected a retry to be scheduled")
	assert.True(t, h.identifyRetryDue(h.identifyRetryGen), "expected the retry to be sent when it comes due")

	assert.False(t, h.Stopped(), "a missing NickServ must not stop the network")
	assert.Empty(t, h.connectionErrors, "a missing NickServ must not be reported as an auth failure")
	assert.Equal(t, StateAuthenticating, h.stateMachine.GetState(), "expected the connection to keep waiting to authenticate")
}

// TestNickServAbsentIgnored covers the 401s that must not schedule an IDENTIFY:
// one we are not waiting on, and one for another nick such as an invite bot.
func TestNickServAbsentIgnored(t *testing.T) {
	tests := []struct {
		name   string
		target string
		setup  func(*Handler)
	}{
		{name: "identify not outstanding", target: "NickServ", setup: func(h *Handler) { h.identifyOutstanding = false }},
		{name: "already authenticated", target: "NickServ", setup: func(h *Handler) { h.authenticated = true }},
		{name: "authenticated over sasl", target: "NickServ", setup: func(h *Handler) { h.saslauthed = true }},
		{name: "another nick", target: "Gatekeeper"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, "test_bot")
			if tt.setup != nil {
				tt.setup(h)
			}
			gen := h.identifyRetryGen

			h.handleErrNoSuchNick(noSuchNick("user_bot", tt.target))

			assert.Equal(t, gen, h.identifyRetryGen, "expected no identify retry to be scheduled")
			assert.Zero(t, h.identifyRetries)
		})
	}
}

// TestIdentifyRetryCancelled verifies a scheduled retry is dropped once it no
// longer belongs to the connection waiting on it.
func TestIdentifyRetryCancelled(t *testing.T) {
	tests := []struct {
		name      string
		supersede func(*Handler)
	}{
		{name: "disconnect", supersede: func(h *Handler) { h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"}) }},
		{name: "stop", supersede: func(h *Handler) { h.Stop() }},
		{
			name: "later miss",
			supersede: func(h *Handler) {
				h.identifyOutstanding = true
				h.handleErrNoSuchNick(noSuchNick("user_bot", "NickServ"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			withNickServAuth(h, "test_bot")
			h.handleErrNoSuchNick(noSuchNick("user_bot", "NickServ"))
			gen := h.identifyRetryGen

			tt.supersede(h)

			assert.False(t, h.identifyRetryDue(gen), "expected the scheduled retry to be cancelled")
		})
	}
}

// TestIdentifyRetryBackoffPerConnection verifies repeated misses walk the
// backoff and a reconnect starts it over: the new connection's services may
// well be back.
func TestIdentifyRetryBackoffPerConnection(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")

	h.handleErrNoSuchNick(noSuchNick("user_bot", "NickServ"))
	h.identifyOutstanding = true
	h.handleErrNoSuchNick(noSuchNick("user_bot", "NickServ"))

	assert.Equal(t, 2, h.identifyRetries, "expected each miss to advance the backoff")

	h.onDisconnect(ircmsg.Message{Command: "DISCONNECT"})

	assert.Zero(t, h.identifyRetries, "expected a reconnect to restart the backoff")
}

// TestIdentifyRetryDefersToOutstandingIdentify covers a retry that comes due
// after another path (a NICK change, bot mode) has already sent an IDENTIFY: a
// second one would only draw a second reply.
func TestIdentifyRetryDefersToOutstandingIdentify(t *testing.T) {
	h, _ := newTestHandler()
	withNickServAuth(h, "test_bot")
	h.handleErrNoSuchNick(noSuchNick("user_bot", "NickServ"))
	gen := h.identifyRetryGen

	h.identifyOutstanding = true

	assert.False(t, h.identifyRetryDue(gen), "expected the retry to leave the outstanding IDENTIFY alone")
}

// TestIdentifyRetryDelay checks the retry follows the connect backoff and stays
// capped however long NickServ is gone.
func TestIdentifyRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int
		base    time.Duration
	}{
		{attempt: 1, base: 15 * time.Second},
		{attempt: 2, base: 30 * time.Second},
		{attempt: 3, base: time.Minute},
		{attempt: 4, base: 2 * time.Minute},
		{attempt: 5, base: 4 * time.Minute},
		{attempt: 6, base: 5 * time.Minute},
		{attempt: 10000, base: 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt %d", tt.attempt), func(t *testing.T) {
			delay := identifyRetryDelay(tt.attempt)

			assert.GreaterOrEqual(t, delay, tt.base)
			assert.Less(t, delay, tt.base+connectRetryMaxJitter)
		})
	}
}
