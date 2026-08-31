// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	stdErr "errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
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

// irc-go invokes disconnect callbacks only for registered sessions, so the
// breaker does not count dial or registration failures handled by Run.
const (
	flappingSessionMinLifetime = 30 * time.Second
	flappingStopThreshold      = 5
	flappingWindow             = 15 * time.Minute
)

// identifyForm is the argument form of the outstanding NickServ IDENTIFY.
//
// The bare form is the default because its failures are always loud and are
// returned before any password comparison. Services that do not accept an
// account argument (Anope 1.8 and its derivatives, still deployed on Rizon)
// parse only the first token as the password, so an account-qualified IDENTIFY
// there is silently read as a wrong password and burns a bad_password strike.
// See https://github.com/autobrr/autobrr/issues/2528.
type identifyForm int

const (
	identifyFormBare identifyForm = iota
	identifyFormAccount
)

type Handler struct {
	m sync.RWMutex

	log                zerolog.Logger
	eventBus           eventBus
	sse                sseServer
	network            *domain.IrcNetwork
	releaseSvc         releaseService
	announceProcessors map[string]announce.Processor
	definitions        map[string]*domain.IndexerDefinition

	client           *ircevent.Connection
	clientState      ircState
	connectedSince   time.Time
	haveDisconnected bool
	shortSessionEnds []time.Time
	authenticated    bool
	saslauthed       bool

	identifyAttempt     identifyForm
	identifyEscalated   bool
	identifyFormLearned identifyForm
	identifyOutstanding bool

	channels *haxmap.Map[string, *Channel]

	connectionErrors       []string
	failedNickServAttempts int

	capabilities map[string]struct{}

	botModeChar string

	stateMachine *ConnectionStateMachine
}

func NewHandler(log zerolog.Logger, eventBus eventBus, sse sseServer, network domain.IrcNetwork, definitions []*domain.IndexerDefinition, releaseSvc releaseService) *Handler {
	h := &Handler{
		log:              log.With().Str("network", network.Server).Logger(),
		eventBus:         eventBus,
		sse:              sse,
		client:           nil,
		clientState:      ircStopped,
		network:          &network,
		releaseSvc:       releaseSvc,
		definitions:      map[string]*domain.IndexerDefinition{},
		authenticated:    false,
		saslauthed:       false,
		connectionErrors: []string{},
		channels:         haxmap.New[string, *Channel](),
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
				h.log.Trace().Str("channel", channelName).Msg("skipping announce channel: not configured on this network instance")
				continue
			}

			skipCleanMessage := false
			if channel.Parse != nil {
				skipCleanMessage = channel.Parse.SkipCleanMessage
			}

			ircChannel := NewChannel(h.log, network.ID, channelName, true, skipCleanMessage, announce.NewAnnounceProcessor(h.log.With().Str("channel", channelName).Logger(), h.releaseSvc, definition))
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

			ircChannel := NewChannel(h.log, network.ID, channelName, false, false, nil)
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
			h.log.Debug().Msg("proxy disabled, skipping")
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
				h.log.Debug().Str("proxy", proxyUrl.Host).Str("server", network.Server).Int("port", network.Port).Msg("using HTTP CONNECT proxy for IRC server")
				proxyDialer = newHTTPProxyDialer(proxyUrl, proxy.Direct, network.TLSSkipVerify)

			default:
				h.log.Debug().Str("proxy_scheme", proxyUrl.Scheme).Str("proxy", proxyUrl.Host).Msg("using proxy")
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
	client.AddDisconnectCallback(func(msg ircmsg.Message) {
		h.onClientDisconnect(client, msg)
	})

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

	// the server has banned us (K-Line/G-Line); stop and surface the reason
	client.AddCallback(ircevent.ERR_YOUREBANNEDCREEP, h.handleBanned) // 465

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
	h.resetIdentifyForm()
	h.m.Unlock()

	if err := func() error {
		// count connect attempts
		connectAttempts := 0
		disconnectTime := time.Now()

		// retry initial connect if network is down
		// using exponential backoff of 15 seconds
		return retry.Do(
			func() error {
				h.log.Debug().Int("attempt", connectAttempts).Msg("connect attempt")

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

					// A fatal in-band failure (a ban/G-Line, or a NickServ auth
					// failure) is detected by its callback DURING registration and
					// calls Stop() (setting ircStopped) before Connect() returns its
					// error. Such a failure is not transient, so abort the reconnect
					// loop immediately instead of waiting out the 15s backoff before
					// the next attempt notices the stop.
					h.m.RLock()
					stopped := h.clientState == ircStopped
					h.m.RUnlock()
					if stopped {
						return retry.Unrecoverable(err)
					}

					// A TLS certificate verification failure (expired or not yet
					// valid cert, unknown CA, hostname mismatch) is not transient:
					// every retry fails identically until the tracker fixes its
					// certificate or the user enables TLSSkipVerify. Surface the
					// reason and stop the network instead of burning the whole
					// backoff schedule on a doomed loop.
					if certErr, ok := stdErr.AsType[*tls.CertificateVerificationError](err); ok {
						errMsg := fmt.Sprintf("TLS certificate verification failed: %v", certErr.Err)
						h.log.Error().Str("reason", errMsg).Msg("stopping network: TLS certificate verification failed")

						h.addConnectError(errMsg)
						h.stateMachine.OnError(errMsg)
						h.Stop()

						return retry.Unrecoverable(err)
					}

					return err
				}

				if connectAttempts > 0 {
					h.log.Debug().Int("attempt", connectAttempts).Dur("offline_duration", time.Since(disconnectTime)).Msg("connected at attempt")
					return nil
				}

				return nil
			},
			retry.OnRetry(func(n uint, err error) {
				if n > 0 {
					h.log.Debug().Uint("attempt", n).Msg("connect attempt")
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
		h.log.Error().Stack().Msg("two concurrent connection attempts detected")
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
	h.setNetworkLocked(network)
	h.m.Unlock()
}

func (h *Handler) SetNetwork(network *domain.IrcNetwork) {
	h.m.Lock()
	h.setNetworkLocked(network)
	h.m.Unlock()
}

// setNetworkLocked swaps the network config, dropping the learned IDENTIFY form
// if the credentials changed: which form works is a property of the account we
// authenticate as, so it cannot outlive an edit to it.
// The caller must hold h.m.
func (h *Handler) setNetworkLocked(network *domain.IrcNetwork) {
	if h.network == nil || h.network.Auth != network.Auth {
		h.identifyFormLearned = identifyFormBare
		h.identifyOutstanding = false
	}

	h.network = network
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
	h.resetFlappingBreakerLocked()
	h.identifyOutstanding = false
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
		h.log.Info().Msg("network re-connected after unexpected disconnect")

		h.eventBus.EmitIRC(context.Background(), events.IRCEvent{
			Event:   events.Event{Type: events.IRCReconnected},
			Network: networkName,
			State:   string(events.IRCReconnected),
			Message: fmt.Sprintf("Network: %s", networkName),
		})
	}

	h.log.Info().Msg("network connected")

	time.Sleep(1 * time.Second)

	// Notify state machine of connection - it will handle auth and channel joining
	h.stateMachine.OnConnected()
}

func (h *Handler) onDisconnect(msg ircmsg.Message) {
	h.onClientDisconnect(h.getClient(), msg)
}

// onClientDisconnect handles a disconnect from the client that emitted it.
func (h *Handler) onClientDisconnect(client *ircevent.Connection, _ ircmsg.Message) {
	h.log.Debug().Msg("disconnect")

	h.m.Lock()
	if h.client != client {
		if h.clientState == ircStopped && h.stateMachine.GetState() != StateDisconnected {
			h.stateMachine.OnDisconnected()
		}
		h.m.Unlock()
		return
	}

	endedAt := time.Now()
	sessionLifetime := time.Duration(0)
	if !h.connectedSince.IsZero() {
		sessionLifetime = endedAt.Sub(h.connectedSince)
	}

	h.connectedSince = time.Time{}

	// reset authenticated
	h.authenticated = false

	// a reconnect starts the identify ladder over: the nick we get back may
	// differ, and the previous connection's escalation says nothing about this one
	h.resetIdentifyForm()

	h.haveDisconnected = true

	manuallyDisconnected := h.clientState == ircStopped
	networkName := h.network.Name
	stopForFlapping := !manuallyDisconnected && h.recordSessionEndLocked(sessionLifetime, endedAt)
	var flappingError string
	if stopForFlapping {
		flappingError = fmt.Sprintf("connection flapping: %d sessions lasted under %s within %s; network stopped to avoid repeated reconnects", flappingStopThreshold, flappingSessionMinLifetime, flappingWindow)
		if !slices.Contains(h.connectionErrors, flappingError) {
			h.connectionErrors = append(h.connectionErrors, flappingError)
		}
		h.haveDisconnected = false
		h.clientState = ircStopped
		h.client = nil
		h.resetFlappingBreakerLocked()
		h.resetChannelState()
		h.stateMachine.OnError(flappingError)
	}

	h.m.Unlock()

	// reset channels monitored status and channel state machines so they
	// rejoin cleanly on reconnect instead of getting stuck in Monitoring
	if !stopForFlapping {
		h.resetChannelState()
	}

	if stopForFlapping {
		h.log.Error().
			Int("sessions", flappingStopThreshold).
			Dur("min_lifetime", flappingSessionMinLifetime).
			Dur("window", flappingWindow).
			Msg("connection flapping; stopping network")

		h.eventBus.EmitIRC(context.Background(), events.IRCEvent{
			Event:   events.Event{Type: events.IRCFlapping},
			Network: networkName,
			State:   string(events.IRCFlapping),
			Message: fmt.Sprintf("Network: %s stopped after repeated short-lived connections; restart it after resolving the connection issue", networkName),
		})

		if client != nil {
			client.Quit()
		}

		return
	}

	// check if we are responsible for disconnect
	if !manuallyDisconnected {
		// only send notification if we did not initiate disconnect/restart/stop
		h.eventBus.EmitIRC(context.Background(), events.IRCEvent{
			Event:   events.Event{Type: events.IRCDisconnected},
			Network: networkName,
			State:   string(events.IRCDisconnected),
			Message: fmt.Sprintf("Network: %s", networkName),
			//Subject: "IRC Disconnected unexpectedly",
		})
	}

	h.stateMachine.OnDisconnected()
}

// recordSessionEndLocked records one registered session and reports whether the
// flapping threshold was reached. The caller must hold h.m.
func (h *Handler) recordSessionEndLocked(lifetime time.Duration, endedAt time.Time) bool {
	if lifetime >= flappingSessionMinLifetime {
		h.resetFlappingBreakerLocked()
		return false
	}

	keepFrom := 0
	for keepFrom < len(h.shortSessionEnds) && endedAt.Sub(h.shortSessionEnds[keepFrom]) >= flappingWindow {
		keepFrom++
	}
	h.shortSessionEnds = append(h.shortSessionEnds[keepFrom:], endedAt)

	if len(h.shortSessionEnds) < flappingStopThreshold {
		return false
	}

	h.resetFlappingBreakerLocked()
	return true
}

// resetFlappingBreakerLocked clears the current short-session streak. The
// caller must hold h.m.
func (h *Handler) resetFlappingBreakerLocked() {
	h.shortSessionEnds = nil
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
	h.log.Trace().Interface("msg_params", msg.Params).Msg("NOTICE from nickserv")

	if len(msg.Params) < 2 {
		return
	}

	h.m.RLock()
	expectingReply := h.identifyOutstanding && !h.authenticated && !h.saslauthed && h.network.Auth.NickServEnabled()
	h.m.RUnlock()
	if !expectingReply {
		return
	}

	// You're now logged in as test-bot
	// Password accepted - you are now recognized.
	if contains(msg.Params[1], "you're now logged in as", "password accepted", "you are now recognized", "you are now identified", "you are already logged in") {
		h.log.Debug().Interface("msg_params", msg.Params).Msg("NOTICE nickserv logged in")
		h.setAuthenticated()
		return
	}

	// The bare IDENTIFY cannot succeed on this network: either NickServ does not
	// know the nick we are connected as (it is not the account and has not been
	// grouped to it), or it wants the account spelled out. Both are fixed by the
	// account-qualified form, so retry once with it before treating anything as
	// terminal. Evaluated ahead of the failure branches below, which would
	// otherwise stop the network on the very notices that make escalation the
	// right move.
	if h.shouldEscalateIdentify(msg.Params[1]) {
		h.escalateIdentify()

		h.log.Debug().Str("notice", msg.Params[1]).Msg("nickserv rejected bare identify, retrying with account")

		h.NickServIdentify()
		return
	}

	if contains(msg.Params[1],
		"Invalid account credentials",
		"Authentication failed: Invalid account credentials",
		"password incorrect",
		"invalid password for", // atheme
	) {
		h.addConnectError(h.badCredentialsError())
		h.log.Error().Msg("NickServ: authentication failed - bad account credentials")

		h.unlearnIdentifyForm()

		h.stateMachine.OnError("nickserv authentication failed: bad credentials")

		// stop network and notify user
		h.Stop()
		return
	}

	if contains(msg.Params[1],
		"Account does not exist",
		"Authentication failed: Account does not exist",
		"isn't registered.",            // Nick ANICK isn't registered
		"is not a registered nickname", // atheme
	) {
		if h.CurrentNick() == h.PreferredNick() {
			h.addConnectError("authentication failed: account does not exist")

			h.unlearnIdentifyForm()

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
			h.log.Warn().Int("attempt", attempts).Msg("NickServ failed login attempts")
			h.addConnectError("authentication failed: nick in use and not authenticated")

			h.stateMachine.OnError("nickserv authentication failed: nick in use")

			// stop network and notify user
			h.Stop()
		}
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
	authenticated := h.authenticated
	saslauthed := h.saslauthed
	identifyOutstanding := h.identifyOutstanding
	nickServEnabled := h.network.Auth.NickServEnabled()
	h.m.RUnlock()

	switch {
	case authenticated || saslauthed:
		h.setAuthenticated()
	case identifyOutstanding:
		return
	case nickServEnabled:
		h.log.Trace().Msg("sending NickServ identify")
		h.NickServIdentify()
	default:
		h.setAuthenticated()
	}
}

// handleLoggedIn handles RPL_LOGGEDIN (900):
// <nick> <nick>!<user>@<host> <account> :You are now logged in as <account>
//
// The source of a numeric is the server, not a nick, so the target has to be
// read from the parameters.
func (h *Handler) handleLoggedIn(m ircmsg.Message) {
	if len(m.Params) < 3 {
		return
	}

	if !h.isOurCurrentNick(m.Params[0]) {
		return
	}

	h.log.Debug().Str("event", "900").Str("account", m.Params[2]).Msg("logged in")

	h.setAuthenticated()
}

// handleSASLSuccess we get here early so set saslauthed before we hit onConnect
func (h *Handler) handleSASLSuccess(_ ircmsg.Message) {
	h.m.Lock()
	h.saslauthed = true
	h.identifyOutstanding = false
	h.m.Unlock()
}

// handleBanned handles ERR_YOUREBANNEDCREEP (465): the server has refused us with
// a K-Line/G-Line ban and will close the link. This is a definitive, network-wide
// rejection, so we surface the ban reason and STOP the network rather than letting
// it reconnect - reconnecting cannot help and typically deepens the ban (the
// example G-Line is literally "reconnect loop").
func (h *Handler) handleBanned(msg ircmsg.Message) {
	// the ban reason is the trailing parameter, e.g.
	//   465 <nick> :You are not welcome on this network. G-Lined: reconnect loop.
	reason := ""
	if n := len(msg.Params); n > 0 {
		reason = strings.TrimSpace(msg.Params[n-1])
	}

	errMsg := "banned from network"
	if reason != "" {
		errMsg = fmt.Sprintf("banned from network: %s", reason)
	}

	h.log.Error().Str("reason", reason).Msg("banned from network (465)")

	h.addConnectError(errMsg)
	h.stateMachine.OnError(errMsg)

	// stop the network: reconnecting is futile and can worsen the ban
	h.Stop()
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
	h.identifyOutstanding = false
	if !alreadyAuthenticated {
		h.authenticated = true
		h.connectionErrors = []string{}
		h.failedNickServAttempts = 0

		// remember the form that worked so the next connect starts on it
		h.identifyFormLearned = h.identifyAttempt
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
			h.log.Error().Stack().Err(err).Str("command", h.network.InviteCommand).Msg("error sending connect command")
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

	if message == "CLIENTINFO" {
		return
	}

	if channel == h.CurrentNick() {
		// this is a DM - possibly an invite bot answering our invite command
		h.log.Debug().Str("direct-message", channel).Str("from-nick", nick).Str("msg", cleanMessage(message)).Msg("got direct-message")

		// a DM from an invite bot while a channel is still awaiting its invite is
		// a rejection, not the invite itself (that arrives as INVITE)
		h.handleInviteResponse(msg)

		// TODO create buffer with user/invite bot
		return
	}

	ircChannel, found := h.channels.Get(channel)
	if !found {
		if h.usesBouncer() {
			h.log.Trace().Str("channel", channel).Msg("ignoring message from unmonitored bouncer channel")
			return
		}

		h.log.Error().Str("channel", channel).Msg("channel not found")
		return
	}

	ircMsg, ok := ircChannel.OnMsg(msg)
	if !ok {
		return
	}

	// publish to SSE stream
	h.broadcastMessage(ircMsg)
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
				h.log.Error().Stack().Err(err).Str("channel", channel.Name).Msg("error joining channel")
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

	h.log.Debug().Str("command", strings.Join(params, " ")).Msg("sending JOIN command")

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
		ircChannel = NewChannel(h.log, h.GetNetwork().ID, channelName, false, false, nil)
		ircChannel.SetStateMachine(NewChannelStateMachine(ircChannel, h, ""))
		h.channels.Set(channelName, ircChannel)
	}

	ircChannel.Configure(channel.ID, channel.Enabled, channel.Password)

	if !ircChannel.IsEnabled() {
		h.log.Debug().Str("channel", channelName).Msg("channel added but disabled, not joining")
		return
	}

	if ircChannel.IsMonitoring() {
		// already joined and monitored, nothing to do
		return
	}

	h.log.Debug().Str("channel", channelName).Msg("adding and joining channel")

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
		h.log.Error().Stack().Err(err).Str("channel", channelName).Msg("error joining channel")
	}
}

// RemoveChannel parts a channel on a running handler and removes it from
// tracking so it no longer counts toward network health, and voids any pending
// retry timers on its state machine.
func (h *Handler) RemoveChannel(name string) {
	channelName := strings.ToLower(name)

	h.log.Debug().Str("channel", channelName).Msg("removing channel")

	if err := h.PartChannel(channelName); err != nil {
		h.log.Error().Stack().Err(err).Str("channel", channelName).Msg("error parting channel")
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
		h.log.Debug().Str("channel", channelName).Msg("channel disabled, parting")
		if err := h.PartChannel(channelName); err != nil {
			h.log.Error().Stack().Err(err).Str("channel", channelName).Msg("error parting channel")
		}
		ircChannel.ResetMonitoring()
		if sm := ircChannel.StateMachine(); sm != nil {
			sm.Reset()
		}

	case !monitoring:
		h.log.Debug().Str("channel", channelName).Msg("channel config changed, (re)joining")
		if sm := ircChannel.StateMachine(); sm != nil {
			sm.Reset()
			sm.Start()
		}

	default:
		h.log.Debug().Str("channel", channelName).Msg("channel config updated while monitoring; applied for next reconnect")
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
		h.log.Debug().Str("channel", channel).Msg("joined unwanted channel, parting")

		if err := h.PartChannel(channel); err != nil {
			h.log.Error().Stack().Err(err).Str("channel", channel).Msg("error parting channel")
		}

		return
	}

	h.log.Info().Str("channel", channel).Msg("join channel")
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

	h.log.Debug().Str("channel", channel).Msg("PART channel")

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

	h.log.Debug().Str("channel", channel).Msg("left channel")
}

// PartChannel parts/leaves channel
func (h *Handler) PartChannel(channel string) error {
	// if using bouncer we do not want to part any channels
	if h.usesBouncer() {
		h.log.Debug().Str("channel", channel).Msg("using bouncer, skipping part channel")
		return nil
	}

	h.log.Debug().Str("channel", channel).Msg("leaving channel")

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
		h.log.Trace().Interface("msg", msg).Msg("JOINED other user")
		return
	}

	// get channel
	channel := strings.ToLower(msg.Params[1])

	h.log.Debug().Str("channel", channel).Msg("JOINED")

	// check if channel is valid and if not lets part
	ircChannel, found := h.channels.Get(channel)
	if found {
		if sm := ircChannel.StateMachine(); sm != nil {
			sm.OnJoinSuccess()
		}

		//ircChannel.SetMonitoring()

		h.channels.Swap(channel, ircChannel)

		h.log.Trace().Str("channel", ircChannel.Name).Msg("set monitoring")

		if ircChannel.DefaultChannel {
			h.log.Info().Str("channel", channel).Msg("monitoring channel")
		} else {
			h.log.Info().Str("channel", channel).Msg("joined extra channel")
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
				h.log.Warn().Str("command", cmd).Msg("sleep command missing duration")
				continue
			}
			secs, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				h.log.Error().Err(err).Str("command", cmd).Msg("error parsing sleep command")
				continue
			}
			h.log.Debug().Int("seconds", secs).Str("command", cmd).Msg("sleeping")
			time.Sleep(time.Duration(secs) * time.Second)
			continue
		}

		h.log.Debug().Str("command", cmd).Msg("sending connect command")

		params := strings.SplitN(cmd, " ", 2)

		if err := h.Send("PRIVMSG", params...); err != nil {
			h.log.Error().Err(err).Str("command", cmd).Msg("error handling connect command")
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

	h.log.Trace().Str("nick", nick).Str("channel", channel).Msg("INVITE to join")
	h.log.Debug().Str("nick", nick).Str("channel", channel).Msg("INVITE, joining")

	ircChannel, found := h.channels.Get(channel)
	if !found {
		h.log.Trace().Str("nick", nick).Str("channel", channel).Msg("invite to join unwanted channel, skipping")
		return
	}

	if sm := ircChannel.StateMachine(); sm != nil {
		sm.OnInvite(nick)
		return
	}

	if err := h.Send("JOIN", channel); err != nil {
		h.log.Error().Stack().Err(err).Str("channel", channel).Msg("error handling join")
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

	// A JOIN rejection is channel-scoped: record it on the channel (deduped and
	// cleared when the channel recovers) and let it gate network health via the
	// channel. It must NOT go into the network-level connectionErrors bucket, which
	// is reserved for genuine network-wide failures (NickServ/SASL auth) and is only
	// cleared on a successful (re)authentication - routing per-channel join errors
	// there leaks them (they never clear on a no-auth network, and survive channel
	// recovery, misrepresenting a healthy network as carrying stale errors).
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

	h.log.Debug().Str("nick", nick).Msg("no such nick")

	for _, channel := range h.channels.Iterator() {
		if !channel.IsEnabled() {
			continue
		}

		if inviteBotNick(channel.InviteCommand()) == nick {
			h.log.Debug().Str("nick", nick).Msg("no such nick, sending invite command")

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

// handleInviteResponse routes a NOTICE or DM from an invite bot to the channels
// awaiting an invite from that bot. A successful invite arrives as an INVITE
// event (or, for some trackers, a server force-join), so a plain message from the
// bot is only a *possible* rejection - not a definitive one. It is therefore NOT
// failed immediately: the channel state machine records the bot's reason and waits
// a short grace for a JOIN before concluding the request was refused (bad IRC key,
// account not registered, etc.). This avoids false-failing bots like PTP's
// Hummingbird that answer "attempting to join you" and then force-join us. See
// ChannelStateMachine.OnInviteBotResponse.
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

		h.log.Debug().Str("bot", nick).Str("channel", channel.Name).Str("reason", reason).Msg("invite bot responded; awaiting join confirmation")
		sm.OnInviteBotResponse(errMsg)
	}
}

// identifyCommand builds the NickServ IDENTIFY for the form currently selected
// for this connection, bare unless escalateIdentify has moved us to the
// account-qualified form.
func (h *Handler) identifyCommand() string {
	h.m.RLock()
	defer h.m.RUnlock()

	return h.identifyCommandLocked()
}

func (h *Handler) identifyCommandLocked() string {
	form := h.identifyAttempt
	account := h.network.Auth.Account
	password := h.network.Auth.Password

	if form == identifyFormAccount && account != "" {
		return fmt.Sprintf("IDENTIFY %s %s", account, password)
	}

	return fmt.Sprintf("IDENTIFY %s", password)
}

// NickServIdentify sends a NickServ IDENTIFY. The whole command is one trailing
// PRIVMSG parameter: split across parameters it lands past the message body,
// where every ircd discards it.
func (h *Handler) NickServIdentify() error {
	h.m.Lock()
	if h.authenticated || h.saslauthed || !h.network.Auth.NickServEnabled() {
		h.identifyOutstanding = false
		h.m.Unlock()
		return nil
	}

	command := h.identifyCommandLocked()
	h.identifyOutstanding = true
	h.m.Unlock()

	if err := h.Send("PRIVMSG", "NickServ", command); err != nil {
		h.m.Lock()
		h.identifyOutstanding = false
		h.m.Unlock()
		h.log.Error().Stack().Err(err).Msg("error identifying with nickserv")
		return err
	}

	return nil
}

// canEscalateIdentify reports whether a failed bare IDENTIFY may be retried in
// the account-qualified form. Escalation is forward-only and happens at most
// once per connection: on services that do not support the account argument the
// retry comes back as "password incorrect", which is indistinguishable from
// genuinely bad credentials and must never drive another attempt.
func (h *Handler) canEscalateIdentify(currentNick string) bool {
	h.m.RLock()
	defer h.m.RUnlock()

	if h.authenticated || h.saslauthed || h.identifyEscalated || h.identifyAttempt != identifyFormBare {
		return false
	}

	if !h.network.Auth.NickServEnabled() || h.network.Auth.Account == "" {
		return false
	}

	// the account form resolves the same target as the bare form, so it can only
	// help when the account differs from the nick we are connected as
	return !strings.EqualFold(h.network.Auth.Account, currentNick)
}

// noticeAllowsIdentifyEscalation reports whether a NickServ NOTICE proves the
// bare IDENTIFY failed for a reason the account-qualified form can fix: either
// the nick we are connected as is unknown to NickServ, or NickServ wants the
// account spelled out.
func noticeAllowsIdentifyEscalation(notice string) bool {
	return contains(notice,
		"isn't registered",             // anope: Nick X isn't registered.
		"is not a registered nickname", // atheme
		"account does not exist",       // ergo
		"insufficient parameters",      // atheme with nickserv::no_nick_ownership
		"invalid parameters",           // ergo
		"help identify",                // anope, ircservices
		"syntax: identify",             // atheme, anope
	)
}

// shouldEscalateIdentify reports whether this NOTICE should move the connection
// to the account-qualified IDENTIFY form.
func (h *Handler) shouldEscalateIdentify(notice string) bool {
	return noticeAllowsIdentifyEscalation(notice) && h.canEscalateIdentify(h.CurrentNick())
}

// badCredentialsError describes a rejected password. An account-qualified
// attempt is ambiguous: the password may really be wrong, or the network may be
// running services that took the account name as the password.
func (h *Handler) badCredentialsError() string {
	h.m.RLock()
	escalated := h.identifyAttempt == identifyFormAccount
	h.m.RUnlock()

	if escalated {
		return "authentication failed: NickServ rejected the account-qualified IDENTIFY. Either the password is wrong, or this network only supports 'IDENTIFY <password>' and needs the nick grouped to the account instead"
	}

	return "authentication failed: Bad account credentials"
}

// escalateIdentify moves this connection to the account-qualified IDENTIFY form.
func (h *Handler) escalateIdentify() {
	h.m.Lock()
	h.identifyAttempt = identifyFormAccount
	h.identifyEscalated = true
	h.m.Unlock()
}

// resetIdentifyForm starts a connection on the form that last authenticated on
// this network and re-arms escalation. Seeding from the learned form keeps a
// reconnect from re-sending a bare IDENTIFY already known to fail here; the
// escalation guards reject escalating out of the account form, so a connection
// that starts there stays there.
// The caller must hold h.m.
func (h *Handler) resetIdentifyForm() {
	h.identifyAttempt = h.identifyFormLearned
	h.identifyEscalated = false
	h.identifyOutstanding = false
}

// unlearnIdentifyForm drops a remembered account-qualified form once it has
// itself been rejected, so a later nick grouping or services change heals on the
// next connect instead of failing the same way forever.
func (h *Handler) unlearnIdentifyForm() {
	h.m.Lock()
	defer h.m.Unlock()

	if h.identifyAttempt == identifyFormAccount {
		h.identifyFormLearned = identifyFormBare
	}
}

// NickChange sets a new nick for our user
func (h *Handler) NickChange(nick string) error {
	h.log.Debug().Str("nick", nick).Msg("NICK change")

	if client := h.getClient(); client != nil {
		client.SetNick(nick)
	}

	return nil
}

// CurrentNick returns our current nick set by the server
func (h *Handler) CurrentNick() string {
	if client := h.getClient(); client != nil {
		return client.CurrentNick()
	}

	return ""
}

// PreferredNick returns our preferred nick from settings
func (h *Handler) PreferredNick() string {
	if client := h.getClient(); client != nil {
		return client.PreferredNick()
	}

	return ""
}

// listens for MODE events
func (h *Handler) handleMode(msg ircmsg.Message) {
	h.log.Trace().Interface("msg", msg).Msg("MODE")

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
	h.log.Debug().Str("command", msg).Msg("sending msg command")

	if err := h.Send("PRIVMSG", channel, msg); err != nil {
		h.log.Error().Stack().Err(err).Str("command", msg).Msg("error sending msg")
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

	// dedup: a repeated identical failure (e.g. NickServ resending the same
	// "registered and protected" NOTICE several times) must not stack duplicates.
	if slices.Contains(h.connectionErrors, message) {
		return
	}

	h.connectionErrors = append(h.connectionErrors, message)
}

// clearConnectErrors drops any recorded network-level errors. It is called when
// the network reaches an operational state: a live, operational connection means
// any prior network-wide failure (a ban, a NickServ/SASL auth failure) no longer
// applies. This is the clear path for networks that never authenticate (no
// password/SASL/bot mode) and so never reach setAuthenticated, which also clears.
func (h *Handler) clearConnectErrors() {
	h.m.Lock()
	defer h.m.Unlock()

	h.connectionErrors = []string{}
}

func (h *Handler) ReportStatus(netw *domain.IrcNetworkWithHealth) {
	h.m.RLock()
	defer h.m.RUnlock()

	// only set connected and connected since if we have an active handler and connection
	if !h.network.Enabled {
		return
	}

	// Surface connection errors regardless of connection state. A network-level
	// failure (e.g. a NickServ authentication failure) drives the network into
	// Error and then Stop()s it - nilling the client and clearing connectedSince -
	// so if we only reported errors while connected, the UI would show the network
	// unhealthy with no reason at exactly the moment the reason matters. These
	// errors survive Stop()/onDisconnect and are cleared when the network next
	// reaches an operational state (clearConnectErrors) or re-authenticates.
	netw.ConnectionErrors = slices.Clone(h.connectionErrors)

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

// broadcastHealth pushes the current network-level health and connection errors to
// the UI via a HEALTH SSE event. Unlike the per-channel STATE events, this carries
// the network-wide failure reason (e.g. a NickServ authentication failure that
// stopped the whole network), so the UI can show WHY a network is unhealthy even
// when no individual channel error explains it. Called on settled network state
// transitions so the reason appears (and clears) in real time rather than only on
// the next poll.
func (h *Handler) broadcastHealth() {
	h.m.RLock()
	healthData := map[string]any{
		"network":           h.network.ID,
		"healthy":           h.computeHealthy(),
		"connection_errors": slices.Clone(h.connectionErrors),
	}
	h.m.RUnlock()

	// connected_since is deliberately omitted: it is the zero time while the network
	// is errored/stopped (exactly when this fires), which the UI would apply as a
	// bogus connection time. The periodic status poll (ReportStatus) is the source
	// of truth for connected_since.
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
