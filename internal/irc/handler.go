// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"crypto/tls"
	"encoding/json"
	stdErr "errors"
	"fmt"
	"math/rand/v2"
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

	clientManuallyDisconnected = errors.New("IRC client was manually disconnected")
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

// Flapping circuit breaker. A connection that repeatedly comes up and dies
// again cannot be fixed by reconnecting to it, so at some point the network has
// to stop and say why rather than keep returning to a server that clearly does
// not want it. flappingStopThreshold sessions shorter than
// flappingSessionMinLifetime, all within flappingWindow, stop the network; a
// session that lives longer clears the count, and so does time passing, because
// one drop per maintenance window is not flapping however often it recurs.
//
// Two paths feed it, because irc-go runs the disconnect callback only for
// connections that finished registering: onDisconnect covers registered
// sessions, and handleServerError covers a server that rejects us earlier with
// an ERROR line (the usual "reconnecting too fast" throttle), counted once per
// attempt. Connect-level failures - refused, unreachable, TLS - are not strikes:
// the reconnect backoff already paces those, and a tracker being down should
// not stop a network that will work again when it returns.
const (
	flappingSessionMinLifetime = 30 * time.Second
	flappingStopThreshold      = 5
	flappingWindow             = 15 * time.Minute
)

// Reconnect backoff. autobrr drives its own reconnect loop rather than the
// irc-go client's, which retries at a flat interval forever and reports nothing
// about attempts that fail before registration. Delays double from
// reconnectBaseDelay and level off at reconnectMaxDelay instead of the loop ever
// giving up, so an unreachable network recovers by itself once the server is
// back. Reaching the server resets the schedule. Jitter spreads the return of
// every autobrr instance across the window, so a tracker coming back up is not
// met by all of them at once.
const (
	reconnectBaseDelay     = 15 * time.Second
	reconnectMaxDelay      = 15 * time.Minute
	reconnectJitterDivisor = 5 // up to a fifth of the delay on top
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

	log                 zerolog.Logger
	sse                 sseServer
	network             *domain.IrcNetwork
	releaseSvc          releaseService
	notificationService notificationSender
	announceProcessors  map[string]announce.Processor
	definitions         map[string]*domain.IndexerDefinition

	client                   *ircevent.Connection
	clientState              ircState
	clientGen                uint64
	connectedSince           time.Time
	haveDisconnected         bool
	consecutiveShortSessions int
	firstShortSession        time.Time
	connectAttempts          int
	errorCounted             bool
	authenticated            bool
	saslauthed               bool

	// stopSig is closed by Stop to wake superviseConnection out of either its wait
	// for the session to end or a reconnect backoff. It is replaced on every Run,
	// which is also how a run recognises that it has been superseded.
	stopSig chan struct{}

	identifyAttempt     identifyForm
	identifyEscalated   bool
	identifyFormLearned identifyForm

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

// Run brings the network up and, once it is live, hands it to
// superviseConnection which owns every reconnect from then on. It returns nil
// as soon as the network is connected, or an error if connecting was abandoned.
func (h *Handler) Run() (err error) {
	// TODO validate
	// check if network requires nickserv
	// check if network or channels requires invite command

	// A manual (re)start opens a fresh window for the backoff and the breaker
	// alike, and gets its own signalling channels so a previous run's supervisor
	// can never be woken by this one. Claiming the state and publishing the
	// channels together means a Stop can never land in between and go unnoticed.
	sessionEnded := make(chan uint64, 1)
	stopSig := make(chan struct{})

	shouldConnect := false
	h.m.Lock()
	if h.clientState == ircStopped {
		shouldConnect = true
		h.clientState = ircConnecting
		h.connectAttempts = 0
		h.consecutiveShortSessions = 0
		h.stopSig = stopSig
	}
	h.m.Unlock()

	if !shouldConnect {
		return connectionInProgress
	}

	h.stateMachine.OnConnecting()

	// either we will successfully transition to `StateRunning`, or else
	// we need to reset the state to `ircStopped`
	defer h.releaseConnectingState(stopSig)

	client, err := h.connectWithBackoff(sessionEnded, stopSig)
	if err != nil {
		return err
	}

	shouldDisconnect := false
	h.m.Lock()
	switch {
	case h.stopSig != stopSig:
		// stopped, or superseded by a newer run, while we were connecting
		shouldDisconnect = true
	case h.clientState == ircConnecting:
		// success!
		h.clientState = ircLive
	default:
		h.log.Error().Stack().Str("state", strconv.Itoa(int(h.clientState))).Msg("unexpected client state after connecting")
		shouldDisconnect = true
	}
	h.m.Unlock()

	if shouldDisconnect {
		// quit the connection this run built, not whatever is current: a Stop has
		// already cleared h.client, and a newer run's client must not be touched
		client.Quit()
		return clientManuallyDisconnected
	}

	go h.superviseConnection(sessionEnded, stopSig)

	return nil
}

// releaseConnectingState returns the handler to stopped if this run is still the
// one holding the connecting state. The ownership check is what makes Stop safe
// to call while a run is mid-connect: Stop wakes that run immediately and the
// caller starts a new one, and clearing the state the new run just claimed would
// abort it before it ever connects.
func (h *Handler) releaseConnectingState(stopSig chan struct{}) {
	h.m.Lock()
	if h.stopSig == stopSig && h.clientState == ircConnecting {
		h.clientState = ircStopped
	}
	h.m.Unlock()
}

// newClient builds the connection for a single attempt. Every attempt gets its
// own Connection because the library closes a session's socket only from
// Connect's error path or from Loop, so a Connection whose session has ended
// cannot be reused safely. Callbacks are registered separately by
// wireCallbacks, once the attempt has claimed its generation.
func (h *Handler) newClient() (*ircevent.Connection, error) {
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

	client := &ircevent.Connection{
		Nick:      network.Nick,
		User:      network.Auth.Account,
		RealName:  network.Auth.Account,
		Password:  network.Pass,
		Server:    addr,
		KeepAlive: 4 * time.Minute,
		Timeout:   2 * time.Minute,
		// Loop() never gets to use this: every session quits it from the disconnect
		// callback, so reconnecting is superviseConnection's alone. Keep it absurdly
		// long anyway - if a library upgrade ever broke that teardown, a second
		// reconnect loop firing every 15 seconds would be the worst way to find out.
		ReconnectFreq: 24 * time.Hour,
		Version:       "autobrr",
		QuitMessage:   "bye from autobrr",
		Debug:         true,
		Log:           subLogger,
	}

	// a proxied network must never fall back to a direct connection - the proxy
	// missing here is a service-side bug, and dialling on regardless would hand
	// the tracker the user's real IP with nothing in the UI saying so
	if network.UseProxy && network.Proxy == nil {
		return nil, errors.New("network is set to use a proxy but none is attached")
	}

	if network.UseProxy && network.Proxy != nil {
		if !network.Proxy.Enabled {
			h.log.Warn().Str("proxy", network.Proxy.Name).Msg("network is set to use a proxy but the proxy is disabled; connecting directly")
		} else {
			if network.Proxy.Addr == "" {
				return nil, errors.New("proxy addr missing")
			}

			proxyUrl, err := url.Parse(network.Proxy.Addr)
			if err != nil {
				return nil, errors.Wrap(err, "could not parse proxy url: %s", network.Proxy.Addr)
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
					return nil, errors.Wrap(err, "could not create proxy dialer from url: %s", network.Proxy.Addr)
				}
			}

			proxyContextDialer, ok := proxyDialer.(proxy.ContextDialer)
			if !ok {
				return nil, errors.New("proxy dialer does not expose DialContext(): %v", proxyDialer)
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

	return client, nil
}

// gated wraps a callback so it runs only while the attempt identified by gen is
// still the current one. The library delivers callbacks asynchronously: a
// message read moments before its session was superseded - by a stop, a restart
// or a newer attempt - can arrive after the replacement is already live, and
// un-gated it would mutate the replacement's state: advance its state machine,
// stop the network over the old session's stale failure, feed its breaker.
func (h *Handler) gated(gen uint64, fn func(ircmsg.Message)) func(ircmsg.Message) {
	return func(msg ircmsg.Message) {
		if !h.ownsClientGen(gen) {
			return
		}

		fn(msg)
	}
}

// wireCallbacks registers every handler callback on the client, each gated to
// the attempt that owns gen.
func (h *Handler) wireCallbacks(client *ircevent.Connection, gen uint64) {
	on := func(command string, fn func(ircmsg.Message)) {
		client.AddCallback(command, h.gated(gen, fn))
	}

	// onConnect and the handlers that can stop the network get gen threaded in:
	// their bodies re-check ownership before acting, because the entry gate
	// cannot cover a body that is paused - by onConnect's own sleep, or by the
	// scheduler across a stop and restart - and resumes against the replacement
	client.AddConnectCallback(h.gated(gen, func(m ircmsg.Message) { h.onConnect(gen, m) }))

	on("MODE", h.handleMode)
	if h.GetNetwork().BotMode {
		on(ircevent.ERR_UMODEUNKNOWNFLAG, h.handleModeUnknownFlag)
	}
	on("INVITE", h.handleInvite)
	on("PART", h.handlePart)
	on("PRIVMSG", h.onPrivMessage)
	on("NOTICE", func(m ircmsg.Message) { h.onNotice(gen, m) })
	on("NICK", h.onNick)
	on("KICK", h.onKick)
	on("JOIN", h.handleJoin)

	on("TOPIC", h.handleTopicChange)
	on(ircevent.RPL_TOPIC, h.handleTopic)
	on(ircevent.RPL_ENDOFNAMES, h.handleJoined) // end of names

	on(ircevent.RPL_LOGGEDIN, h.handleLoggedIn)
	on(ircevent.RPL_SASLSUCCESS, h.handleSASLSuccess)
	on(ircevent.ERR_SASLFAIL, func(m ircmsg.Message) { h.handleSASLFail(gen, m) })

	// the server has banned us (K-Line/G-Line); stop and surface the reason
	on(ircevent.ERR_YOUREBANNEDCREEP, func(m ircmsg.Message) { h.handleBanned(gen, m) }) // 465

	// the server is closing the link and saying why; the pre-registration case
	// is invisible to the disconnect callback, so the breaker counts it here
	on("ERROR", h.handleServerError)

	// surface failed JOIN attempts on the affected channel
	on(ircevent.ERR_CHANNELISFULL, h.handleJoinError)  // 471
	on(ircevent.ERR_INVITEONLYCHAN, h.handleJoinError) // 473
	on(ircevent.ERR_BANNEDFROMCHAN, h.handleJoinError) // 474
	on(ircevent.ERR_BADCHANNELKEY, h.handleJoinError)  // 475
	on(ircevent.ERR_NEEDREGGEDNICK, h.handleJoinError) // 477
	on(ircevent.ERR_NOSUCHNICK, h.handleErrNoSuchNick)
}

// connectOnce runs a single connection attempt on a freshly built client and
// returns the live connection, or the reason there is not one.
func (h *Handler) connectOnce(sessionEnded chan uint64, stopSig <-chan struct{}) (*ircevent.Connection, error) {
	client, err := h.newClient()
	if err != nil {
		return nil, configError{err}
	}

	h.m.Lock()
	// The claim below must be refused outright for a run that is no longer the
	// current one, not merely resolved by the ownership checks after Connect. An
	// attempt from a stopped run that was already past its loop's stop check
	// would otherwise supersede the replacement run's connection and gate every
	// one of its callbacks off.
	if h.stopSig != stopSig {
		h.m.Unlock()
		return nil, clientManuallyDisconnected
	}

	h.saslauthed = false
	h.errorCounted = false
	h.clientGen++
	gen := h.clientGen
	h.client = client
	h.resetIdentifyForm()
	h.m.Unlock()

	h.wireCallbacks(client, gen)

	client.AddDisconnectCallback(func(msg ircmsg.Message) {
		// This session is over and the Connection is single-use, so make the
		// library's Loop() exit and tear the socket down instead of reconnecting on
		// its own schedule - reconnecting is superviseConnection's job. Disconnect
		// callbacks run before the read loop releases the wait group Loop is
		// blocked on, so Loop is guaranteed to observe the quit.
		client.Quit()

		h.onSessionEnded(gen, msg, sessionEnded)
	})

	if err := client.Connect(); err != nil {
		return nil, err
	}

	// Loop only tears this session down once it ends; every reconnect goes
	// through superviseConnection so that one policy governs all of them.
	go client.Loop()

	// The run can be stopped, or superseded by a restart, while this attempt is
	// still dialling - and Stop can only quit a client that was already published,
	// which this one may not have been. Nothing else will ever tear this session
	// down, so quit it here rather than leave it registered, joined to the
	// announce channels and feeding a network the user has stopped.
	if h.stoppedFor(stopSig) || !h.ownsClientGen(gen) {
		h.log.Debug().Msg("discarding a connection that completed after the network was stopped")

		client.Quit()

		return nil, clientManuallyDisconnected
	}

	return client, nil
}

// onSessionEnded handles a finished session on behalf of the attempt that owns
// gen. An attempt that has been superseded must stay silent: its connection is
// no longer the network's, so letting it run the disconnect bookkeeping would
// reset the live session's channels, drive its state machine to Disconnected
// and count a foreign failure toward the flapping breaker.
func (h *Handler) onSessionEnded(gen uint64, msg ircmsg.Message, sessionEnded chan uint64) {
	if !h.ownsClientGen(gen) {
		h.log.Debug().Msg("ignoring the end of a superseded connection")
		return
	}

	// Deliver the wakeup even if the bookkeeping below panics: the library
	// swallows callback panics, and a lost token leaves the supervisor waiting
	// forever for a session end that already happened.
	defer func() {
		select {
		case sessionEnded <- gen:
		default:
			// The buffer can be holding a stale token: an attempt can complete
			// registration and still fail Connect (a SASL stall, a socket dying on
			// the welcome boundary), and its leftover would turn this send into a
			// no-op right as the live session ends - permanently, since nothing else
			// wakes the supervisor. Only the newest token matters, so displace the
			// old one. The retry cannot race another send: generations are claimed
			// under the lock, so there is never more than one current sender.
			select {
			case <-sessionEnded:
			default:
			}
			select {
			case sessionEnded <- gen:
			default:
			}
		}
	}()

	h.onDisconnect(msg)
}

// ownsClientGen reports whether the attempt identified by gen is still the most
// recent one, i.e. whether its connection is the network's current connection.
func (h *Handler) ownsClientGen(gen uint64) bool {
	h.m.RLock()
	defer h.m.RUnlock()

	return h.clientGen == gen
}

// waitForSessionEnd blocks until the current session ends, reporting false if
// the network was stopped instead. Signals are tagged with the attempt that sent
// them: an attempt can register and then still fail, leaving a token behind that
// would otherwise wake the supervisor while a later connection is live and send
// it off to reconnect on top of it.
func (h *Handler) waitForSessionEnd(sessionEnded chan uint64, stopSig <-chan struct{}) bool {
	for {
		select {
		case gen := <-sessionEnded:
			if h.ownsClientGen(gen) {
				return true
			}

			h.log.Debug().Msg("ignoring a session-end signal from a superseded connection")

		case <-stopSig:
			return false
		}
	}
}

// connectWithBackoff attempts to connect until it succeeds, the network is
// stopped, or the failure is one that retrying cannot fix. Delays grow
// exponentially from reconnectBaseDelay and level off at reconnectMaxDelay
// rather than giving up, so a network that is merely unreachable recovers on
// its own whenever the server comes back.
func (h *Handler) connectWithBackoff(sessionEnded chan uint64, stopSig <-chan struct{}) (*ircevent.Connection, error) {
	for {
		// #1239: don't retry if the user manually disconnected with Stop()
		if h.stoppedFor(stopSig) {
			return nil, clientManuallyDisconnected
		}

		client, err := h.connectOnce(sessionEnded, stopSig)
		if err == nil {
			// reaching the server clears the schedule: an outage tomorrow must not
			// inherit the delay an outage today grew to
			h.resetConnectBackoff()

			return client, nil
		}

		h.log.Error().Err(err).Msg("client encountered connection error")

		// A fatal in-band failure (a ban/G-Line, a SASL or NickServ auth failure)
		// is detected by its callback DURING registration and stops the network
		// before Connect() returns its error. Such a failure is not transient, so
		// abort immediately rather than waiting out a delay to notice the stop.
		if h.stoppedFor(stopSig) {
			return nil, err
		}

		if reason, fatal := fatalConnectError(err); fatal {
			h.log.Error().Str("reason", reason).Msg("stopping network: connection cannot succeed")

			h.addConnectError(reason)
			h.stateMachine.OnError(reason)
			h.Stop()

			return nil, err
		}

		delay := h.nextConnectDelay()
		h.log.Debug().Dur("delay", delay).Msg("waiting before next connect attempt")

		if !h.waitOrStop(delay, stopSig) {
			return nil, clientManuallyDisconnected
		}
	}
}

// superviseConnection owns the connection for the rest of its life: it waits
// for the live session to end and then reconnects under the same policy as the
// first connect. The irc-go client would otherwise reconnect on its own at a
// flat interval, with no attempt accounting and no visibility into failures
// that happen before registration.
func (h *Handler) superviseConnection(sessionEnded chan uint64, stopSig <-chan struct{}) {
	for {
		connectedAt := time.Now()

		if !h.waitForSessionEnd(sessionEnded, stopSig) {
			return
		}

		if h.stoppedFor(stopSig) {
			return
		}

		if delay := h.reconnectDelayAfter(time.Since(connectedAt)); delay > 0 {
			if !h.waitOrStop(delay, stopSig) {
				return
			}
		}

		client, err := h.connectWithBackoff(sessionEnded, stopSig)
		if err != nil {
			h.log.Debug().Err(err).Msg("stopped reconnecting")
			return
		}

		// the network can be stopped while the attempt is in flight; connectOnce
		// discards such a connection, but re-check so the supervisor never settles
		// down to wait for a session that will not arrive
		if h.stoppedFor(stopSig) {
			client.Quit()
			return
		}
	}
}

// fatalConnectError reports failures that every subsequent attempt would repeat
// identically, so the network should stop with a reason instead of retrying.
func fatalConnectError(err error) (string, bool) {
	// An expired or not-yet-valid certificate, an unknown CA or a hostname
	// mismatch keeps failing until the tracker fixes its certificate or the user
	// enables TLSSkipVerify.
	if certErr, ok := stdErr.AsType[*tls.CertificateVerificationError](err); ok {
		return fmt.Sprintf("TLS certificate verification failed: %v", certErr.Err), true
	}

	// The network's own settings cannot be dialled at all - a malformed proxy
	// URL, say. Retrying re-reads the same settings, so it can only fail again.
	if cfgErr, ok := stdErr.AsType[configError](err); ok {
		return cfgErr.Error(), true
	}

	return "", false
}

// configError marks a failure to build a connection out of the network's own
// settings, as opposed to a failure to reach the server.
type configError struct{ error }

func (e configError) Unwrap() error { return e.error }

// nextConnectDelay returns how long to wait before the next attempt, doubling
// per consecutive failure up to the cap. The jitter keeps every autobrr from
// returning to a recovering tracker in the same instant.
func (h *Handler) nextConnectDelay() time.Duration {
	h.m.Lock()
	h.connectAttempts++
	attempt := h.connectAttempts
	h.m.Unlock()

	delay := reconnectBaseDelay
	for range attempt - 1 {
		if delay >= reconnectMaxDelay {
			break
		}
		delay *= 2
	}

	if delay > reconnectMaxDelay {
		delay = reconnectMaxDelay
	}

	return delay + rand.N(delay/reconnectJitterDivisor)
}

func (h *Handler) resetConnectBackoff() {
	h.m.Lock()
	h.connectAttempts = 0
	h.m.Unlock()
}

// reconnectDelayAfter returns how long to wait before reconnecting, given how
// long the session that just ended lasted. A session that held for a useful
// length of time proves the connection works, so we come straight back the way
// any IRC client does after a netsplit - pausing there would cost announces for
// nothing. A session that did not hold is the flapping case: pace the return so
// a server that keeps dropping us is not hammered.
func (h *Handler) reconnectDelayAfter(lifetime time.Duration) time.Duration {
	if lifetime >= flappingSessionMinLifetime {
		return 0
	}

	return h.nextConnectDelay()
}

// stoppedFor reports whether the run that owns stopSig should give up. It
// answers for that run specifically rather than for the handler as a whole:
// a Restart replaces the handler's state, so a supervisor from the previous run
// that was blocked in Connect while the swap happened would otherwise see a
// freshly live handler and keep reconnecting alongside the new one.
func (h *Handler) stoppedFor(stopSig <-chan struct{}) bool {
	select {
	case <-stopSig:
		return true
	default:
	}

	return h.Stopped()
}

// waitOrStop sleeps for d, reporting false if the network was stopped first so
// a user disabling a network is not left waiting out a long backoff.
func (h *Handler) waitOrStop(d time.Duration, stopSig <-chan struct{}) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-stopSig:
		return false
	}
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
	h.authenticated = false
	h.haveDisconnected = false
	h.resetIdentifyForm()
	client := h.client
	h.clientState = ircStopped
	h.client = nil
	// everything the stopped session still delivers is stale from here on: its
	// drain can outlive this stop by minutes when the server ignores our QUIT,
	// and un-superseded its callbacks would be charged to whatever runs next -
	// a strike on the breaker, a bogus disconnect notification, a stale auth
	// failure stopping a freshly fixed network
	h.clientGen++
	// wake superviseConnection wherever it is waiting; cleared so a second Stop
	// (the breaker and a user action can race) cannot close it twice
	if h.stopSig != nil {
		close(h.stopSig)
		h.stopSig = nil
	}
	h.m.Unlock()

	if client != nil {
		h.log.Debug().Msg("Disconnecting...")
		h.resetChannelState()
		client.Quit()
	}

	// the disconnect callback used to park the state machine, but it is gated
	// off for a superseded generation, so the stop owns it - including a stop
	// that lands mid-dial, before any client was published
	h.stateMachine.OnStopped()
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
func (h *Handler) onConnect(gen uint64, m ircmsg.Message) {
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

		h.notificationService.Send(domain.NotificationEventIRCReconnected, domain.NotificationPayload{
			Subject: "IRC Reconnected",
			Message: fmt.Sprintf("Network: %s", networkName),
		})
	}

	h.log.Info().Msg("network connected")

	// stored credentials that the mechanism excludes are silently unused, which
	// looks exactly like an auth failure from the outside (registered-only
	// channels refuse the join) - say so once per connection instead
	h.m.RLock()
	auth := h.network.Auth
	h.m.RUnlock()

	if auth.Mechanism == domain.IRCAuthMechanismNone && auth.Password != "" {
		h.log.Warn().Msg("network has stored credentials but its identification mechanism is None, so they are not used")
	}

	time.Sleep(1 * time.Second)

	// The gate passed a full second ago; a stop or restart fits comfortably
	// inside that sleep. Driving the state machine for a superseded session is
	// not self-healing: OnConnected walks it to JoiningChannels, every channel
	// errors against the quit client, and the next session's OnConnected has no
	// valid transition out - the network registers but monitors nothing.
	if !h.ownsClientGen(gen) {
		return
	}

	// Notify state machine of connection - it will handle auth and channel joining
	h.stateMachine.OnConnected()
}

// onDisconnect is the disconnect callback
func (h *Handler) onDisconnect(_ ircmsg.Message) {
	h.log.Debug().Msg("disconnect")

	h.m.Lock()

	// connectedSince can be zero if the session died before StateConnected's
	// entry action recorded it; that is still a short session, not a long one
	sessionLifetime := time.Duration(0)
	if !h.connectedSince.IsZero() {
		sessionLifetime = time.Since(h.connectedSince)
	}

	h.connectedSince = time.Time{}
	h.authenticated = false

	// a reconnect starts the identify ladder over: the nick we get back may
	// differ, and the previous connection's escalation says nothing about this one
	h.resetIdentifyForm()

	h.haveDisconnected = true

	manuallyDisconnected := h.clientState == ircStopped
	networkName := h.network.Name

	h.m.Unlock()

	// reset channels monitored status and channel state machines so they
	// rejoin cleanly on reconnect instead of getting stuck in Monitoring
	h.resetChannelState()

	if !manuallyDisconnected && h.noteSessionEnded(sessionLifetime) {
		h.tripFlappingBreaker("")
		return
	}

	if !manuallyDisconnected {
		// only send notification if we did not initiate disconnect/restart/stop
		h.notificationService.Send(domain.NotificationEventIRCDisconnected, domain.NotificationPayload{
			Subject: "IRC Disconnected unexpectedly",
			Message: fmt.Sprintf("Network: %s", networkName),
		})
	}

	h.stateMachine.OnDisconnected()
}

// noteSessionEnded records the outcome of one connection attempt and reports
// whether it completed a flapping streak. A session that reached
// flappingSessionMinLifetime proves the connection can hold and clears the
// streak, so only genuinely repeated failures trip the breaker.
func (h *Handler) noteSessionEnded(lifetime time.Duration) bool {
	h.m.Lock()
	defer h.m.Unlock()

	if lifetime >= flappingSessionMinLifetime {
		h.consecutiveShortSessions = 0
		return false
	}

	// Strikes expire. Flapping means failing repeatedly in quick succession; a
	// network dropped once during a maintenance window is not flapping however
	// many maintenance windows it sees, and stopping it for that would leave a
	// perfectly good network down until someone noticed.
	now := time.Now()
	if h.consecutiveShortSessions == 0 || now.Sub(h.firstShortSession) > flappingWindow {
		h.consecutiveShortSessions = 0
		h.firstShortSession = now
	}

	h.consecutiveShortSessions++
	if h.consecutiveShortSessions < flappingStopThreshold {
		return false
	}

	h.consecutiveShortSessions = 0

	return true
}

// tripFlappingBreaker stops the network after repeated failed connections and
// surfaces why. serverReason is the server's own explanation when it gave one
// (an ERROR line), otherwise empty.
func (h *Handler) tripFlappingBreaker(serverReason string) {
	errMsg := fmt.Sprintf("connection flapping: %d consecutive connections failed or lasted under %s; network stopped to avoid hammering the server - fix the underlying issue, then restart the network", flappingStopThreshold, flappingSessionMinLifetime)
	if serverReason != "" {
		errMsg = fmt.Sprintf("%s. Server said: %s", errMsg, serverReason)
	}

	h.log.Error().Int("attempts", flappingStopThreshold).Dur("min_lifetime", flappingSessionMinLifetime).Str("server_reason", serverReason).Msg("connection flapping; stopping network")

	h.addConnectError(errMsg)

	h.notificationService.Send(domain.NotificationEventIRCDisconnected, domain.NotificationPayload{
		Subject: "IRC network stopped",
		Message: fmt.Sprintf("Network: %s stopped after repeated failed connections", h.GetNetwork().Name),
	})

	h.stateMachine.OnError(errMsg)
	h.Stop()
}

// handleServerError handles an ERROR line, which is how a server explains why it
// is closing the link - most importantly BEFORE registration completes
// ("Trying to reconnect too fast", connection-limit and ban messages). Those
// sessions never reach the disconnect callback (irc-go only runs it for
// registered connections), so without counting them here the backoff loop
// would keep returning to a server that is actively refusing us, which is
// exactly the hammering the breaker exists to stop. A registered session is
// left to onDisconnect so one failure is never counted twice.
func (h *Handler) handleServerError(msg ircmsg.Message) {
	reason := ""
	if n := len(msg.Params); n > 0 {
		reason = strings.TrimSpace(msg.Params[n-1])
	}

	h.m.RLock()
	manuallyDisconnected := h.clientState == ircStopped
	registered := !h.connectedSince.IsZero()
	h.m.RUnlock()

	// servers commonly answer our own QUIT with an ERROR line
	if manuallyDisconnected {
		return
	}

	if registered {
		// onDisconnect owns the accounting for a session that got this far
		return
	}

	h.log.Warn().Str("reason", reason).Msg("server refused the connection before registration")

	// surface the server's own words: the connection error is otherwise the only
	// thing the user can see, and "connection refused" says far less than the
	// reason the server gave for refusing. No state changes here, so nothing
	// else broadcasts the update - push it rather than wait for the next poll
	if reason != "" {
		h.addConnectError(fmt.Sprintf("server refused the connection: %s", reason))
		h.broadcastHealth()
	}

	// One attempt is one strike. Servers are free to send several ERROR lines
	// before closing the link, and counting each of them would stop the network
	// on a single refusal while reporting it as five.
	h.m.Lock()
	alreadyCounted := h.errorCounted
	h.errorCounted = true
	h.m.Unlock()

	if alreadyCounted {
		return
	}

	if h.noteSessionEnded(0) {
		h.tripFlappingBreaker(reason)
	}
}

// onNotice handles NOTICE events
func (h *Handler) onNotice(gen uint64, msg ircmsg.Message) {
	switch msg.Nick() {
	case "NickServ":
		h.handleNickServ(gen, msg)
	default:
		// a NOTICE from an invite bot while a channel is still awaiting its
		// invite is a rejection, not the invite itself (that arrives as INVITE)
		h.handleInviteResponse(msg)
	}
}

// handleNickServ is called from NOTICE events
func (h *Handler) handleNickServ(gen uint64, msg ircmsg.Message) {
	// re-checked past the entry gate: several branches below stop the network,
	// and a delivery paused across a stop and restart must act on the session it
	// belongs to, not on the replacement
	if !h.ownsClientGen(gen) {
		return
	}

	h.log.Trace().Interface("msg_params", msg.Params).Msg("NOTICE from nickserv")

	if len(msg.Params) < 2 {
		return
	}

	// We never identify on this network, so nothing sent by a nick calling itself
	// NickServ is an answer to us. Ignoring it outright matters most on the very
	// networks the user opted out of services for: nick registration is not
	// enforced there, so anyone can take the nick and would otherwise be able to
	// elicit an IDENTIFY (escalation sends the stored password) or stop the
	// network with a bogus rejection.
	h.m.RLock()
	nickServEnabled := h.network.Auth.NickServEnabled()
	h.m.RUnlock()

	if !nickServEnabled {
		h.log.Trace().Msg("ignoring nickserv notice: services authentication is not enabled for this network")
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
	shouldSendNickserv := !h.authenticated && !h.saslauthed && h.network.Auth.NickServEnabled()
	h.m.RUnlock()

	if shouldSendNickserv {
		h.log.Trace().Msg("on connect not authenticated and password not empty: send nickserv identify")
		h.NickServIdentify()
	} else {
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
	h.m.Unlock()
}

// handleBanned handles ERR_YOUREBANNEDCREEP (465): the server has refused us with
// a K-Line/G-Line ban and will close the link. This is a definitive, network-wide
// rejection, so we surface the ban reason and STOP the network rather than letting
// it reconnect - reconnecting cannot help and typically deepens the ban (the
// example G-Line is literally "reconnect loop").
func (h *Handler) handleBanned(gen uint64, msg ircmsg.Message) {
	// re-checked past the entry gate: this stops the network, and a delivery
	// paused across a stop and restart must not stop the replacement
	if !h.ownsClientGen(gen) {
		return
	}

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

func (h *Handler) handleSASLFail(gen uint64, _ ircmsg.Message) {
	// re-checked past the entry gate: this stops the network, and a delivery
	// paused across a stop and restart must not stop the replacement
	if !h.ownsClientGen(gen) {
		return
	}

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
	form := h.identifyAttempt
	account := h.network.Auth.Account
	password := h.network.Auth.Password
	h.m.RUnlock()

	if form == identifyFormAccount && account != "" {
		return fmt.Sprintf("IDENTIFY %s %s", account, password)
	}

	return fmt.Sprintf("IDENTIFY %s", password)
}

// NickServIdentify sends a NickServ IDENTIFY. The whole command is one trailing
// PRIVMSG parameter: split across parameters it lands past the message body,
// where every ircd discards it.
func (h *Handler) NickServIdentify() error {
	if err := h.Send("PRIVMSG", "NickServ", h.identifyCommand()); err != nil {
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

	// escalation sends the password, so it answers to the same gate as the first
	// IDENTIFY: a network that does not use services must never emit one
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
