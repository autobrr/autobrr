// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alphadose/haxmap"
	"github.com/avast/retry-go"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircfmt"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
	"golang.org/x/net/proxy"
)

var (
	connectionInProgress = errors.New("A connection attempt is already in progress")

	clientDisconnected = errors.New("Message cannot be sent because client is disconnected")

	clientManuallyDisconnected = retry.Unrecoverable(errors.New("IRC client was manually disconnected"))
)

type ircState uint

const (
	ircStopped    ircState = iota // (Handler.client) is nil
	ircConnecting                 // still nil
	ircLive                       // (Handler.client) is non-nil and valid
)

type Handler struct {
	m deadlock.RWMutex

	log                 zerolog.Logger
	sse                 *sse.Server
	network             *domain.IrcNetwork
	releaseSvc          release.Service
	notificationService notification.Service
	definitions         map[string]*domain.IndexerDefinition

	client           *ircevent.Connection
	clientState      ircState
	connectedSince   time.Time
	haveDisconnected bool
	authenticated    bool
	saslauthed       bool

	channels *haxmap.Map[string, *Channel]
	bots     *haxmap.Map[string, *domain.IrcUser]

	connectionErrors       []string
	failedNickServAttempts int

	capabilities map[string]struct{}

	botModeChar string
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
		authenticated:       false,
		saslauthed:          false,
		connectionErrors:    []string{},
		channels:            haxmap.New[string, *Channel](),
		bots:                haxmap.New[string, *domain.IrcUser](),
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

		// TODO reverse range and do over h.network.channels instead?

		// indexers can use multiple channels, but it's not common, but let's handle that anyway.
		for _, channelName := range definition.IRC.Channels {
			// some channels are defined in mixed case
			channelName = strings.ToLower(channelName)

			ircChannel := NewChannel(h.log, channelName, true, announce.NewAnnounceProcessor(h.log.With().Str("channel", channelName).Logger(), h.releaseSvc, definition))

			ircChannel.RegisterAnnouncers(definition.IRC.Announcers)

			h.channels.Set(channelName, ircChannel)
		}

		// look for user defined channels and add
		for _, channel := range h.network.Channels {
			channelName := strings.ToLower(channel.Name)

			if ch, found := h.channels.Get(channelName); found {

				ch.ID = channel.ID
				ch.Enabled = channel.Enabled

				h.channels.Swap(channelName, ch)

				continue
			}

			ircChannel := NewChannel(h.log, channelName, false, nil)

			h.channels.Set(channelName, ircChannel)
		}

		for _, announcer := range definition.IRC.Announcers {
			announcer = strings.ToLower(announcer)
			h.bots.Set(announcer, &domain.IrcUser{Nick: announcer})
		}

		if h.network.InviteCommand != "" {
			connectCommands := strings.Split(strings.ReplaceAll(h.network.InviteCommand, "/msg", ""), ",")
			for _, cmd := range connectCommands {
				cmd = strings.TrimSpace(cmd)

				parts := strings.Split(cmd, " ")
				if len(parts) > 2 {
					nick := strings.ToLower(parts[0])
					h.log.Debug().Msgf("invite command: %s bot %s", cmd, nick)

					h.bots.Set(nick, &domain.IrcUser{Nick: nick, State: domain.IrcUserStateUninitialized})
				}
			}
		}
	}
}

func (h *Handler) removeIndexer() {
	// TODO remove validAnnouncers
	// TODO remove validChannels
	// TODO remove definition
	// TODO remove announceProcessor
}

func (h *Handler) Run() (err error) {
	// TODO validate
	// check if network requires nickserv
	// check if network or channels requires invite command

	addr := fmt.Sprintf("%s:%d", h.network.Server, h.network.Port)

	if h.network.UseBouncer && h.network.BouncerAddr != "" {
		addr = h.network.BouncerAddr
	}

	// this used to be TraceLevel but was changed to DebugLevel during connect to see the info without needing to change loglevel
	// we change back to TraceLevel in the handleJoined method.
	subLogger := zstdlog.NewStdLoggerWithLevel(h.log.With().Logger(), zerolog.TraceLevel)

	shouldConnect := false
	h.m.Lock()
	if h.clientState == ircStopped {
		shouldConnect = true
		h.clientState = ircConnecting
	}
	h.m.Unlock()

	if !shouldConnect {
		return connectionInProgress
	}

	// either we will successfully transition to `ircLive`, or else
	// we need to reset the state to `ircStopped`
	defer func() {
		h.m.Lock()
		if h.clientState == ircConnecting {
			h.clientState = ircStopped
		}
		h.m.Unlock()
	}()

	client := &ircevent.Connection{
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

	if h.network.UseProxy && h.network.Proxy != nil {
		if !h.network.Proxy.Enabled {
			h.log.Debug().Msgf("proxy disabled, skip")
		} else {
			if h.network.Proxy.Addr == "" {
				return errors.New("proxy addr missing")
			}

			proxyUrl, err := url.Parse(h.network.Proxy.Addr)
			if err != nil {
				return errors.Wrap(err, "could not parse proxy url: %s", h.network.Proxy.Addr)
			}

			// set user and pass if not empty
			if h.network.Proxy.User != "" && h.network.Proxy.Pass != "" {
				proxyUrl.User = url.UserPassword(h.network.Proxy.User, h.network.Proxy.Pass)
			}

			proxyDialer, err := proxy.FromURL(proxyUrl, proxy.Direct)
			if err != nil {
				return errors.Wrap(err, "could not create proxy dialer from url: %s", h.network.Proxy.Addr)
			}
			proxyContextDialer, ok := proxyDialer.(proxy.ContextDialer)
			if !ok {
				return errors.Wrap(err, "proxy dialer does not expose DialContext(): %v", proxyDialer)
			}

			client.DialContext = proxyContextDialer.DialContext
		}
	}

	if h.network.Auth.Mechanism == domain.IRCAuthMechanismSASLPlain {
		if h.network.Auth.Account != "" && h.network.Auth.Password != "" {
			client.SASLLogin = h.network.Auth.Account
			client.SASLPassword = h.network.Auth.Password
			client.SASLOptional = true
			client.UseSASL = true
		}
	}

	if h.network.TLS {
		// In Go 1.22 old insecure ciphers was removed. A lot of old IRC networks still uses those, so we need to allow those.
		unsafeCipherSuites := make([]uint16, 0, len(tls.InsecureCipherSuites())+len(tls.CipherSuites()))
		for _, suite := range tls.InsecureCipherSuites() {
			unsafeCipherSuites = append(unsafeCipherSuites, suite.ID)
		}
		for _, suite := range tls.CipherSuites() {
			unsafeCipherSuites = append(unsafeCipherSuites, suite.ID)
		}

		client.UseTLS = true
		client.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS10,
			CipherSuites:       unsafeCipherSuites,
		}
	}

	client.AddConnectCallback(h.onConnect)
	client.AddDisconnectCallback(h.onDisconnect)

	client.AddCallback("MODE", h.handleMode)
	if h.network.BotMode {
		client.AddCallback(ircevent.ERR_UMODEUNKNOWNFLAG, h.handleModeUnknownFlag)
	}
	client.AddCallback("INVITE", h.handleInvite)
	client.AddCallback("PART", h.handlePart)
	client.AddCallback("PRIVMSG", h.onPrivMessage)
	client.AddCallback("NOTICE", h.onNotice)
	client.AddCallback("NICK", h.onNick)
	client.AddCallback("KICK", h.onKick)
	client.AddCallback("JOIN", h.handleJoin)

	client.AddCallback(ircevent.RPL_TOPIC, h.handleTopic)
	client.AddCallback(ircevent.RPL_NAMREPLY, h.handleNames)
	client.AddCallback(ircevent.RPL_ENDOFNAMES, h.handleJoined) // end of names
	client.AddCallback(ircevent.RPL_SASLSUCCESS, h.handleSASLSuccess)
	//client.AddCallback(ircevent.ERR_SASLFAIL, h.handleSASLFail)

	//h.setConnectionStatus()
	h.saslauthed = false

	h.client = client

	if err := func() error {
		// count connect attempts
		connectAttempts := 0
		disconnectTime := time.Now()

		// retry initial connect if network is down
		// using exponential backoff of 15 seconds
		return retry.Do(
			func() error {
				h.log.Debug().Msgf("connect attempt %d", connectAttempts)

				// #1239: don't retry if the user manually disconnected with Stop()
				h.m.RLock()
				manuallyDisconnected := h.clientState == ircStopped
				h.m.RUnlock()

				if manuallyDisconnected {
					return clientManuallyDisconnected
				}

				if err := client.Connect(); err != nil {
					h.log.Error().Err(err).Msg("client encountered connection error")
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

	shouldDisconnect := false
	h.m.Lock()
	switch h.clientState {
	case ircStopped:
		// concurrent Stop(), bail
		shouldDisconnect = true
	case ircConnecting:
		// success!
		//h.client = client
		h.clientState = ircLive
	case ircLive:
		// impossible
		h.log.Error().Stack().Msgf("two concurrent connection attempts detected")
		shouldDisconnect = true
	}
	h.m.Unlock()

	if shouldDisconnect {
		client.Quit()
		return clientManuallyDisconnected
	}

	go client.Loop()

	return nil
}

func (h *Handler) isOurNick(nick string) bool {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.network.Nick == nick
}

func (h *Handler) isOurCurrentNick(nick string) bool {
	return h.CurrentNick() == nick
}

func (h *Handler) setConnectionStatus() {
	h.m.Lock()
	if h.client != nil && h.client.Connected() {
		h.connectedSince = time.Now()
	}
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

func (h *Handler) resetChannelState() {
	h.channels.ForEach(func(s string, channel *Channel) bool {
		channel.ResetMonitoring()

		h.channels.Set(s, channel)

		return true
	})
}

// Stop the network and quit
func (h *Handler) Stop() {
	h.m.Lock()
	h.connectedSince = time.Time{}
	client := h.client
	h.clientState = ircStopped
	h.client = nil
	h.m.Unlock()

	if client != nil {
		h.log.Debug().Msg("Disconnecting...")
		h.resetChannelState()
		client.Quit()
	}
}

func (h *Handler) Stopped() bool {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.clientState == ircStopped
}

// Restart stops the network and then runs it
func (h *Handler) Restart() error {
	h.Stop()

	time.Sleep(2 * time.Second)

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
		if h.haveDisconnected && h.clientState == ircLive {
			h.log.Info().Msgf("network re-connected after unexpected disconnect: %s", h.network.Name)

			h.notificationService.Send(domain.NotificationEventIRCReconnected, domain.NotificationPayload{
				Subject: "IRC Reconnected",
				Message: fmt.Sprintf("Network: %s", h.network.Name),
			})

			// reset haveDisconnected
			h.haveDisconnected = false
		}
		h.m.Unlock()

		h.log.Info().Msgf("network connected to: %s", h.network.Name)
	}()

	time.Sleep(1 * time.Second)

	if h.network.BotMode && h.botModeSupported() {
		// if we set Bot Mode, we'll try to authenticate after the MODE response
		h.setBotMode()
		return
	}

	h.authenticate()
}

// onDisconnect is the disconnect callback
func (h *Handler) onDisconnect(m ircmsg.Message) {
	h.log.Debug().Msgf("DISCONNECT")

	h.m.Lock()

	// reset connectedSince
	h.connectedSince = time.Time{}

	// reset authenticated
	h.authenticated = false

	h.haveDisconnected = true

	manuallyDisconnected := h.clientState == ircStopped

	h.m.Unlock()

	// reset channels monitored status
	h.channels.ForEach(func(s string, channel *Channel) bool {
		channel.ResetMonitoring()

		h.channels.Set(s, channel)

		return true
	})

	// check if we are responsible for disconnect
	if !manuallyDisconnected {
		// only send notification if we did not initiate disconnect/restart/stop
		h.notificationService.Send(domain.NotificationEventIRCDisconnected, domain.NotificationPayload{
			Subject: "IRC Disconnected unexpectedly",
			Message: fmt.Sprintf("Network: %s", h.network.Name),
		})
	}

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

		h.Send("PRIVMSG", "NickServ", fmt.Sprintf("IDENTIFY %s %s", h.network.Auth.Account, h.network.Auth.Password))
	}
}

func (h *Handler) getClient() *ircevent.Connection {
	//h.m.RLock()
	client := h.client
	//h.m.RUnlock()
	return client
}

func (h *Handler) Send(command string, params ...string) error {
	if client := h.getClient(); client != nil {
		return client.Send(command, params...)
	} else {
		return clientDisconnected
	}
}

// botModeSupported checks if IRCv3 Bot Mode is supported by the server
// See https://ircv3.net/specs/extensions/bot-mode
func (h *Handler) botModeSupported() bool {
	h.botModeChar = h.client.ISupport()["BOT"]

	return h.botModeChar != ""
}

// setBotMode attempts to set Bot Mode on ourselves
// See https://ircv3.net/specs/extensions/bot-mode
func (h *Handler) setBotMode() {
	h.client.Send("MODE", h.CurrentNick(), "+"+h.botModeChar)
}

// authenticate sends NickServIdentify if not authenticated
func (h *Handler) authenticate() {
	h.m.RLock()
	shouldSendNickserv := !h.authenticated && !h.saslauthed && h.network.Auth.Password != ""
	h.m.RUnlock()

	if shouldSendNickserv {
		h.log.Trace().Msg("on connect not authenticated and password not empty: send nickserv identify")
		h.NickServIdentify(h.network.Auth.Password)
	} else {
		h.setAuthenticated()
	}
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
	h.m.Lock()
	alreadyAuthenticated := h.authenticated
	if !alreadyAuthenticated {
		h.authenticated = true
		h.connectionErrors = []string{}
		h.failedNickServAttempts = 0
	}
	h.m.Unlock()

	if alreadyAuthenticated {
		return
	}

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

	h.authenticate()
}

func (h *Handler) onKick(msg ircmsg.Message) {
	h.log.Trace().Msgf("KICK event: %s params: %v", msg.Nick(), msg.Params)
	if len(msg.Params) < 1 {
		return
	}

	if !h.isOurNick(msg.Params[1]) {
		return
	}

	channelName := strings.ToLower(msg.Params[0])

	channel, found := h.channels.Get(channelName)
	if !found {
		return
	}
	channel.ResetMonitoring()

	// TODO set again or swap?
	h.channels.Swap(channelName, channel)
}

func (h *Handler) publishSSEMsg(msg domain.IrcMessage) {
	key := genSSEKey(h.network.ID, msg.Channel)

	h.sse.Publish(key, &sse.Event{
		Data: msg.Bytes(),
	})
}

// onPrivMessage handles PRIVMSG events
func (h *Handler) onPrivMessage(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}
	// parse announce
	nick := msg.Nick()
	channel := strings.ToLower(msg.Params[0])
	message := msg.Params[1]

	// clean message
	cleanedMsg := cleanMessage(message)

	if message == "CLIENTINFO" {
		return
	}

	if channel == h.CurrentNick() {
		// this is a DM from another user
		h.log.Debug().Str("direct-message", channel).Str("from-nick", nick).Msg(cleanedMsg)

		//h.SendMsg(nick, fmt.Sprintf("pingpong: %s", cleanedMsg))

		// TODO create buffer with user/invite bot
		return
	}

	ircChannel, found := h.channels.Get(channel)
	if !found {
		h.log.Error().Msgf("channel %s not found", channel)
		return
	}

	ircChannel.OnMsg(msg)

	// publish to SSE stream
	h.publishSSEMsg(domain.IrcMessage{Channel: channel, Nick: nick, Message: cleanedMsg, Time: time.Now()})

	//h.log.Debug().Str("channel", channel).Str("nick", nick).Msg(cleanedMsg)

	return
}

// JoinChannels sends multiple join commands
func (h *Handler) JoinChannels() {
	h.channels.ForEach(func(s string, channel *Channel) bool {
		// TODO check if enabled
		if err := h.JoinChannel(channel.Name, channel.Password); err != nil {
			h.log.Error().Stack().Err(err).Msgf("error joining channel %s", channel.Name)
		}
		time.Sleep(1 * time.Second)
		return true
	})
}

// JoinChannel sends join command
func (h *Handler) JoinChannel(channel string, password string) error {
	params := []string{channel}
	// support channel password
	if password != "" {
		params = append(params, password)
	}

	h.log.Debug().Msgf("sending JOIN command %s", strings.Join(params, " "))

	return h.Send("JOIN", params...)
}

// handleJoin listens for JOIN events
func (h *Handler) handleJoin(msg ircmsg.Message) {
	channel := strings.ToLower(msg.Params[0])

	ircChannel, found := h.channels.Get(channel)
	if !found {
		if h.isOurCurrentNick(msg.Nick()) {
			h.log.Debug().Msgf("Joined unwanted channel %s, lets part it..", channel)

			if err := h.PartChannel(channel); err != nil {
				h.log.Error().Stack().Err(err).Msgf("error parting channel %s", channel)
			}
		}

		return
	}

	if !h.isOurCurrentNick(msg.Nick()) {
		h.log.Trace().Msgf("JOIN channel %s other user: %s", channel, msg.Nick())

		ircChannel.SetUsers([]string{msg.Nick()})

		// TODO set or swap ircChannel on handler?
		//h.channels.Swap(channel, ircChannel)

		botUser, foundBot := h.bots.Get(msg.Nick())
		if foundBot {
			botUser.Present = true
			botUser.State = domain.IrcUserStatePresent
			h.bots.Swap(msg.Nick(), botUser)
		}

		return
	}

	h.log.Info().Msgf("Join channel %s", channel)
}

// handlePart listens for PART events
func (h *Handler) handlePart(msg ircmsg.Message) {
	channel := strings.ToLower(msg.Params[0])

	if !h.isOurCurrentNick(msg.Nick()) {
		h.log.Trace().Msgf("PART other user: %+v", msg)

		ircChannel, found := h.channels.Get(channel)
		if !found {
			return
		}

		ircChannel.RemoveUser(msg.Nick())

		botUser, foundBot := h.bots.Get(msg.Nick())
		if foundBot {
			botUser.Present = false
			botUser.State = domain.IrcUserStateNotPresent
			h.bots.Swap(msg.Nick(), botUser)
		}

		return
	}

	h.log.Debug().Msgf("PART channel %s", channel)

	ircChannel, found := h.channels.Get(channel)
	if !found {
		return
	}

	ircChannel.ResetMonitoring()

	h.channels.Swap(channel, ircChannel)

	h.log.Debug().Msgf("Left channel %s", channel)
}

// PartChannel parts/leaves channel
func (h *Handler) PartChannel(channel string) error {
	// if using bouncer we do not want to part any channels
	if h.network.UseBouncer {
		h.log.Debug().Msgf("using bouncer, skip part channel %s", channel)
		return nil
	}

	h.log.Debug().Msgf("Leaving channel %s", channel)

	return h.Send("PART", channel)

	// TODO remove announceProcessor
}

// handleTopic listens for 332
func (h *Handler) handleTopic(msg ircmsg.Message) {
	channel := strings.ToLower(msg.Params[1])
	topic := msg.Params[2]

	h.log.Trace().Msgf("TOPIC: %s %s", channel, topic)

	// set topic for channel
	ircChannel, found := h.channels.Get(channel)
	if found {
		h.log.Trace().Msgf("set channel %s topic: %s", ircChannel.Name, topic)

		ircChannel.SetTopic(topic)

		h.channels.Swap(channel, ircChannel)

		return
	}
}

// handleNames listens for ircevent.RPL_NAMREPLY
func (h *Handler) handleNames(msg ircmsg.Message) {
	channel := strings.ToLower(msg.Params[2])

	if len(msg.Params) >= 3 {
		names := strings.ToLower(msg.Params[3])

		h.log.Trace().Msgf("CHANNEL NAMES START: %s %s", channel, names)

		ircChannel, found := h.channels.Get(channel)
		if found {
			ircChannel.SetUsers(strings.Split(names, " "))

			h.channels.Swap(channel, ircChannel)

			h.log.Trace().Msgf("set names: %s", ircChannel.Name)
		}
	}
}

// handleJoined listens for ENF OF NAMES event, this is where we know we are monitoring a channel
func (h *Handler) handleJoined(msg ircmsg.Message) {
	if !h.isOurCurrentNick(msg.Params[0]) {
		h.log.Trace().Msgf("JOINED other user: %+v", msg)
		return
	}

	// get channel
	channel := strings.ToLower(msg.Params[1])

	h.log.Debug().Msgf("JOINED: %s", channel)

	// check if channel is valid and if not lets part
	ircChannel, found := h.channels.Get(channel)
	if found {
		ircChannel.SetMonitoring()

		h.channels.Swap(channel, ircChannel)

		h.log.Trace().Msgf("set monitoring: %s", ircChannel.Name)

		if ircChannel.DefaultChannel {
			h.log.Info().Msgf("Monitoring channel %s", channel)
		} else {
			h.log.Info().Msgf("Joined extra channel %s", channel)
		}

		return
	}
}

// sendConnectCommands sends invite commands
func parseInviteCommands(msg string) ([]string, error) {
	connectCommands := strings.Split(strings.ReplaceAll(msg, "/msg", ""), ",")

	parsedCommands := make([]string, 0)

	for _, command := range connectCommands {
		cmd := strings.TrimSpace(command)

		// if there's an extra , (comma) the command will be empty so lets skip that
		if cmd == "" {
			continue
		}

		parsedCommands = append(parsedCommands, cmd)

		//params := strings.SplitN(cmd, " ", 2)

		//if err := h.Send("PRIVMSG", params...); err != nil {
		//	h.log.Error().Err(err).Msgf("error handling connect command: %s", cmd)
		//	return nil, err
		//}

	}

	return parsedCommands, nil
}

// sendConnectCommands sends invite commands
func (h *Handler) sendConnectCommands(msg string) error {
	connectCommands := strings.Split(strings.ReplaceAll(msg, "/msg", ""), ",")

	for _, command := range connectCommands {
		cmd := strings.TrimSpace(command)

		// if there's an extra , (comma) the command will be empty so lets skip that
		if cmd == "" {
			continue
		}

		h.log.Debug().Msgf("sending connect command: %s", cmd)

		params := strings.SplitN(cmd, " ", 2)

		if err := h.Send("PRIVMSG", params...); err != nil {
			h.log.Error().Err(err).Msgf("error handling connect command: %s", cmd)
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

	_, found := h.channels.Get(channel)
	if !found {
		h.log.Trace().Msgf("invite from %s to join: %s - unwanted channel, skip joining", msg.Nick(), channel)
		return
	}

	h.log.Debug().Msgf("INVITE from %s, joining %s", msg.Nick(), channel)

	if err := h.Send("JOIN", channel); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error handling join: %s", channel)
		return
	}

	return
}

// NickServIdentify sends NickServ Identify commands
func (h *Handler) NickServIdentify(password string) error {
	if err := h.Send("PRIVMSG", "NickServ", fmt.Sprintf("IDENTIFY %s", password)); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error identifying with nickserv")
		return err
	}

	return nil
}

// NickChange sets a new nick for our user
func (h *Handler) NickChange(nick string) error {
	h.log.Debug().Msgf("NICK change: %s", nick)

	if client := h.getClient(); client != nil {
		client.SetNick(nick)
	}

	return nil
}

// CurrentNick returns our current nick set by the server
func (h *Handler) CurrentNick() string {
	if client := h.getClient(); client != nil {
		return client.CurrentNick()
	} else {
		return ""
	}
}

// PreferredNick returns our preferred nick from settings
func (h *Handler) PreferredNick() string {
	if client := h.getClient(); client != nil {
		return client.PreferredNick()
	} else {
		return ""
	}
}

// listens for MODE events
func (h *Handler) handleMode(msg ircmsg.Message) {
	h.log.Trace().Msgf("MODE: %+v", msg)

	nick := msg.Params[0]
	//channel := msg.Params[1]
	channel := strings.ToLower(msg.Params[1])

	// if our nick and user mode +r (Identifies the nick as being Registered (settable by services only)) then return
	if h.isOurCurrentNick(nick) && strings.Contains(channel, "+r") {
		h.setAuthenticated()

		return
	}

	if h.network.BotMode && h.botModeChar != "" && h.isOurCurrentNick(nick) && strings.Contains(channel, "+"+h.botModeChar) {
		h.authenticate()
	}
}

// listens for ERR_UMODEUNKNOWNFLAG events
func (h *Handler) handleModeUnknownFlag(msg ircmsg.Message) {
	// if Bot Mode setting failed, still try to authenticate
	h.authenticate()
}

func (h *Handler) SendMsg(channel, msg string) error {
	h.log.Debug().Msgf("sending msg command: %s", msg)

	if err := h.Send("PRIVMSG", channel, msg); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error sending msg: %s", msg)
		return err
	}

	return nil
}

// cleanMessage irc line can contain lots of extra stuff like color so lets clean that
func cleanMessage(message string) string {
	return ircfmt.Strip(message)
}

func (h *Handler) addConnectError(message string) {
	h.m.Lock()
	defer h.m.Unlock()

	h.connectionErrors = append(h.connectionErrors, message)
}

func (h *Handler) ReportStatus(netw *domain.IrcNetworkWithHealth) {
	h.m.RLock()
	defer h.m.RUnlock()

	// only set connected and connected since if we have an active handler and connection
	if !h.network.Enabled {
		return
	}
	if h.client == nil {
		return
	}
	netw.Connected = h.connectedSince != time.Time{}
	netw.ConnectedSince = h.connectedSince
	netw.CurrentNick = h.client.CurrentNick()
	netw.PreferredNick = h.client.PreferredNick()

	if !netw.Connected {
		return
	}

	channelsHealthy := true

	h.channels.ForEach(func(s string, channel *Channel) bool {
		channelsHealthy = channel.Monitoring

		if !channelsHealthy {
			return false
		}

		return true
	})

	//h.bots.ForEach(func(botName string, user *User) bool {
	//	netw.Bots = append(netw.Bots, domain.IrcUser{Nick: user.Nick})
	//	return true
	//})

	netw.Healthy = channelsHealthy

	netw.ConnectionErrors = slices.Clone(h.connectionErrors)
}

// DetermineNetworkRestartRequired diff currentState and desiredState to determine if restart is required to reach desired state
func DetermineNetworkRestartRequired(currentState, desiredState domain.IrcNetwork) ([]string, bool) {
	restartNeeded := false
	var fieldsChanged []string

	if currentState.Server != desiredState.Server {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "server")
	}
	if currentState.Port != desiredState.Port {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "port")
	}
	if currentState.TLS != desiredState.TLS {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "tls")
	}
	if currentState.Pass != desiredState.Pass {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "pass")
	}
	if currentState.InviteCommand != desiredState.InviteCommand {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "invite command")
	}
	if currentState.UseBouncer != desiredState.UseBouncer {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "use bouncer")
	}
	if currentState.BouncerAddr != desiredState.BouncerAddr {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "bouncer addr")
	}
	if currentState.BotMode != desiredState.BotMode {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "bot mode")
	}
	if currentState.UseProxy != desiredState.UseProxy {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "use proxy")
	}
	if currentState.ProxyId != desiredState.ProxyId {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "proxy id")
	}
	if currentState.Auth.Mechanism != desiredState.Auth.Mechanism {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "auth mechanism")
	}
	if currentState.Auth.Account != desiredState.Auth.Account {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "auth account")
	}
	if currentState.Auth.Password != desiredState.Auth.Password {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "auth password")
	}

	return fieldsChanged, restartNeeded
}
