// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/avast/retry-go"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircfmt"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

type channelHealth struct {
	m deadlock.RWMutex

	name            string
	monitoring      bool
	monitoringSince time.Time
	lastAnnounce    time.Time
}

// SetLastAnnounce set last announce to now
func (ch *channelHealth) SetLastAnnounce() {
	ch.m.Lock()
	ch.lastAnnounce = time.Now()
	ch.m.Unlock()
}

// SetMonitoring set monitoring and time
func (ch *channelHealth) SetMonitoring() {
	ch.m.Lock()
	ch.monitoring = true
	ch.monitoringSince = time.Now()
	ch.m.Unlock()
}

// resetMonitoring remove monitoring and time
func (ch *channelHealth) resetMonitoring() {
	ch.m.Lock()
	ch.monitoring = false
	ch.monitoringSince = time.Time{}
	ch.lastAnnounce = time.Time{}
	ch.m.Unlock()
}

type Handler struct {
	log                 zerolog.Logger
	sse                 *sse.Server
	network             *domain.IrcNetwork
	releaseSvc          release.Service
	notificationService notification.Service
	announceProcessors  map[string]announce.Processor
	definitions         map[string]*domain.IndexerDefinition

	client *ircevent.Connection
	m      deadlock.RWMutex

	connectedSince       time.Time
	haveDisconnected     bool
	manuallyDisconnected bool

	validAnnouncers map[string]struct{}
	validChannels   map[string]struct{}
	channelHealth   map[string]*channelHealth

	connectionErrors       []string
	failedNickServAttempts int

	authenticated bool
	saslauthed    bool
}

func NewHandler(log zerolog.Logger, sse *sse.Server, network domain.IrcNetwork, definitions []*domain.IndexerDefinition, releaseSvc release.Service, notificationSvc notification.Service) *Handler {
	h := &Handler{
		log:                 log.With().Str("network", network.Server).Logger(),
		sse:                 sse,
		client:              nil,
		network:             &network,
		releaseSvc:          releaseSvc,
		notificationService: notificationSvc,
		definitions:         map[string]*domain.IndexerDefinition{},
		announceProcessors:  map[string]announce.Processor{},
		validAnnouncers:     map[string]struct{}{},
		validChannels:       map[string]struct{}{},
		channelHealth:       map[string]*channelHealth{},
		authenticated:       false,
		saslauthed:          false,
		connectionErrors:    []string{},
	}

	// init indexer, announceProcessor
	h.InitIndexers(definitions)

	return h
}

func (h *Handler) InitIndexers(definitions []*domain.IndexerDefinition) {
	// Networks can be shared by multiple indexers but channels are unique
	// so let's add a new AnnounceProcessor per channel
	for _, definition := range definitions {
		if _, ok := h.definitions[definition.Identifier]; ok {
			continue
		}

		h.definitions[definition.Identifier] = definition

		// indexers can use multiple channels, but it's not common, but let's handle that anyway.
		for _, channel := range definition.IRC.Channels {
			// some channels are defined in mixed case
			channel = strings.ToLower(channel)

			h.announceProcessors[channel] = announce.NewAnnounceProcessor(h.log, h.releaseSvc, definition)

			h.channelHealth[channel] = &channelHealth{
				name:       channel,
				monitoring: false,
			}

			// create map of valid channels
			h.validChannels[channel] = struct{}{}
		}

		// create map of valid announcers
		for _, announcer := range definition.IRC.Announcers {
			h.validAnnouncers[strings.ToLower(announcer)] = struct{}{}
		}
	}
}

func (h *Handler) removeIndexer() {
	// TODO remove validAnnouncers
	// TODO remove validChannels
	// TODO remove definition
	// TODO remove announceProcessor
}

func (h *Handler) Run() error {
	// TODO validate
	// check if network requires nickserv
	// check if network or channels requires invite command

	addr := fmt.Sprintf("%s:%d", h.network.Server, h.network.Port)

	if h.network.UseBouncer && h.network.BouncerAddr != "" {
		addr = h.network.BouncerAddr
	}

	// this used to be TraceLevel but was changed to DebugLevel during connect to see the info without needing to change loglevel
	// we change back to TraceLevel in the handleJoined method.
	subLogger := zstdlog.NewStdLoggerWithLevel(h.log.With().Logger(), zerolog.DebugLevel)

	h.client = &ircevent.Connection{
		Nick:          h.network.Nick,
		User:          h.network.Auth.Account,
		RealName:      h.network.Auth.Account,
		Password:      h.network.Pass,
		Server:        addr,
		KeepAlive:     4 * time.Minute,
		Timeout:       2 * time.Minute,
		ReconnectFreq: 15 * time.Second,
		Version:       "autobrr",
		QuitMessage:   "bye from autobrr",
		Debug:         true,
		Log:           subLogger,
	}

	if h.network.Auth.Mechanism == domain.IRCAuthMechanismSASLPlain {
		if h.network.Auth.Account != "" && h.network.Auth.Password != "" {
			h.client.SASLLogin = h.network.Auth.Account
			h.client.SASLPassword = h.network.Auth.Password
			h.client.SASLOptional = true
			h.client.UseSASL = true
		}
	}

	if h.network.TLS {
		h.client.UseTLS = true
		h.client.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	h.client.AddConnectCallback(h.onConnect)
	h.client.AddDisconnectCallback(h.onDisconnect)

	h.client.AddCallback("MODE", h.handleMode)
	h.client.AddCallback("INVITE", h.handleInvite)
	h.client.AddCallback("366", h.handleJoined)
	h.client.AddCallback("PART", h.handlePart)
	h.client.AddCallback("PRIVMSG", h.onMessage)
	h.client.AddCallback("NOTICE", h.onNotice)
	h.client.AddCallback("NICK", h.onNick)
	h.client.AddCallback("903", h.handleSASLSuccess)

	//h.setConnectionStatus()
	h.saslauthed = false

	if err := func() error {
		// count connect attempts
		connectAttempts := 0
		disconnectTime := time.Now()

		// retry initial connect if network is down
		// using exponential backoff of 15 seconds
		return retry.Do(
			func() error {
				h.log.Debug().Msgf("connect attempt %d", connectAttempts)

				if err := h.client.Connect(); err != nil {
					connectAttempts++
					return err
				}

				if connectAttempts > 0 {
					h.log.Debug().Msgf("connected at attempt (%d) offline for %s", connectAttempts, time.Since(disconnectTime))
					return nil
				}

				return nil
			},
			retry.OnRetry(func(n uint, err error) {
				if n > 0 {
					h.log.Debug().Msgf("%s connect attempt %d", h.network.Name, n)
				}
			}),
			retry.Delay(time.Second*15),
			retry.Attempts(25),
			retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
				return retry.BackOffDelay(n, err, config)
			}),
		)
	}(); err != nil {
		return err
	}

	h.client.Loop()

	return nil
}

func (h *Handler) isOurNick(nick string) bool {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.network.Nick == nick
}

func (h *Handler) isOurCurrentNick(nick string) bool {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.client.CurrentNick() == nick
}

func (h *Handler) setConnectionStatus() {
	h.m.Lock()
	if h.client.Connected() {
		h.connectedSince = time.Now()
	}
	h.m.Unlock()
	//else {
	//	h.connectedSince = time.Time{}
	//	//h.channelHealth = map[string]*channelHealth{}
	//	h.resetChannelHealth()
	//}
}

func (h *Handler) resetConnectionStatus() {
	h.m.Lock()
	h.connectedSince = time.Time{}
	h.resetChannelHealth()
	h.m.Unlock()
}

func (h *Handler) GetNetwork() *domain.IrcNetwork {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.network
}

func (h *Handler) UpdateNetwork(network *domain.IrcNetwork) {
	h.m.Lock()
	h.network = network
	h.m.Unlock()
}

func (h *Handler) SetNetwork(network *domain.IrcNetwork) {
	h.m.Lock()
	h.network = network
	h.m.Unlock()
}

func (h *Handler) AddChannelHealth(channel string) {
	h.m.Lock()
	h.channelHealth[channel] = &channelHealth{
		name:            channel,
		monitoring:      true,
		monitoringSince: time.Now(),
	}
	h.m.Unlock()
}

func (h *Handler) resetChannelHealth() {
	for _, ch := range h.channelHealth {
		ch.resetMonitoring()
	}
}

// Stop the network and quit
func (h *Handler) Stop() {
	h.m.Lock()
	h.connectedSince = time.Time{}
	h.manuallyDisconnected = true

	if h.client.Connected() {
		h.log.Debug().Msg("Disconnecting...")
	}
	h.m.Unlock()

	h.resetChannelHealth()

	h.client.Quit()
}

// Restart stops the network and then runs it
func (h *Handler) Restart() error {
	h.log.Debug().Msg("Restarting network...")
	h.Stop()

	return h.Run()
}

// onConnect is the connect callback
func (h *Handler) onConnect(m ircmsg.Message) {
	// 0. Authenticated via SASL - join
	// 1. No nickserv, no invite command - join
	// 2. Nickserv password - join after auth
	// 3. nickserv and invite command - send nickserv pass, wait for mode to send invite cmd, then join
	// 4. invite command - join

	h.setConnectionStatus()

	func() {
		h.m.Lock()
		if h.haveDisconnected {
			h.notificationService.Send(domain.NotificationEventIRCReconnected, domain.NotificationPayload{
				Subject: "IRC Reconnected",
				Message: fmt.Sprintf("Network: %s", h.network.Name),
			})

			// reset haveDisconnected
			h.haveDisconnected = false
		}
		h.m.Unlock()

		h.log.Debug().Msgf("connected to: %s", h.network.Name)
	}()

	time.Sleep(1 * time.Second)

	h.authenticate()

}

// onDisconnect is the disconnect callback
func (h *Handler) onDisconnect(m ircmsg.Message) {
	h.log.Debug().Msgf("DISCONNECT")

	h.m.Lock()

	// reset connectedSince
	h.connectedSince = time.Time{}

	// reset channelHealth
	for _, ch := range h.channelHealth {
		ch.resetMonitoring()
	}

	// reset authenticated
	h.authenticated = false

	h.haveDisconnected = true

	// check if we are responsible for disconnect
	if !h.manuallyDisconnected {
		// only send notification if we did not initiate disconnect/restart/stop
		h.notificationService.Send(domain.NotificationEventIRCDisconnected, domain.NotificationPayload{
			Subject: "IRC Disconnected unexpectedly",
			Message: fmt.Sprintf("Network: %s", h.network.Name),
		})
	} else {
		// reset
		h.manuallyDisconnected = false
	}
	h.m.Unlock()
}

// onNotice handles NOTICE events
func (h *Handler) onNotice(msg ircmsg.Message) {
	switch msg.Nick() {
	case "NickServ":
		h.handleNickServ(msg)
	}
}

// handleNickServ is called from NOTICE events
func (h *Handler) handleNickServ(msg ircmsg.Message) {
	h.log.Trace().Msgf("NOTICE from nickserv: %v", msg.Params)

	if contains(msg.Params[1],
		"Invalid account credentials",
		"Authentication failed: Invalid account credentials",
		"password incorrect",
	) {
		h.addConnectError("authentication failed: Bad account credentials")
		h.log.Error().Msg("NickServ: authentication failed - bad account credentials")

		// stop network and notify user
		h.Stop()
		return
	}

	if contains(msg.Params[1],
		"Account does not exist",
		"Authentication failed: Account does not exist",
		"isn't registered.", // Nick ANICK isn't registered
	) {
		if h.CurrentNick() == h.PreferredNick() {
			h.addConnectError("authentication failed: account does not exist")

			// stop network and notify user
			h.Stop()
		}
	}

	if contains(msg.Params[1],
		"This nickname is registered and protected",
		"please choose a different nick",
		"choose a different nick",
	) {
		h.authenticate()

		h.failedNickServAttempts++
		if h.failedNickServAttempts >= 3 {
			h.log.Warn().Msgf("NickServ %d failed login attempts", h.failedNickServAttempts)
			h.addConnectError("authentication failed: nick in use and not authenticated")

			// stop network and notify user
			h.Stop()
		}
	}

	// You're now logged in as test-bot
	// Password accepted - you are now recognized.
	if contains(msg.Params[1], "you're now logged in as", "password accepted", "you are now recognized") {
		h.log.Debug().Msgf("NOTICE nickserv logged in: %v", msg.Params)
	}

	// fallback for networks that require both password and nick to NickServ IDENTIFY
	// Invalid parameters. For usage, do /msg NickServ HELP IDENTIFY
	if contains(msg.Params[1], "invalid parameters", "help identify") {
		h.log.Debug().Msgf("NOTICE nickserv invalid: %v", msg.Params)

		if err := h.client.Send("PRIVMSG", "NickServ", fmt.Sprintf("IDENTIFY %s %s", h.network.Auth.Account, h.network.Auth.Password)); err != nil {
			return
		}
	}
}

// authenticate sends NickServIdentify if not authenticated
func (h *Handler) authenticate() bool {
	h.m.RLock()
	defer h.m.RUnlock()

	if h.authenticated {
		return true
	}

	if !h.saslauthed && h.network.Auth.Password != "" {
		h.log.Trace().Msg("on connect not authenticated and password not empty: send nickserv identify")
		if err := h.NickServIdentify(h.network.Auth.Password); err != nil {
			h.log.Error().Stack().Err(err).Msg("error nickserv")
			return false
		}

		return false
	} else {
		h.setAuthenticated()
	}

	// return and wait for NOTICE of nickserv auth
	return true
}

// handleSASLSuccess we get here early so set saslauthed before we hit onConnect
func (h *Handler) handleSASLSuccess(msg ircmsg.Message) {
	h.m.Lock()
	h.saslauthed = true
	h.m.Unlock()
}

// setAuthenticated sets the states for authenticated, connectionErrors, failedNickServAttempts
// and then sends inviteCommand and after that JoinChannels
func (h *Handler) setAuthenticated() {
	h.authenticated = true
	h.connectionErrors = []string{}
	h.failedNickServAttempts = 0

	h.inviteCommand()
	h.JoinChannels()
}

// send invite commands if not empty
func (h *Handler) inviteCommand() {
	if h.network.InviteCommand != "" {
		h.log.Trace().Msg("on connect invite command not empty: send connect commands")
		if err := h.sendConnectCommands(h.network.InviteCommand); err != nil {
			h.log.Error().Stack().Err(err).Msgf("error sending connect command %s", h.network.InviteCommand)
			return
		}
	}
}

func contains(s string, substr ...string) bool {
	s = strings.ToLower(s)
	for _, c := range substr {
		c = strings.ToLower(c)
		if strings.Contains(s, c) {
			return true
		} else if c == s {
			return true
		}
	}
	return false
}

// onNick handles NICK events
func (h *Handler) onNick(msg ircmsg.Message) {
	h.log.Trace().Msgf("NICK event: %s params: %v", msg.Nick(), msg.Params)
	if len(msg.Params) < 1 {
		return
	}

	if msg.Params[0] != h.PreferredNick() {
		return
	}

	if !h.authenticated {
		h.authenticate()
	}
}

func (h *Handler) publishSSEMsg(msg domain.IrcMessage) {
	key := genSSEKey(h.network.ID, msg.Channel)

	h.sse.Publish(key, &sse.Event{
		Data: msg.Bytes(),
	})
}

// onMessage handles PRIVMSG events
func (h *Handler) onMessage(msg ircmsg.Message) {

	if len(msg.Params) < 2 {
		return
	}
	// parse announce
	nick := msg.Nick()
	channel := msg.Params[0]
	message := msg.Params[1]

	// clean message
	cleanedMsg := h.cleanMessage(message)

	// publish to SSE stream
	h.publishSSEMsg(domain.IrcMessage{Channel: channel, Nick: nick, Message: cleanedMsg, Time: time.Now()})

	// check if message is from a valid channel, if not return
	if validChannel := h.isValidChannel(channel); !validChannel {
		return
	}

	// check if message is from announce bot, if not return
	if validAnnouncer := h.isValidAnnouncer(nick); !validAnnouncer {
		return
	}

	h.log.Debug().Str("channel", channel).Str("nick", nick).Msg(cleanedMsg)

	if err := h.sendToAnnounceProcessor(channel, cleanedMsg); err != nil {
		h.log.Error().Stack().Err(err).Msgf("could not queue line: %s", cleanedMsg)
		return
	}

	return
}

// send the msg to announce processor
func (h *Handler) sendToAnnounceProcessor(channel string, msg string) error {
	channel = strings.ToLower(channel)

	// check if queue exists
	queue, ok := h.announceProcessors[channel]
	if !ok {
		return errors.New("queue '%s' not found", channel)
	}

	// if it exists, add msg
	if err := queue.AddLineToQueue(channel, msg); err != nil {
		h.log.Error().Stack().Err(err).Msgf("could not queue line: %s", msg)
		return err
	}

	if v, ok := h.channelHealth[channel]; ok {
		v.SetLastAnnounce()
	}

	return nil
}

// JoinChannels sends multiple join commands
func (h *Handler) JoinChannels() {
	for _, channel := range h.network.Channels {
		if err := h.JoinChannel(channel.Name, channel.Password); err != nil {
			h.log.Error().Stack().Err(err).Msgf("error joining channel %s", channel.Name)
		}
		time.Sleep(1 * time.Second)
	}
}

// JoinChannel sends join command
func (h *Handler) JoinChannel(channel string, password string) error {
	m := ircmsg.Message{
		Command: "JOIN",
		Params:  []string{channel},
	}

	// support channel password
	if password != "" {
		m.Params = []string{channel, password}
	}

	h.log.Debug().Msgf("sending JOIN command %s", strings.Join(m.Params, " "))

	if err := h.client.SendIRCMessage(m); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error handling join: %s", channel)
		return err
	}

	return nil
}

// handlePart listens for PART events
func (h *Handler) handlePart(msg ircmsg.Message) {
	if !h.isOurCurrentNick(msg.Nick()) {
		h.log.Trace().Msgf("PART other user: %+v", msg)
		return
	}

	channel := strings.ToLower(msg.Params[0])
	h.log.Debug().Msgf("PART channel %s", channel)

	// reset monitoring status
	if v, ok := h.channelHealth[channel]; ok {
		v.resetMonitoring()
	}

	// TODO remove announceProcessor

	h.log.Debug().Msgf("Left channel %s", channel)
}

// PartChannel parts/leaves channel
func (h *Handler) PartChannel(channel string) error {
	h.log.Debug().Msgf("Leaving channel %s", channel)

	if err := h.client.Part(channel); err != nil {
		h.log.Error().Err(err).Msgf("error handling part: %s", channel)
		return err
	}

	// TODO remove announceProcessor

	return nil
}

// handleJoined listens for 366 JOIN events
func (h *Handler) handleJoined(msg ircmsg.Message) {
	if !h.isOurCurrentNick(msg.Params[0]) {
		h.log.Trace().Msgf("JOINED other user: %+v", msg)
		return
	}

	// get channel
	channel := strings.ToLower(msg.Params[1])

	h.log.Debug().Msgf("JOINED: %s", channel)

	// check if channel is valid and if not lets part
	if valid := h.isValidHandlerChannel(channel); !valid {
		if err := h.PartChannel(msg.Params[1]); err != nil {
			h.log.Error().Err(err).Msgf("error handling part for unwanted channel: %s", msg.Params[1])
			return
		}
		return
	}

	h.m.Lock()
	// set monitoring on current channelHealth, or add new
	if v, ok := h.channelHealth[channel]; ok {
		if v != nil {
			v.monitoring = true
			v.monitoringSince = time.Now()

			h.log.Trace().Msgf("set monitoring: %s", v.name)
		}

	} else {
		h.channelHealth[channel] = &channelHealth{
			name:            channel,
			monitoring:      true,
			monitoringSince: time.Now(),
		}

		h.log.Trace().Msgf("add channel health monitoring: %s", channel)
	}
	h.m.Unlock()

	// if not valid it's considered an extra channel
	if valid := h.isValidChannel(channel); !valid {
		h.log.Info().Msgf("Joined extra channel %s", channel)
		return
	}

	h.log.Info().Msgf("Monitoring channel %s", channel)

	// reset log level to Trace now that we are monitoring a channel
	h.client.Log = zstdlog.NewStdLoggerWithLevel(h.log.With().Logger(), zerolog.TraceLevel)
}

// sendConnectCommands sends invite commands
func (h *Handler) sendConnectCommands(msg string) error {
	connectCommand := strings.ReplaceAll(msg, "/msg", "")
	connectCommands := strings.Split(connectCommand, ",")

	for _, command := range connectCommands {
		cmd := strings.TrimSpace(command)

		// if there's an extra , (comma) the command will be empty so lets skip that
		if cmd == "" {
			continue
		}

		m := ircmsg.Message{
			Command: "PRIVMSG",
			Params:  strings.Split(cmd, " "),
		}

		h.log.Debug().Msgf("sending connect command: %s", cmd)

		if err := h.client.SendIRCMessage(m); err != nil {
			h.log.Error().Err(err).Msgf("error handling connect command: %v", m)
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

// handleInvite listens for INVITE events
func (h *Handler) handleInvite(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}

	// get channel
	channel := strings.ToLower(msg.Params[1])

	h.log.Trace().Msgf("INVITE from %s to join: %s", msg.Nick(), channel)

	if validChannel := h.isValidHandlerChannel(channel); !validChannel {
		h.log.Trace().Msgf("invite from %s to join: %s - invalid channel, skip joining", msg.Nick(), channel)
		return
	}

	h.log.Debug().Msgf("INVITE from %s, joining %s", msg.Nick(), channel)

	if err := h.client.Join(msg.Params[1]); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error handling join: %s", msg.Params[1])
		return
	}

	return
}

// NickServIdentify sends NickServ Identify commands
func (h *Handler) NickServIdentify(password string) error {
	m := ircmsg.Message{
		Command: "PRIVMSG",
		Params:  []string{"NickServ", "IDENTIFY", password},
	}

	h.log.Debug().Msgf("NickServ: %v", m)

	if err := h.client.SendIRCMessage(m); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error identifying with nickserv: %v", m)
		return err
	}

	return nil
}

// NickChange sets a new nick for our user
func (h *Handler) NickChange(nick string) error {
	h.log.Debug().Msgf("NICK change: %s", nick)

	h.client.SetNick(nick)

	return nil
}

// CurrentNick returns our current nick set by the server
func (h *Handler) CurrentNick() string {
	return h.client.CurrentNick()
}

// PreferredNick returns our preferred nick from settings
func (h *Handler) PreferredNick() string {
	return h.client.PreferredNick()
}

// listens for MODE events
func (h *Handler) handleMode(msg ircmsg.Message) {
	h.log.Trace().Msgf("MODE: %+v", msg)

	// if our nick and user mode +r (Identifies the nick as being Registered (settable by services only)) then return
	if h.isOurCurrentNick(msg.Params[0]) && strings.Contains(msg.Params[1], "+r") {
		if !h.authenticated {
			h.setAuthenticated()
		}

		return
	}

	return
}

func (h *Handler) SendMsg(channel, msg string) error {
	h.log.Debug().Msgf("sending msg command: %s", msg)

	if err := h.client.Privmsg(channel, msg); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error sending msg: %s", msg)
		return err
	}

	return nil
}

// check if announcer is one from the list in the definition
func (h *Handler) isValidAnnouncer(nick string) bool {
	h.m.RLock()
	defer h.m.RUnlock()

	_, ok := h.validAnnouncers[strings.ToLower(nick)]
	return ok
}

// check if channel is one from the list in the definition
func (h *Handler) isValidChannel(channel string) bool {
	h.m.RLock()
	defer h.m.RUnlock()

	_, ok := h.validChannels[strings.ToLower(channel)]
	return ok
}

// check if channel is from definition or user defined
func (h *Handler) isValidHandlerChannel(channel string) bool {
	channel = strings.ToLower(channel)

	h.m.RLock()
	defer h.m.RUnlock()

	if _, ok := h.validChannels[channel]; ok {
		return true
	}

	for _, ircChannel := range h.network.Channels {
		if channel == strings.ToLower(ircChannel.Name) {
			return true
		}
	}

	return false
}

// irc line can contain lots of extra stuff like color so lets clean that
func (h *Handler) cleanMessage(message string) string {
	return ircfmt.Strip(message)
}

func (h *Handler) addConnectError(message string) {
	h.m.Lock()
	defer h.m.Unlock()

	h.connectionErrors = append(h.connectionErrors, message)
}

// Healthy if enabled but not monitoring return false,
//
// if any channel is enabled but not monitoring return false,
// else return true
func (h *Handler) Healthy() bool {
	isHealthy := h.networkHealth()
	if !isHealthy {
		h.log.Warn().Msg("network unhealthy")
		return isHealthy
	}

	h.log.Trace().Msg("network healthy")

	return true
}

func (h *Handler) networkHealth() bool {
	if h.network.Enabled {
		if !h.client.Connected() {
			return false
		}
		if (h.connectedSince == time.Time{}) {
			return false
		}

		for _, channel := range h.network.Channels {
			name := strings.ToLower(channel.Name)

			if chanHealth, ok := h.channelHealth[name]; ok {
				chanHealth.m.RLock()

				if !chanHealth.monitoring {
					chanHealth.m.RUnlock()
					return false
				}

				chanHealth.m.RUnlock()
			}
		}
	}

	return true
}
