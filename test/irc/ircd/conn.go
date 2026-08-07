// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

package ircd

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/ergochat/irc-go/ircmsg"
)

// conn is a single client connection and its registration/session state.
type conn struct {
	srv *Server
	nc  net.Conn
	br  *bufio.Reader

	wmu sync.Mutex // guards writes to nc

	nick       string
	user       string
	account    string // set once authenticated (SASL or NickServ); "" = unregistered
	registered bool
	sawCapLS   bool
	capEnded   bool
}

func newConn(s *Server, nc net.Conn) *conn {
	return &conn{srv: s, nc: nc, br: bufio.NewReader(nc)}
}

func (c *conn) close() {
	_ = c.nc.Close()
	c.srv.unregisterConn(c)
}

// send writes a single IRC line (CRLF is appended).
func (c *conn) send(line string) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_, _ = c.nc.Write([]byte(line + "\r\n"))
}

func (c *conn) sendf(format string, args ...any) {
	c.send(fmt.Sprintf(format, args...))
}

// numeric writes a server numeric reply targeted at this client's nick.
func (c *conn) numeric(code int, params ...string) {
	nick := c.nick
	if nick == "" {
		nick = "*"
	}
	line := fmt.Sprintf(":%s %03d %s", serverName, code, nick)
	if len(params) > 0 {
		line += " " + strings.Join(params[:len(params)-1], " ")
		line += " :" + params[len(params)-1]
	}
	c.send(line)
}

func (c *conn) serve() {
	defer c.close()
	for {
		line, err := c.br.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		msg, perr := ircmsg.ParseLine(line)
		if perr != nil {
			continue
		}
		if quit := c.handle(msg); quit {
			return
		}
	}
}

func (c *conn) handle(msg ircmsg.Message) (quit bool) {
	switch strings.ToUpper(msg.Command) {
	case "CAP":
		c.handleCap(msg)
	case "NICK":
		if len(msg.Params) > 0 {
			c.nick = msg.Params[0]
		}
		c.tryRegister()
	case "USER":
		if len(msg.Params) > 0 {
			c.user = msg.Params[0]
		}
		c.tryRegister()
	case "PASS":
		// server password; tests don't require one
	case "AUTHENTICATE":
		c.handleAuthenticate(msg)
	case "PING":
		token := ""
		if len(msg.Params) > 0 {
			token = msg.Params[len(msg.Params)-1]
		}
		c.sendf(":%s PONG %s :%s", serverName, serverName, token)
	case "PONG":
		// keepalive reply; ignore
	case "JOIN":
		c.handleJoin(msg)
	case "PART":
		c.handlePart(msg)
	case "PRIVMSG":
		c.handleMessage(msg, false)
	case "NOTICE":
		c.handleMessage(msg, true)
	case "MODE":
		c.handleMode(msg)
	case "QUIT":
		return true
	default:
		// WHO/WHOIS/USERHOST/LIST/... - not needed by the handler, ignore
	}
	return false
}

func (c *conn) handleCap(msg ircmsg.Message) {
	if len(msg.Params) == 0 {
		return
	}
	switch strings.ToUpper(msg.Params[0]) {
	case "LS":
		c.sawCapLS = true
		c.sendf(":%s CAP * LS :sasl", serverName)
	case "REQ":
		caps := ""
		if len(msg.Params) > 1 {
			caps = msg.Params[1]
		}
		c.sendf(":%s CAP * ACK :%s", serverName, caps)
	case "END":
		c.capEnded = true
		c.tryRegister()
	case "LIST":
		c.sendf(":%s CAP * LIST :", serverName)
	}
}

// handleAuthenticate implements SASL PLAIN.
func (c *conn) handleAuthenticate(msg ircmsg.Message) {
	if len(msg.Params) == 0 {
		return
	}
	arg := msg.Params[0]

	if strings.EqualFold(arg, "PLAIN") {
		c.send("AUTHENTICATE +")
		return
	}

	// arg is the base64 payload: authzid \0 authcid \0 passwd
	raw, err := base64.StdEncoding.DecodeString(arg)
	if err != nil {
		c.numeric(904, "SASL authentication failed")
		return
	}
	parts := strings.Split(string(raw), "\x00")
	if len(parts) != 3 || !c.srv.checkAccount(parts[1], parts[2]) {
		c.numeric(904, "SASL authentication failed")
		return
	}

	c.account = parts[1]
	c.sendf(":%s 900 %s %s!%s %s :You are now logged in as %s", serverName, c.nick, c.nick, userHost, c.account, c.account)
	c.numeric(903, "SASL authentication successful")
}

// tryRegister completes registration once NICK, USER and (if the client started
// CAP negotiation) CAP END have all been received, sending the welcome burst the
// ergochat client waits on before it considers itself connected.
func (c *conn) tryRegister() {
	if c.registered || c.nick == "" || c.user == "" {
		return
	}
	if c.sawCapLS && !c.capEnded {
		return // still negotiating capabilities
	}

	// connect-time ban: reject with 465 before completing registration (no 001), so
	// the client's Connect() fails - modelling a G-Line applied on connect.
	if c.srv.banReason != "" && c.srv.banAtRegistration {
		c.numeric(465, c.srv.banReason)
		_ = c.nc.Close()
		return
	}

	// ERROR lines delivered before the welcome, the way a throttling server
	// refuses a reconnect. The link is deliberately left open afterwards so the
	// test can observe how the client accounts for them without waiting out a
	// reconnect cycle per refusal; a real server sends one and closes here.
	for range c.srv.preRegistrationErrorCount {
		c.sendf("ERROR :%s", c.srv.preRegistrationError)
	}

	c.registered = true
	c.srv.registerConn(c)

	c.sendf(":%s 001 %s :Welcome to the test network, %s", serverName, c.nick, c.nick)
	c.sendf(":%s 002 %s :Your host is %s", serverName, c.nick, serverName)
	c.sendf(":%s 003 %s :This server is for tests", serverName, c.nick)
	c.sendf(":%s 004 %s %s testircd oiwr biklmnopstv", serverName, c.nick, serverName)
	c.sendf(":%s 005 %s CHANTYPES=# PREFIX=(ov)@+ NETWORK=Test CASEMAPPING=ascii :are supported by this server", serverName, c.nick)
	c.sendf(":%s 375 %s :- %s Message of the Day -", serverName, c.nick, serverName)
	c.sendf(":%s 372 %s :- in-process test ircd", serverName, c.nick)
	c.sendf(":%s 376 %s :End of /MOTD command.", serverName, c.nick)

	// modelled ban: registration succeeds, then the server K-Line/G-Lines us with a
	// 465 and closes the link. Sending it post-welcome (rather than rejecting at
	// registration) means the handler's Connect() succeeds and the 465 is delivered
	// on the read loop - exercising handleBanned exactly as an in-session ban would,
	// without tripping ircevent's connect-retry.
	if c.srv.banReason != "" {
		c.numeric(465, c.srv.banReason)
		_ = c.nc.Close()
	}
}

func (c *conn) handleJoin(msg ircmsg.Message) {
	if !c.registered || len(msg.Params) == 0 {
		return
	}
	names := strings.Split(msg.Params[0], ",")
	keys := []string{}
	if len(msg.Params) > 1 {
		keys = strings.Split(msg.Params[1], ",")
	}
	for i, name := range names {
		key := ""
		if i < len(keys) {
			key = keys[i]
		}
		c.joinOne(name, key)
	}
}

func (c *conn) joinOne(name, key string) {
	ch := c.srv.getOrCreateChannel(name)

	c.srv.mu.Lock()
	badKey := ch.key != "" && key != ch.key
	needInvite := ch.inviteOnly && !ch.invited[strings.ToLower(c.nick)]
	needReg := ch.regOnly && c.account == ""
	c.srv.mu.Unlock()

	switch {
	case badKey:
		c.numeric(475, ch.name, "Cannot join channel (+k) - bad key")
		return
	case needInvite:
		c.numeric(473, ch.name, "Cannot join channel (+i) - you must be invited")
		return
	case needReg:
		c.numeric(477, ch.name, "Cannot join channel (+r) - you need a registered nick")
		return
	}

	c.srv.mu.Lock()
	ch.members[strings.ToLower(c.nick)] = c
	ch.joins++
	names := ch.namesList(c.nick)
	c.srv.mu.Unlock()

	// echo the JOIN to the joiner and any other members
	c.srv.broadcast(ch, fmt.Sprintf(":%s!%s JOIN %s", c.nick, userHost, ch.name), nil)

	c.sendf(":%s 353 %s = %s :%s", serverName, c.nick, ch.name, strings.Join(names, " "))
	c.sendf(":%s 366 %s %s :End of /NAMES list.", serverName, c.nick, ch.name)
}

// namesList returns the NAMES entries for a channel: the joining client, other
// members, and any visible announcers (auditorium announcers are omitted).
func (ch *Channel) namesList(self string) []string {
	out := []string{self}
	for nick := range ch.members {
		if !strings.EqualFold(nick, self) {
			out = append(out, nick)
		}
	}
	out = append(out, ch.visible...)
	return out
}

func (c *conn) handlePart(msg ircmsg.Message) {
	if len(msg.Params) == 0 {
		return
	}
	for _, name := range strings.Split(msg.Params[0], ",") {
		ch := c.srv.getOrCreateChannel(name)
		c.srv.broadcast(ch, fmt.Sprintf(":%s!%s PART %s", c.nick, userHost, ch.name), nil)
		c.srv.mu.Lock()
		delete(ch.members, strings.ToLower(c.nick))
		c.srv.mu.Unlock()
	}
}

func (c *conn) handleMessage(msg ircmsg.Message, notice bool) {
	if len(msg.Params) < 2 {
		return
	}
	target := msg.Params[0]
	text := msg.Params[1]

	kind := "PRIVMSG"
	if notice {
		kind = "NOTICE"
	}

	if isChannel(target) {
		ch := c.srv.getOrCreateChannel(target)
		c.srv.broadcast(ch, fmt.Sprintf(":%s!%s %s %s :%s", c.nick, userHost, kind, ch.name, text), c)
		return
	}

	// direct message: NickServ, a virtual bot, another client, or nobody
	if strings.EqualFold(target, "NickServ") {
		c.srv.mu.Lock()
		c.srv.nickservMsgs++
		c.srv.mu.Unlock()
		c.handleNickServ(text)
		return
	}
	if bot := c.srv.getBot(target); bot != nil {
		if bot.OnPrivmsg != nil {
			bot.OnPrivmsg(bot, c.nick, text)
		}
		return
	}
	c.srv.mu.Lock()
	dst := c.srv.conns[strings.ToLower(target)]
	c.srv.mu.Unlock()
	if dst != nil {
		dst.sendf(":%s!%s %s %s :%s", c.nick, userHost, kind, target, text)
		return
	}

	// unknown target - the handler relies on this to detect an absent invite bot
	c.numeric(401, target, "No such nick/channel")
}

// handleNickServ implements a minimal NickServ that accepts IDENTIFY and marks
// the client authenticated, both via a recognisable NOTICE and a +r user mode.
func (c *conn) handleNickServ(text string) {
	fields := strings.Fields(text)
	if len(fields) < 2 || !strings.EqualFold(fields[0], "IDENTIFY") {
		return
	}
	account := c.nick
	if len(fields) >= 3 {
		account = fields[1] // IDENTIFY <account> <password>
	}
	c.account = account

	c.sendf(":NickServ!services@%s NOTICE %s :You are now identified for %s", serverName, c.nick, account)
	c.sendf(":%s MODE %s +r", serverName, c.nick)
}

func (c *conn) handleMode(msg ircmsg.Message) {
	// The handler only queries; reply just enough that nothing blocks. A channel
	// MODE query gets an empty mode reply; a self MODE is echoed back.
	if len(msg.Params) == 0 {
		return
	}
	target := msg.Params[0]
	if isChannel(target) {
		c.sendf(":%s 324 %s %s +", serverName, c.nick, target)
		return
	}
	if len(msg.Params) > 1 {
		c.sendf(":%s!%s MODE %s %s", c.nick, userHost, target, msg.Params[1])
	}
}

func isChannel(target string) bool {
	if target == "" {
		return false
	}
	switch target[0] {
	case '#', '&', '+', '!':
		return true
	default:
		return false
	}
}
