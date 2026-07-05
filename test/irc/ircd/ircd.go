// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

// Package ircd is a minimal, in-process IRC server for autobrr's integration
// tests. It implements just enough of the protocol for the ergochat/irc-go
// client (used by internal/irc) to register, authenticate (NONE / SASL PLAIN /
// NickServ), join channels, and exchange messages - plus scriptable channel
// modes (+k/+i/+r), auditorium-style hidden announcers, and virtual bots that
// stand in for tracker announcers and invite gatekeepers.
//
// It is deliberately not a conforming IRCd: it models only the behaviour the
// autobrr handler relies on, so tests can reproduce the per-tracker setups in
// irc_test.md (channel modes, invite commands, auth mechanisms) without Docker
// or an external binary.
package ircd

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// serverName is the source used for server-originated messages/numerics.
const serverName = "test.ircd"

// userHost is the fixed user@host mask used in message prefixes. The handler
// only ever matches on the nick, so the mask is cosmetic.
const userHost = "test@ircd.local"

// Server is an in-process IRC server bound to an ephemeral loopback port.
type Server struct {
	tb testing.TB
	ln net.Listener

	mu       sync.Mutex
	channels map[string]*Channel // lowercase name -> channel
	bots     map[string]*Bot     // lowercase nick -> virtual bot
	conns    map[string]*conn    // lowercase nick -> registered real connection
	accounts map[string]string   // lowercase account -> password (SASL/NickServ)

	// requireValidSASL rejects SASL logins that don't match a registered account.
	// Off by default so tests that don't care about credentials just work.
	requireValidSASL bool

	closed chan struct{}
	wg     sync.WaitGroup
}

// Option configures a Server at construction.
type Option func(*Server)

// WithAccount registers an account name/password usable for SASL and NickServ.
func WithAccount(name, password string) Option {
	return func(s *Server) { s.accounts[strings.ToLower(name)] = password }
}

// RequireValidSASL makes SASL logins fail unless they match a registered
// account (see WithAccount). Use it to exercise the SASL-failure path.
func RequireValidSASL() Option {
	return func(s *Server) { s.requireValidSASL = true }
}

// New starts a server on 127.0.0.1:0 and registers cleanup with tb.
func New(tb testing.TB, opts ...Option) *Server {
	tb.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("ircd: listen: %v", err)
	}

	s := &Server{
		tb:       tb,
		ln:       ln,
		channels: map[string]*Channel{},
		bots:     map[string]*Bot{},
		conns:    map[string]*conn{},
		accounts: map[string]string{},
		closed:   make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}

	s.wg.Go(s.acceptLoop)

	tb.Cleanup(s.Close)
	return s
}

// Addr is the host:port the server is listening on.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Host is the listen host (127.0.0.1).
func (s *Server) Host() string {
	host, _, _ := net.SplitHostPort(s.Addr())
	return host
}

// Port is the ephemeral port the server bound.
func (s *Server) Port() int {
	_, p, _ := net.SplitHostPort(s.Addr())
	n, _ := strconv.Atoi(p)
	return n
}

// Close shuts the server down and drops all connections. Safe to call twice.
func (s *Server) Close() {
	select {
	case <-s.closed:
		return
	default:
		close(s.closed)
	}

	_ = s.ln.Close()

	s.mu.Lock()
	conns := make([]*conn, 0, len(s.conns))
	for _, c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	for _, c := range conns {
		c.close()
	}
	s.wg.Wait()
}

func (s *Server) acceptLoop() {
	for {
		nc, err := s.ln.Accept()
		if err != nil {
			return // listener closed on Close()
		}
		c := newConn(s, nc)
		s.wg.Go(c.serve)
	}
}

// ---- channels & modes ----

// Channel is a test channel with scriptable modes.
type Channel struct {
	name       string
	key        string          // +k channel key; "" = none
	inviteOnly bool            // +i
	regOnly    bool            // +r / needs a registered (authenticated) nick
	invited    map[string]bool // nicks that have been INVITE'd (lowercase)
	visible    []string        // extra names shown in NAMES (announcers/bots)
	members    map[string]*conn
	joins      int // count of successful joins (to assert (no) rejoin)
}

// JoinCount returns how many times a client has successfully joined the channel.
// Tests use it to assert that, e.g., a kicked channel is not auto-rejoined.
func (s *Server) JoinCount(channel string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch := s.channels[strings.ToLower(channel)]; ch != nil {
		return ch.joins
	}
	return 0
}

// ChannelOption configures a channel.
type ChannelOption func(*Channel)

// Key sets the channel key (+k); joining requires the matching key.
func Key(k string) ChannelOption { return func(c *Channel) { c.key = k } }

// InviteOnly marks the channel +i; joining requires a prior INVITE.
func InviteOnly() ChannelOption { return func(c *Channel) { c.inviteOnly = true } }

// RegisteredOnly marks the channel +r; joining requires an authenticated nick.
func RegisteredOnly() ChannelOption { return func(c *Channel) { c.regOnly = true } }

// Announcer lists nick in NAMES so the joining client "sees" it present.
func Announcer(nick string) ChannelOption {
	return func(c *Channel) { c.visible = append(c.visible, nick) }
}

// HiddenAnnouncer models an auditorium (+u): the announcer is NOT listed in
// NAMES, so the client never sees it, yet its announces still reach the channel.
func HiddenAnnouncer(string) ChannelOption { return func(c *Channel) {} }

// AddChannel creates (or reconfigures) a channel.
func (s *Server) AddChannel(name string, opts ...ChannelOption) *Channel {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := &Channel{
		name:    name,
		invited: map[string]bool{},
		members: map[string]*conn{},
	}
	for _, o := range opts {
		o(ch)
	}
	s.channels[strings.ToLower(name)] = ch
	return ch
}

// Invite pre-authorises nick to join an +i channel (as if a bot INVITE'd it).
func (s *Server) Invite(nick, channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch := s.channels[strings.ToLower(channel)]; ch != nil {
		ch.invited[strings.ToLower(nick)] = true
	}
}

// getOrCreateChannel returns a configured channel, auto-creating a plain
// (mode-less) one for join targets the test didn't pre-declare.
func (s *Server) getOrCreateChannel(name string) *Channel {
	key := strings.ToLower(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := s.channels[key]
	if ch == nil {
		ch = &Channel{name: name, invited: map[string]bool{}, members: map[string]*conn{}}
		s.channels[key] = ch
	}
	return ch
}

// ---- virtual bots (announcers / invite gatekeepers) ----

// Bot is a scripted virtual user. It is not a real connection: the server
// invokes OnPrivmsg when a real client messages the bot, and the bot can push
// INVITE/NOTICE/PRIVMSG back to clients via its methods.
type Bot struct {
	srv  *Server
	nick string

	// OnPrivmsg is called (on the sender's read goroutine) when a real client
	// sends this bot a PRIVMSG - e.g. an invite command. from is the sender nick.
	OnPrivmsg func(b *Bot, from, text string)
}

// AddBot registers a virtual bot nick.
func (s *Server) AddBot(nick string, onPrivmsg func(b *Bot, from, text string)) *Bot {
	b := &Bot{srv: s, nick: nick, OnPrivmsg: onPrivmsg}
	s.mu.Lock()
	s.bots[strings.ToLower(nick)] = b
	s.mu.Unlock()
	return b
}

// Invite sends an INVITE from the bot to a client for a channel, and authorises
// the join on +i channels.
func (b *Bot) Invite(nick, channel string) {
	b.srv.Invite(nick, channel)
	b.srv.sendToNick(nick, fmt.Sprintf(":%s!%s INVITE %s %s", b.nick, userHost, nick, channel))
}

// Notice sends a NOTICE from the bot to a client (e.g. an invite rejection).
func (b *Bot) Notice(nick, text string) {
	b.srv.sendToNick(nick, fmt.Sprintf(":%s!%s NOTICE %s :%s", b.nick, userHost, nick, text))
}

// Privmsg sends a PRIVMSG from the bot directly to a client.
func (b *Bot) Privmsg(nick, text string) {
	b.srv.sendToNick(nick, fmt.Sprintf(":%s!%s PRIVMSG %s :%s", b.nick, userHost, nick, text))
}

// ForceJoin has the server push a client into a channel on the bot's behalf (see
// Server.ForceJoin). It models a gatekeeper that acknowledges the invite command
// and then force-joins you, rather than sending an INVITE you must act on yourself.
func (b *Bot) ForceJoin(nick, channel string) {
	b.srv.ForceJoin(nick, channel)
}

// ---- scripting helpers used by tests ----

// Announce broadcasts a channel PRIVMSG as if fromNick (an announcer) posted it.
func (s *Server) Announce(channel, fromNick, text string) {
	ch := s.getOrCreateChannel(channel)
	s.broadcast(ch, fmt.Sprintf(":%s!%s PRIVMSG %s :%s", fromNick, userHost, ch.name, text), nil)
}

// Kick removes nick from a channel by an operator and notifies it.
func (s *Server) Kick(channel, nick, by, reason string) {
	ch := s.getOrCreateChannel(channel)
	line := fmt.Sprintf(":%s!%s KICK %s %s :%s", by, userHost, ch.name, nick, reason)
	s.broadcast(ch, line, nil)
	s.sendToNick(nick, line)

	s.mu.Lock()
	delete(ch.members, strings.ToLower(nick))
	s.mu.Unlock()
}

// ForceJoin pushes a client into a channel as if the server joined it (a
// SAJOIN-style force-join): it adds the nick to the channel, echoes the JOIN and
// sends the NAMES reply, bypassing the +i/+k/+r gate checks. This models a
// tracker whose invite bot acknowledges the request and then has the server join
// you, instead of sending an INVITE. No-op if the nick is not a connected client.
func (s *Server) ForceJoin(nick, channel string) {
	ch := s.getOrCreateChannel(channel)

	s.mu.Lock()
	c := s.conns[strings.ToLower(nick)]
	if c == nil {
		s.mu.Unlock()
		return
	}
	ch.members[strings.ToLower(nick)] = c
	ch.joins++
	names := ch.namesList(nick)
	s.mu.Unlock()

	// echo the JOIN to the joiner (and any other members) and send NAMES, mirroring
	// conn.joinOne so the handler's RPL_ENDOFNAMES join detection fires
	s.broadcast(ch, fmt.Sprintf(":%s!%s JOIN %s", nick, userHost, ch.name), nil)
	c.sendf(":%s 353 %s = %s :%s", serverName, nick, ch.name, strings.Join(names, " "))
	c.sendf(":%s 366 %s %s :End of /NAMES list.", serverName, nick, ch.name)
}

// sendToNick delivers a raw line to a registered client by nick (no-op if the
// nick is not a connected client).
func (s *Server) sendToNick(nick, line string) {
	s.mu.Lock()
	c := s.conns[strings.ToLower(nick)]
	s.mu.Unlock()
	if c != nil {
		c.send(line)
	}
}

// broadcast delivers a raw line to every member of a channel, optionally
// excluding one connection (e.g. the sender).
func (s *Server) broadcast(ch *Channel, line string, except *conn) {
	s.mu.Lock()
	members := make([]*conn, 0, len(ch.members))
	for _, c := range ch.members {
		if c != except {
			members = append(members, c)
		}
	}
	s.mu.Unlock()

	for _, c := range members {
		c.send(line)
	}
}

func (s *Server) getBot(nick string) *Bot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bots[strings.ToLower(nick)]
}

func (s *Server) registerConn(c *conn) {
	s.mu.Lock()
	s.conns[strings.ToLower(c.nick)] = c
	s.mu.Unlock()
}

func (s *Server) unregisterConn(c *conn) {
	s.mu.Lock()
	if c.nick != "" && s.conns[strings.ToLower(c.nick)] == c {
		delete(s.conns, strings.ToLower(c.nick))
	}
	for _, ch := range s.channels {
		delete(ch.members, strings.ToLower(c.nick))
	}
	s.mu.Unlock()
}

// checkAccount reports whether authcid/password is acceptable for SASL.
func (s *Server) checkAccount(authcid, password string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	want, ok := s.accounts[strings.ToLower(authcid)]
	if ok {
		return want == password
	}
	// unknown account: accept unless the test demanded strict validation
	return !s.requireValidSASL
}
