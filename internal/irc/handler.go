// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alphadose/haxmap"
	"github.com/avast/retry-go"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircfmt"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"golang.org/x/net/proxy"
)

var (
	connectionInProgress = errors.New("A connection attempt is already in progress")

	clientDisconnected = errors.New("Message cannot be sent because client is disconnected")

	clientManuallyDisconnected = retry.Unrecoverable(errors.New("IRC client was manually disconnected"))
)

const (
	EventStreamKey = "irc"
)

//type State string
//
//const (
//	StateStopped State = "stopped"
//	//StateConnecting State = "connecting"
//	StateRunning State = "running"
//)

type ircState uint

const (
	ircStopped    ircState = iota // (Handler.client) is nil
	ircConnecting                 // still nil
	ircLive                       // (Handler.client) is non-nil and valid
)

type Handler struct {
	m sync.RWMutex

	log                 zerolog.Logger
	sse                 sseServer
	network             *domain.IrcNetwork
	releaseSvc          releaseService
	notificationService notificationSender
	announceProcessors  map[string]announce.Processor
	definitions         map[string]*domain.IndexerDefinition

	client           *ircevent.Connection
	clientState      ircState
	connectedSince   time.Time
	haveDisconnected bool
	authenticated    bool
	saslauthed       bool

	channels *haxmap.Map[string, *Channel]

	connectionErrors       []string
	failedNickServAttempts int

	capabilities map[string]struct{}

	botModeChar string

	stateMachine *ConnectionStateMachine
}

func NewHandler(log zerolog.Logger, sse sseServer, network domain.IrcNetwork, definitions []*domain.IndexerDefinition, releaseSvc releaseService, notificationSvc notificationSender) *Handler {
	h := &Handler{
		log:                 log.With().Str("network", network.Server).Logger(),
		sse:                 sse,
		client:              nil,
		clientState:         ircStopped,
		network:             &network,
		releaseSvc:          releaseSvc,
		notificationService: notificationSvc,
		definitions:         map[string]*domain.IndexerDefinition{},
		authenticated:       false,
		saslauthed:          false,
		connectionErrors:    []string{},
		channels:            haxmap.New[string, *Channel](),
	}

	// init state machine
	h.stateMachine = NewConnectionStateMachine(h)

	// init indexer, announceProcessor
	h.InitIndexers(definitions)

	return h
}

func (h *Handler) InitIndexers(definitions []*domain.IndexerDefinition) {
	network := h.GetNetwork()

	connectCommands := make([]string, 0)
	if network.InviteCommand != "" {
		cmds := strings.Split(strings.ReplaceAll(network.InviteCommand, "/msg", ""), ",")
		for _, cmd := range cmds {
			cmd = strings.TrimSpace(cmd)

			connectCommands = append(connectCommands, cmd)
		}
	}

	// Indexer definitions are matched to a network by server, not by network ID
	// (see indexerService.GetIndexersByIRCNetwork). Several separate network
	// instances can therefore share one server - e.g. the same tracker with
	// different nicks/usernames - and each instance receives the definitions of
	// *every* indexer on that server. We must only create/join the announce
	// channels that are actually configured on THIS instance, otherwise one
	// instance joins every sibling instance's channels. The channels stored on
	// the network (network.Channels) are that authoritative per-instance set.
	configuredChannels := make(map[string]struct{}, len(network.Channels))
	for _, channel := range network.Channels {
		configuredChannels[strings.ToLower(channel.Name)] = struct{}{}
	}

	// Networks can be shared by multiple indexers but channels are unique
	// so let's add a new AnnounceProcessor per channel
	for _, definition := range definitions {
		if _, ok := h.definitions[definition.Identifier]; ok {
			continue
		}

		h.definitions[definition.Identifier] = definition

		// handle invite command
		inviteCommand := ""
		defaultInvCmd := ""
		for _, setting := range definition.IRC.Settings {
			if setting.Name == "invite_command" {
				defaultInvCmd = strings.ToLower(setting.Default)
				break
			}
		}

		if defaultInvCmd != "" {
			for _, cmd := range connectCommands {
				cmd = strings.TrimSpace(strings.ReplaceAll(cmd, "/msg", ""))
				parts := strings.Split(cmd, " ")
				if len(parts) < 2 {
					continue
				}
				if strings.HasPrefix(defaultInvCmd, strings.ToLower(parts[0])) {
					inviteCommand = cmd
					break
				}
			}
		}

		// indexers can use multiple channels, but it's not common, but let's handle that anyway.
		for _, channel := range definition.IRC.Channels {
			// some channels are defined in mixed case
			channelName := strings.ToLower(channel.Name)

			// skip announce channels not configured on this network instance -
			// they belong to another instance sharing the same server.
			if _, ok := configuredChannels[channelName]; !ok {
				h.log.Trace().Msgf("skipping announce channel %s: not configured on this network instance", channelName)
				continue
			}

			ircChannel := NewChannel(h.log, network.ID, channelName, true, announce.NewAnnounceProcessor(h.log.With().Str("channel", channelName).Logger(), h.releaseSvc, definition))
			ircChannel.SetStateMachine(NewChannelStateMachine(ircChannel, h, inviteCommand))
			ircChannel.SetInviteCommand(inviteCommand)

			ircChannel.RegisterAnnouncers(channel.Announcers)

			h.channels.Set(channelName, ircChannel)
		}

		// look for user-defined channels and add
		for _, channel := range network.Channels {
			channelName := strings.ToLower(channel.Name)

			if ch, found := h.channels.Get(channelName); found {
				ch.Configure(channel.ID, channel.Enabled, channel.Password)

				if ch.StateMachine() == nil {
					ch.SetStateMachine(NewChannelStateMachine(ch, h, inviteCommand))
				}

				h.channels.Swap(channelName, ch)

				continue
			}

			ircChannel := NewChannel(h.log, network.ID, channelName, false, nil)
			ircChannel.Configure(channel.ID, channel.Enabled, channel.Password)
			ircChannel.SetStateMachine(NewChannelStateMachine(ircChannel, h, ""))

			h.channels.Set(channelName, ircChannel)
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

	// snapshot the network once so a concurrent SetNetwork/UpdateNetwork cannot
	// tear the config we build the connection from
	network := h.GetNetwork()

	addr := fmt.Sprintf("%s:%d", network.Server, network.Port)

	if network.UseBouncer && network.BouncerAddr != "" {
		addr = network.BouncerAddr
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

	h.stateMachine.OnConnecting()

	// either we will successfully transition to `StateRunning`, or else
	// we need to reset the state to `ircStopped`
	defer func() {
		h.m.Lock()
		if h.clientState == ircConnecting {
			h.clientState = ircStopped
		}
		h.m.Unlock()
	}()

	client := &ircevent.Connection{
		Nick:          network.Nick,
		User:          network.Auth.Account,
		RealName:      network.Auth.Account,
		Password:      network.Pass,
		Server:        addr,
		KeepAlive:     4 * time.Minute,
		Timeout:       2 * time.Minute,
		ReconnectFreq: 15 * time.Second,
		Version:       "autobrr",
		QuitMessage:   "bye from autobrr",
		Debug:         true,
		Log:           subLogger,
	}

	if network.UseProxy && network.Proxy != nil {
		if !network.Proxy.Enabled {
			h.log.Debug().Msgf("proxy disabled, skip")
		} else {
			if network.Proxy.Addr == "" {
				return errors.New("proxy addr missing")
			}

			proxyUrl, err := url.Parse(network.Proxy.Addr)
			if err != nil {
				return errors.Wrap(err, "could not parse proxy url: %s", network.Proxy.Addr)
			}

			// set user and pass if not empty
			if network.Proxy.User != "" && network.Proxy.Pass != "" {
				proxyUrl.User = url.UserPassword(network.Proxy.User, network.Proxy.Pass)
			}

			var proxyDialer proxy.Dialer

			switch proxyUrl.Scheme {
			case "http", "https":
				h.log.Debug().Msgf("Using HTTP CONNECT proxy: %s for IRC server %s:%d", proxyUrl.Host, network.Server, network.Port)
				proxyDialer = newHTTPProxyDialer(proxyUrl, proxy.Direct, network.TLSSkipVerify)

			default:
				h.log.Debug().Msgf("Using %s proxy: %s", proxyUrl.Scheme, proxyUrl.Host)
				proxyDialer, err = proxy.FromURL(proxyUrl, proxy.Direct)
				if err != nil {
					return errors.Wrap(err, "could not create proxy dialer from url: %s", network.Proxy.Addr)
				}
			}

			proxyContextDialer, ok := proxyDialer.(proxy.ContextDialer)
			if !ok {
				return errors.New("proxy dialer does not expose DialContext(): %v", proxyDialer)
			}

			client.DialContext = proxyContextDialer.DialContext
		}
	}

	if network.Auth.Mechanism == domain.IRCAuthMechanismSASLPlain {
		if network.Auth.Account != "" && network.Auth.Password != "" {
			client.SASLLogin = network.Auth.Account
			client.SASLPassword = network.Auth.Password
			client.SASLOptional = true
			client.UseSASL = true
		}
	}

	if network.TLS {
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
			InsecureSkipVerify: network.TLSSkipVerify,
			MinVersion:         tls.VersionTLS10,
			CipherSuites:       unsafeCipherSuites,
		}
	}

	client.AddConnectCallback(h.onConnect)
	client.AddDisconnectCallback(h.onDisconnect)

	client.AddCallback("MODE", h.handleMode)
	if network.BotMode {
		client.AddCallback(ircevent.ERR_UMODEUNKNOWNFLAG, h.handleModeUnknownFlag)
	}
	client.AddCallback("INVITE", h.handleInvite)
	client.AddCallback("PART", h.handlePart)
	client.AddCallback("PRIVMSG", h.onPrivMessage)
	client.AddCallback("NOTICE", h.onNotice)
	client.AddCallback("NICK", h.onNick)
	client.AddCallback("KICK", h.onKick)
	client.AddCallback("JOIN", h.handleJoin)

	client.AddCallback("TOPIC", h.handleTopicChange)
	client.AddCallback(ircevent.RPL_TOPIC, h.handleTopic)
	client.AddCallback(ircevent.RPL_ENDOFNAMES, h.handleJoined) // end of names

	client.AddCallback(ircevent.RPL_LOGGEDIN, h.handleLoggedIn)
	client.AddCallback(ircevent.RPL_SASLSUCCESS, h.handleSASLSuccess)
	client.AddCallback(ircevent.ERR_SASLFAIL, h.handleSASLFail)

	// surface failed JOIN attempts on the affected channel
	client.AddCallback(ircevent.ERR_CHANNELISFULL, h.handleJoinError)  // 471
	client.AddCallback(ircevent.ERR_INVITEONLYCHAN, h.handleJoinError) // 473
	client.AddCallback(ircevent.ERR_BANNEDFROMCHAN, h.handleJoinError) // 474
	client.AddCallback(ircevent.ERR_BADCHANNELKEY, h.handleJoinError)  // 475
	client.AddCallback(ircevent.ERR_NEEDREGGEDNICK, h.handleJoinError) // 477
	client.AddCallback(ircevent.ERR_NOSUCHNICK, h.handleErrNoSuchNick)

	//h.setConnectionStatus()
	h.m.Lock()
	h.saslauthed = false
	h.client = client
	h.m.Unlock()

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
					h.log.Debug().Msgf("%s connect attempt %d", network.Name, n)
				}
			}),
			retry.Delay(time.Second*15),
			retry.Attempts(25),
			retry.MaxJitter(time.Second*10),
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
	// soju just reports JOIN (366) messages with the wildcard.
	// CurrentNick() and usesBouncer() each take h.m independently (never nested)
	// so this stays safe to call from the IRC callbacks.
	return h.CurrentNick() == nick || (h.usesBouncer() && nick == "*")
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
	for key, channel := range h.channels.Iterator() {
		channel.ResetMonitoring()

		// reset the channel state machine so it can rejoin on reconnect.
		if sm := channel.StateMachine(); sm != nil {
			sm.Reset()
		}

		h.channels.Set(key, channel)
	}
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
	h.setConnectionStatus()

	networkName := h.GetNetwork().Name

	h.m.Lock()
	reconnected := h.haveDisconnected && h.clientState == ircLive
	if reconnected {
		// reset haveDisconnected
		h.haveDisconnected = false
	}
	h.m.Unlock()

	// notify outside the lock so we never hold h.m across the notification I/O
	if reconnected {
		h.log.Info().Msgf("network re-connected after unexpected disconnect: %s", networkName)

		h.notificationService.Send(domain.NotificationEventIRCReconnected, domain.NotificationPayload{
			Subject: "IRC Reconnected",
			Message: fmt.Sprintf("Network: %s", networkName),
		})
	}

	h.log.Info().Msgf("network connected to: %s", networkName)

	time.Sleep(1 * time.Second)

	// Notify state machine of connection - it will handle auth and channel joining
	h.stateMachine.OnConnected()
}

// onDisconnect is the disconnect callback
func (h *Handler) onDisconnect(_ ircmsg.Message) {
	h.log.Debug().Msgf("DISCONNECT")

	h.m.Lock()

	// reset connectedSince
	h.connectedSince = time.Time{}

	// reset authenticated
	h.authenticated = false

	h.haveDisconnected = true

	manuallyDisconnected := h.clientState == ircStopped
	networkName := h.network.Name

	h.m.Unlock()

	// reset channels monitored status and channel state machines so they
	// rejoin cleanly on reconnect instead of getting stuck in Monitoring
	h.resetChannelState()

	// check if we are responsible for disconnect
	if !manuallyDisconnected {
		// only send notification if we did not initiate disconnect/restart/stop
		h.notificationService.Send(domain.NotificationEventIRCDisconnected, domain.NotificationPayload{
			Subject: "IRC Disconnected unexpectedly",
			Message: fmt.Sprintf("Network: %s", networkName),
		})
	}

	h.stateMachine.OnDisconnected()
}

// onNotice handles NOTICE events
func (h *Handler) onNotice(msg ircmsg.Message) {
	switch msg.Nick() {
	case "NickServ":
		h.handleNickServ(msg)
	default:
		// a NOTICE from an invite bot while a channel is still awaiting its
		// invite is a rejection, not the invite itself (that arrives as INVITE)
		h.handleInviteResponse(msg)
	}
}

// handleNickServ is called from NOTICE events
func (h *Handler) handleNickServ(msg ircmsg.Message) {
	h.log.Trace().Msgf("NOTICE from nickserv: %v", msg.Params)

	// You're now logged in as test-bot
	// Password accepted - you are now recognized.
	if contains(msg.Params[1], "you're now logged in as", "password accepted", "you are now recognized", "you are now identified", "you are already logged in") {
		h.log.Debug().Msgf("NOTICE nickserv logged in: %v", msg.Params)
		h.setAuthenticated()
		return
	}

	if contains(msg.Params[1],
		"Invalid account credentials",
		"Authentication failed: Invalid account credentials",
		"password incorrect",
	) {
		h.addConnectError("authentication failed: Bad account credentials")
		h.log.Error().Msg("NickServ: authentication failed - bad account credentials")

		h.stateMachine.OnError("nickserv authentication failed: bad credentials")

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

			h.stateMachine.OnError("nickserv authentication failed: account does not exist")

			// stop network and notify user
			h.Stop()
			return
		}
	}

	if contains(msg.Params[1],
		"This nickname is registered and protected",
		"please choose a different nick",
		"choose a different nick",
	) {
		h.authenticate()

		h.m.Lock()
		h.failedNickServAttempts++
		attempts := h.failedNickServAttempts
		h.m.Unlock()

		if attempts >= 3 {
			h.log.Warn().Msgf("NickServ %d failed login attempts", attempts)
			h.addConnectError("authentication failed: nick in use and not authenticated")

			h.stateMachine.OnError("nickserv authentication failed: nick in use")

			// stop network and notify user
			h.Stop()
		}
	}

	// fallback for networks that require both password and nick to NickServ IDENTIFY
	// Invalid parameters. For usage, do /msg NickServ HELP IDENTIFY
	if contains(msg.Params[1], "invalid parameters", "help identify") {
		h.log.Debug().Msgf("NOTICE nickserv invalid: %v", msg.Params)

		net := h.GetNetwork()
		h.Send("PRIVMSG", "NickServ", fmt.Sprintf("IDENTIFY %s %s", net.Auth.Account, net.Auth.Password))
	}
}

func (h *Handler) getClient() *ircevent.Connection {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.client
}

// usesBouncer reports whether the network is configured to use a bouncer.
func (h *Handler) usesBouncer() bool {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.network.UseBouncer
}

// botModeConfig returns whether bot mode is enabled and the negotiated bot-mode char.
func (h *Handler) botModeConfig() (bool, string) {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.network.BotMode, h.botModeChar
}

func (h *Handler) setBotModeChar(char string) {
	h.m.Lock()
	defer h.m.Unlock()
	h.botModeChar = char
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
	client := h.getClient()
	if client == nil {
		return false
	}

	char := client.ISupport()["BOT"]
	h.setBotModeChar(char)

	return char != ""
}

// setBotMode attempts to set Bot Mode on ourselves
// See https://ircv3.net/specs/extensions/bot-mode
func (h *Handler) setBotMode() {
	client := h.getClient()
	if client == nil {
		return
	}

	_, char := h.botModeConfig()
	client.Send("MODE", h.CurrentNick(), "+"+char)
}

// authenticate sends NickServIdentify if not authenticated
func (h *Handler) authenticate() {
	h.m.RLock()
	password := h.network.Auth.Password
	shouldSendNickserv := !h.authenticated && !h.saslauthed && password != ""
	h.m.RUnlock()

	if shouldSendNickserv {
		h.log.Trace().Msg("on connect not authenticated and password not empty: send nickserv identify")
		h.NickServIdentify(password)
	} else {
		h.setAuthenticated()
	}
}

func (h *Handler) handleLoggedIn(m ircmsg.Message) {
	h.log.Trace().Str("event", "900").Msg("logged in")
	nick := m.Nick()
	if h.isOurCurrentNick(nick) {
		h.setAuthenticated()
	}
}

// handleSASLSuccess we get here early so set saslauthed before we hit onConnect
func (h *Handler) handleSASLSuccess(_ ircmsg.Message) {
	h.m.Lock()
	h.saslauthed = true
	h.m.Unlock()
}

func (h *Handler) handleSASLFail(_ ircmsg.Message) {
	h.addConnectError("authentication failed: SASL negotiation failed")
	h.stateMachine.OnError("sasl authentication failed")

	h.m.Lock()
	h.saslauthed = false
	h.m.Unlock()

	h.Stop()
}

// setAuthenticated sets the states for authenticated, connectionErrors, failedNickServAttempts
// and then notifies the state machine which handles invite commands and joining channels
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

	// Notify state machine - it will handle joining channels and invite commands
	h.stateMachine.OnAuthenticated()
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
	// NICK <newnick>
	if len(msg.Params) < 1 {
		return
	}

	nick := msg.Nick()
	h.log.Trace().Str("event", "NICK").Str("old_nick", nick).Str("new_nick", msg.Params[0]).Msg("user changed nick")

	if !h.isOurCurrentNick(nick) {
		return
	}

	h.authenticate()
}

func (h *Handler) onKick(msg ircmsg.Message) {
	// KICK <channel> <nick> [<reason>]
	if len(msg.Params) < 2 {
		return
	}

	nick := msg.Nick()
	channelName := strings.ToLower(msg.Params[0])
	affectedNick := msg.Params[1]
	reason := strings.Join(msg.Params[2:], " ")
	h.log.Trace().Str("event", "KICK").Str("nick", affectedNick).Str("kicked_by", nick).Str("channel", channelName).Str("reason", reason).Msg("kicked from channel")

	if !h.isOurCurrentNick(affectedNick) {
		return
	}

	ircChannel, found := h.channels.Get(channelName)
	if !found {
		return
	}

	if sm := ircChannel.StateMachine(); sm != nil {
		sm.OnKicked(affectedNick, nick, reason)
	}
	// TODO set again or swap?
	//h.channels.Swap(channelName, ircChannel)
}

func (h *Handler) broadcastEvent(event string, data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		h.log.Error().Stack().Err(err).Msg("error marshalling data")
		return
	}

	h.sse.Publish(EventStreamKey, &sse.Event{
		Event: []byte(event),
		Data:  bytes,
	})
}

func (h *Handler) broadcastMessage(msg domain.IrcMessage) {
	h.sse.Publish(EventStreamKey, &sse.Event{
		Event: []byte("PRIVMSG"),
		Data:  msg.Bytes(),
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
		// this is a DM - possibly an invite bot answering our invite command
		h.log.Debug().Str("direct-message", channel).Str("from-nick", nick).Str("msg", cleanedMsg).Msg("got direct-message")

		// a DM from an invite bot while a channel is still awaiting its invite is
		// a rejection, not the invite itself (that arrives as INVITE)
		h.handleInviteResponse(msg)

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
	h.broadcastMessage(domain.IrcMessage{Network: h.GetNetwork().ID, Channel: channel, Nick: nick, Message: cleanedMsg, Time: time.Now()})

	//h.log.Debug().Str("channel", channel).Str("nick", nick).Msg(cleanedMsg)

	return
}

// JoinChannels sends multiple join commands
func (h *Handler) JoinChannels() {
	h.log.Debug().Msg("Joining channels")
	for _, channel := range h.channels.Iterator() {
		if !channel.IsEnabled() {
			continue
		}

		if sm := channel.StateMachine(); sm != nil {
			sm.Start()
		} else {
			if err := h.JoinChannel(channel.Name, channel.GetPassword()); err != nil {
				h.log.Error().Stack().Err(err).Msgf("error joining channel %s", channel.Name)
			}
		}
	}
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

// AddChannel registers a channel on an already-running handler and starts its
// join workflow. The Channel (and its state machine) MUST exist in h.channels
// before the JOIN is sent: otherwise the server's JOIN echo reaches handleJoin,
// finds no matching channel, and immediately parts it as "unwanted" - so a
// channel added to a live network never gets monitored until a full restart.
// Safe to call for a channel that already exists (refreshes config, starts it
// only if it is enabled and not already monitoring).
func (h *Handler) AddChannel(channel domain.IrcChannel) {
	channelName := strings.ToLower(channel.Name)

	ircChannel, found := h.channels.Get(channelName)
	if !found {
		// a user-defined extra channel has no indexer announce processor
		ircChannel = NewChannel(h.log, h.GetNetwork().ID, channelName, false, nil)
		ircChannel.SetStateMachine(NewChannelStateMachine(ircChannel, h, ""))
		h.channels.Set(channelName, ircChannel)
	}

	ircChannel.Configure(channel.ID, channel.Enabled, channel.Password)

	if !ircChannel.IsEnabled() {
		h.log.Debug().Msgf("channel %s added but disabled, not joining", channelName)
		return
	}

	if ircChannel.IsMonitoring() {
		// already joined and monitored, nothing to do
		return
	}

	h.log.Debug().Msgf("adding and joining channel %s", channelName)

	if sm := ircChannel.StateMachine(); sm != nil {
		// Reset() first (mirrors UpdateChannel's re-join branch): Start() runs its
		// transition from Idle, so a channel parked in a sticky state - InviteFailed
		// or Parted, whose transition tables do not permit AwaitingInvite - is still
		// re-driven through the join workflow instead of having the transition
		// silently dropped as invalid.
		sm.Reset()
		sm.Start()
		return
	}

	if err := h.JoinChannel(ircChannel.Name, ircChannel.GetPassword()); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error joining channel %s", channelName)
	}
}

// RemoveChannel parts a channel on a running handler and removes it from
// tracking so it no longer counts toward network health, and voids any pending
// retry timers on its state machine.
func (h *Handler) RemoveChannel(name string) {
	channelName := strings.ToLower(name)

	h.log.Debug().Msgf("removing channel %s", channelName)

	if err := h.PartChannel(channelName); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error parting channel %s", channelName)
	}

	if ch, found := h.channels.Get(channelName); found {
		if sm := ch.StateMachine(); sm != nil {
			sm.Reset() // stop monitoring and void any pending timers
		}
	}

	h.channels.Del(channelName)
}

// UpdateChannel re-applies the persisted config (enabled, password) to an
// already-tracked channel and reconciles it on the fly - no network restart.
//   - disabled: part the channel
//   - enabled but not currently monitored (newly enabled, or a prior join failed
//     e.g. on a wrong key): (re)join with the updated config
//   - enabled and already monitored: the new config is stored for the next
//     (re)connect; we do not disrupt an active channel by re-joining
func (h *Handler) UpdateChannel(channel domain.IrcChannel) {
	channelName := strings.ToLower(channel.Name)

	ircChannel, found := h.channels.Get(channelName)
	if !found {
		// not tracked yet - treat as an add
		h.AddChannel(channel)
		return
	}

	wasEnabled := ircChannel.IsEnabled()
	oldPassword := ircChannel.GetPassword()
	monitoring := ircChannel.IsMonitoring()

	ircChannel.Configure(channel.ID, channel.Enabled, channel.Password)

	enabledChanged := channel.Enabled != wasEnabled
	passwordChanged := channel.Password != oldPassword

	switch {
	case !enabledChanged && !passwordChanged:
		// nothing actionable changed

	case !channel.Enabled:
		h.log.Debug().Msgf("channel %s disabled, parting", channelName)
		if err := h.PartChannel(channelName); err != nil {
			h.log.Error().Stack().Err(err).Msgf("error parting channel %s", channelName)
		}
		ircChannel.ResetMonitoring()
		if sm := ircChannel.StateMachine(); sm != nil {
			sm.Reset()
		}

	case !monitoring:
		h.log.Debug().Msgf("channel %s config changed, (re)joining", channelName)
		if sm := ircChannel.StateMachine(); sm != nil {
			sm.Reset()
			sm.Start()
		}

	default:
		h.log.Debug().Msgf("channel %s config updated while monitoring; applied for next reconnect", channelName)
	}
}

// handleJoin listens for JOIN events. autobrr only cares about its own joins:
// if we joined a channel we don't track, part it; other users' joins are ignored
// (we don't track the channel user list).
func (h *Handler) handleJoin(msg ircmsg.Message) {
	channel := strings.ToLower(msg.Params[0])
	nick := msg.Nick()

	if !h.isOurCurrentNick(nick) {
		return
	}

	if _, found := h.channels.Get(channel); !found {
		h.log.Debug().Msgf("Joined unwanted channel %s, lets part it..", channel)

		if err := h.PartChannel(channel); err != nil {
			h.log.Error().Stack().Err(err).Msgf("error parting channel %s", channel)
		}

		return
	}

	h.log.Info().Msgf("Join channel %s", channel)
}

func (h *Handler) setChannelError(channelName, errMsg string) {
	channelName = strings.ToLower(channelName)

	if ch, found := h.channels.Get(channelName); found {
		ch.SetConnectionError(errMsg)
		if sm := ch.StateMachine(); sm != nil {
			sm.OnError(errMsg)
		}
		h.channels.Swap(channelName, ch)
	}
}

func (h *Handler) markPendingChannelErrors(errMsg string) {
	for name, channel := range h.channels.Iterator() {
		if !channel.IsEnabled() {
			continue
		}

		if channel.IsMonitoring() {
			continue
		}

		channel.SetConnectionError(errMsg)
		h.channels.Swap(name, channel)
	}
}

// handlePart listens for PART events. Only our own parts matter - other users'
// parts are ignored.
func (h *Handler) handlePart(msg ircmsg.Message) {
	channel := strings.ToLower(msg.Params[0])
	nick := msg.Nick()

	if !h.isOurCurrentNick(nick) {
		return
	}

	h.log.Debug().Msgf("PART channel %s", channel)

	ircChannel, found := h.channels.Get(channel)
	if !found {
		return
	}

	// clear the monitoring flag before OnParted so the Parted broadcast (and the
	// network health it carries) reflect that we left the channel
	ircChannel.ResetMonitoring()

	if sm := ircChannel.StateMachine(); sm != nil {
		sm.OnParted()
	}

	h.channels.Swap(channel, ircChannel)

	h.log.Debug().Msgf("Left channel %s", channel)
}

// PartChannel parts/leaves channel
func (h *Handler) PartChannel(channel string) error {
	// if using bouncer we do not want to part any channels
	if h.usesBouncer() {
		h.log.Debug().Msgf("using bouncer, skip part channel %s", channel)
		return nil
	}

	h.log.Debug().Msgf("Leaving channel %s", channel)

	return h.Send("PART", channel)

	// TODO remove announceProcessor
}

// handleTopic listens for 332 ircevent.
func (h *Handler) handleTopic(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}
	channel := strings.ToLower(msg.Params[1])
	topic := msg.Params[2]

	h.log.Trace().Str("topic", topic).Str("channel", channel).Msg("TOPIC")

	// set topic for channel
	ircChannel, found := h.channels.Get(channel)
	if found {
		h.log.Trace().Str("topic", topic).Str("channel", ircChannel.Name).Msg("set new channel topic")

		ircChannel.SetTopic(topic)

		h.channels.Swap(channel, ircChannel)

		return
	}
}

// handleTopicChange listens for TOPIC events
func (h *Handler) handleTopicChange(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}
	channel := strings.ToLower(msg.Params[0])
	topic := msg.Params[1]

	h.log.Trace().Str("topic", topic).Str("channel", channel).Msg("TOPIC")

	// set topic for channel
	ircChannel, found := h.channels.Get(channel)
	if found {
		h.log.Trace().Str("topic", topic).Str("channel", ircChannel.Name).Msg("set new channel topic")

		ircChannel.SetTopic(topic)

		h.channels.Swap(channel, ircChannel)

		return
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
		if sm := ircChannel.StateMachine(); sm != nil {
			sm.OnJoinSuccess()
		}

		//ircChannel.SetMonitoring()

		h.channels.Swap(channel, ircChannel)

		h.log.Trace().Msgf("set monitoring: %s", ircChannel.Name)

		if ircChannel.DefaultChannel {
			h.log.Info().Msgf("Monitoring channel %s", channel)
		} else {
			h.log.Info().Msgf("Joined extra channel %s", channel)
		}

		// Notify state machine that we've joined a channel
		h.stateMachine.OnChannelJoined(channel)

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

		if strings.HasPrefix(cmd, "/sleep") {
			parts := strings.SplitN(cmd, " ", 2)
			if len(parts) < 2 {
				h.log.Warn().Msgf("sleep command missing duration: %s", cmd)
				continue
			}
			secs, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				h.log.Error().Err(err).Msgf("error parsing sleep command: %s", cmd)
				continue
			}
			h.log.Debug().Msgf("sleeping for %d seconds: %s", secs, cmd)
			time.Sleep(time.Duration(secs) * time.Second)
			continue
		}

		h.log.Debug().Msgf("sending connect command: %s", cmd)

		params := strings.SplitN(cmd, " ", 2)

		if err := h.Send("PRIVMSG", params...); err != nil {
			h.log.Error().Err(err).Msgf("error handling connect command: %s", cmd)
			return err
		}

		// TODO RETRY if error or not successful

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
	nick := msg.Nick()

	h.log.Trace().Msgf("INVITE from %s to join: %s", nick, channel)
	h.log.Debug().Msgf("INVITE from %s, joining %s", nick, channel)

	ircChannel, found := h.channels.Get(channel)
	if !found {
		h.log.Trace().Msgf("invite from %s to join: %s - unwanted channel, skip joining", nick, channel)
		return
	}

	if sm := ircChannel.StateMachine(); sm != nil {
		sm.OnInvite(nick)
		return
	}

	if err := h.Send("JOIN", channel); err != nil {
		h.log.Error().Stack().Err(err).Msgf("error handling join: %s", channel)
	}
}

// joinErrorReason maps a JOIN-error numeric to a short human-readable cause.
var joinErrorReason = map[string]string{
	ircevent.ERR_CHANNELISFULL:  "channel is full (+l)",
	ircevent.ERR_INVITEONLYCHAN: "channel is invite-only (+i)",
	ircevent.ERR_BANNEDFROMCHAN: "banned from channel (+b)",
	ircevent.ERR_BADCHANNELKEY:  "wrong or missing channel password (+k)",
	ircevent.ERR_NEEDREGGEDNICK: "must be identified with services to join (+r)",
}

// handleJoinError handles the IRC error numerics for a failed JOIN and surfaces
// the failure on the specific channel immediately, instead of leaving it in
// Joining until the join timeout fires. Covers 471 (full), 473 (invite-only),
// 474 (banned), 475 (bad key) and 477 (need registered nick).
func (h *Handler) handleJoinError(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}

	channel := strings.ToLower(msg.Params[1])

	// the server's trailing text (e.g. "Cannot join channel (+k)"), if present
	serverReason := ""
	if len(msg.Params) > 2 {
		serverReason = msg.Params[len(msg.Params)-1]
	}

	cause := joinErrorReason[msg.Command]
	if cause == "" {
		cause = "join rejected"
	}

	errMsg := fmt.Sprintf("could not join %s: %s", channel, cause)
	if serverReason != "" {
		errMsg = fmt.Sprintf("%s (%s)", errMsg, serverReason)
	}

	h.log.Warn().Str("channel", channel).Str("numeric", msg.Command).Str("reason", serverReason).Msg("channel join rejected")

	h.addConnectError(errMsg)
	h.setChannelError(channel, errMsg)
	h.stateMachine.OnChannelError(channel, errMsg)
}

// handleErrNoSuchNick listens for ircevent.ERR_NOSUCHNICK events
func (h *Handler) handleErrNoSuchNick(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}

	// get channel
	nick := strings.ToLower(msg.Params[1])

	h.log.Debug().Str("nick", nick).Msgf("No such nick")

	for _, channel := range h.channels.Iterator() {
		if !channel.IsEnabled() {
			continue
		}

		if inviteBotNick(channel.InviteCommand()) == nick {
			h.log.Debug().Str("nick", nick).Msgf("No such nick, sending invite command")

			// start retry loop of invite command here
			if sm := channel.StateMachine(); sm != nil {
				sm.OnNoSuchNick(nick)
			}
		}
	}
}

// inviteBotNick returns the nick an invite command is sent to: its first
// whitespace-delimited token (sendInviteCommand PRIVMSGs that token), lowercased
// for case-insensitive comparison. Matching the exact token - rather than a
// prefix of the whole command - avoids parking a channel on a message from an
// unrelated bot whose nick merely prefixes the real bot's (e.g. "voy" vs
// "voyager", or "voyager" vs an invite command starting "voyager2").
func inviteBotNick(inviteCommand string) string {
	fields := strings.Fields(inviteCommand)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(fields[0])
}

// isChannelTarget reports whether an IRC message target names a channel rather
// than a user. Channels start with one of the RFC/ISUPPORT CHANTYPES prefixes.
func isChannelTarget(target string) bool {
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

// handleInviteResponse treats a NOTICE or DM from an invite bot as a failed
// invite. A successful invite always arrives as an INVITE event, so a plain
// message from the bot while a channel is still AwaitingInvite means the request
// was rejected (bad IRC key, account not registered on the tracker, etc.). The
// bot's reason is surfaced on every channel awaiting an invite from that bot so
// the user can see why the join is stuck, and the channel parks in InviteFailed
// (a rejection is definitive; only an absent bot keeps retrying via backoff).
//
// The guards make this safe to call from the PRIVMSG and NOTICE hot paths: the
// message must be addressed to us directly, not to a channel (a channel announce
// is never matched - this also stops a bot that is both the announcer and the
// invite bot from tripping a sibling channel), and only channels in
// AwaitingInvite whose invite command targets this exact bot are touched, so
// ordinary bot chatter cannot raise a false error.
func (h *Handler) handleInviteResponse(msg ircmsg.Message) {
	// a direct message from the bot is addressed to our nick; a channel message
	// (announce, topic, etc.) targets the channel and must not be treated as an
	// invite reply
	if len(msg.Params) < 1 || isChannelTarget(msg.Params[0]) {
		return
	}

	nick := strings.ToLower(msg.Nick())
	if nick == "" || nick == "nickserv" {
		return
	}

	reason := ""
	if len(msg.Params) > 0 {
		reason = strings.TrimSpace(msg.Params[len(msg.Params)-1])
	}

	for _, channel := range h.channels.Iterator() {
		if !channel.IsEnabled() {
			continue
		}

		sm := channel.StateMachine()
		if sm == nil || sm.CurrentState() != ChannelStateAwaitingInvite {
			continue
		}

		if botNick := inviteBotNick(channel.InviteCommand()); botNick == "" || botNick != nick {
			continue
		}

		errMsg := fmt.Sprintf("invite rejected by %s", msg.Nick())
		if reason != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, reason)
		}

		h.log.Warn().Str("bot", nick).Str("channel", channel.Name).Str("reason", reason).Msg("invite request rejected by bot")
		sm.OnInviteFailed(errMsg)
	}
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

	// MODE <target> <modestring> [<args>...]
	if len(msg.Params) < 2 {
		return
	}

	target := msg.Params[0]
	modes := msg.Params[1]

	// if our nick is set +r (Identifies the nick as being Registered, settable by
	// services only) then we're authenticated
	if h.isOurCurrentNick(target) && modeAdds(modes, 'r') {
		h.setAuthenticated()

		return
	}

	if botModeEnabled, botModeChar := h.botModeConfig(); botModeEnabled && len(botModeChar) == 1 && h.isOurCurrentNick(target) && modeAdds(modes, botModeChar[0]) {
		h.authenticate()
	}
}

// modeAdds reports whether an IRC mode string adds the given flag: the flag
// appears in a '+' section and is not subsequently removed. This is stricter
// than a substring check, which could be tripped by an unrelated multi-char
// mode string that merely contains "+<flag>".
func modeAdds(modes string, flag byte) bool {
	adding := false
	added := false
	for i := 0; i < len(modes); i++ {
		switch modes[i] {
		case '+':
			adding = true
		case '-':
			adding = false
		case flag:
			added = adding
		}
	}
	return added
}

// listens for ERR_UMODEUNKNOWNFLAG events
func (h *Handler) handleModeUnknownFlag(_ ircmsg.Message) {
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

	netw.ConnectionErrors = slices.Clone(h.connectionErrors)
	netw.Healthy = h.computeHealthy()
}

// computeHealthy reports whether the network is healthy: the connection state
// machine is healthy AND every enabled announce (default) channel is monitoring
// without errors. Non-default channels (user-added extras) are surfaced
// per-channel and deliberately do not gate network health, so one flaky extra
// channel cannot flip the whole network red.
func (h *Handler) computeHealthy() bool {
	channelsHealthy := true

	for _, channel := range h.channels.Iterator() {
		snap := channel.Snapshot()
		if !snap.Enabled || !snap.DefaultChannel {
			continue
		}

		if !snap.Monitoring || len(snap.ConnectionErrors) > 0 {
			channelsHealthy = false
			break
		}
	}

	return h.stateMachine.IsHealthy() && channelsHealthy
}

func (h *Handler) ReportHealth() {
	//h.m.RLock()
	//defer h.m.RUnlock()

	healthData := map[string]any{
		"network":           h.network.ID,
		"healthy":           false,
		"connection_errors": []string{"Connection timeout"},
	}
	h.broadcastEvent("HEALTH", healthData)
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
	if currentState.TLSSkipVerify != desiredState.TLSSkipVerify {
		restartNeeded = true
		fieldsChanged = append(fieldsChanged, "tls skip verify")
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

type SSEMsg map[string]any

func (m SSEMsg) MustBytes() []byte {
	b, _ := json.Marshal(m)
	return b
}

func (m SSEMsg) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}
