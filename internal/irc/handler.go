package irc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/rs/zerolog/log"
	"gopkg.in/irc.v3"
)

var (
	connectTimeout = 15 * time.Second
)

type channelHealth struct {
	name            string
	monitoring      bool
	monitoringSince time.Time
	lastAnnounce    time.Time
}

// SetLastAnnounce set last announce to now
func (h *channelHealth) SetLastAnnounce() {
	h.lastAnnounce = time.Now()
}

// SetMonitoring set monitoring and time
func (h *channelHealth) SetMonitoring() {
	h.monitoring = true
	h.monitoringSince = time.Now()
}

// resetMonitoring remove monitoring and time
func (h *channelHealth) resetMonitoring() {
	h.monitoring = false
	h.monitoringSince = time.Time{}
}

type Handler struct {
	network            *domain.IrcNetwork
	filterService      filter.Service
	releaseService     release.Service
	announceProcessors map[string]announce.Processor
	definitions        map[string]*domain.IndexerDefinition

	client  *irc.Client
	conn    net.Conn
	ctx     context.Context
	stopped chan struct{}
	cancel  context.CancelFunc

	lastPing       time.Time
	connected      bool
	connectedSince time.Time
	// tODO disconnectedTime

	validAnnouncers map[string]struct{}
	validChannels   map[string]struct{}
	channelHealth   map[string]*channelHealth
}

func NewHandler(network domain.IrcNetwork, filterService filter.Service, releaseService release.Service, definitions []domain.IndexerDefinition) *Handler {
	h := &Handler{
		client:             nil,
		conn:               nil,
		ctx:                nil,
		stopped:            make(chan struct{}),
		network:            &network,
		filterService:      filterService,
		releaseService:     releaseService,
		definitions:        map[string]*domain.IndexerDefinition{},
		announceProcessors: map[string]announce.Processor{},
		validAnnouncers:    map[string]struct{}{},
		validChannels:      map[string]struct{}{},
		channelHealth:      map[string]*channelHealth{},
	}

	// init indexer, announceProcessor
	h.InitIndexers(definitions)

	return h
}

func (h *Handler) InitIndexers(definitions []domain.IndexerDefinition) {
	// Networks can be shared by multiple indexers but channels are unique
	// so let's add a new AnnounceProcessor per channel
	for _, definition := range definitions {
		if _, ok := h.definitions[definition.Identifier]; ok {
			continue
		}

		h.definitions[definition.Identifier] = &definition

		// indexers can use multiple channels, but it'h not common, but let'h handle that anyway.
		for _, channel := range definition.IRC.Channels {
			// some channels are defined in mixed case
			channel = strings.ToLower(channel)

			h.announceProcessors[channel] = announce.NewAnnounceProcessor(definition, h.filterService, h.releaseService)

			h.channelHealth[channel] = &channelHealth{
				name:       channel,
				monitoring: false,
			}

			// create map of valid channels
			h.validChannels[channel] = struct{}{}
		}

		// create map of valid announcers
		for _, announcer := range definition.IRC.Announcers {
			h.validAnnouncers[announcer] = struct{}{}
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
	//log.Debug().Msgf("server %+v", h.network)

	if h.network.Server == "" {
		return errors.New("addr not set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.ctx = ctx
	h.cancel = cancel

	dialer := net.Dialer{
		Timeout: connectTimeout,
	}

	var netConn net.Conn
	var err error

	addr := fmt.Sprintf("%v:%v", h.network.Server, h.network.Port)

	// decide to use SSL or not
	if h.network.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: true,
		}

		netConn, err = dialer.DialContext(h.ctx, "tcp", addr)
		if err != nil {
			log.Error().Err(err).Msgf("failed to dial %v", addr)
			return fmt.Errorf("failed to dial %q: %v", addr, err)
		}

		netConn = tls.Client(netConn, tlsConf)
		h.conn = netConn
	} else {
		netConn, err = dialer.DialContext(h.ctx, "tcp", addr)
		if err != nil {
			log.Error().Err(err).Msgf("failed to dial %v", addr)
			return fmt.Errorf("failed to dial %q: %v", addr, err)
		}

		h.conn = netConn
	}

	log.Info().Msgf("Connected to: %v", addr)

	config := irc.ClientConfig{
		Nick:    h.network.NickServ.Account,
		User:    h.network.NickServ.Account,
		Name:    h.network.NickServ.Account,
		Pass:    h.network.Pass,
		Handler: irc.HandlerFunc(h.handleMessage),
	}

	// Create the client
	client := irc.NewClient(h.conn, config)

	h.client = client

	// set connected since now
	h.setConnectionStatus()

	// Connect
	err = client.RunContext(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("could not connect to %v", addr)

		// reset connection status on handler and channels
		h.resetConnectionStatus()

		return err
	}

	return nil
}

func (h *Handler) handleMessage(c *irc.Client, m *irc.Message) {
	switch m.Command {
	case "001":
		// 001 is a welcome event, so we join channels there
		err := h.onConnect(h.network.Channels)
		if err != nil {
			log.Error().Msgf("error joining channels %v", err)
		}

	case "372", "375", "376":
		// Handle MOTD

	// 322 TOPIC
	// 333 UP
	// 353 @
	// 396 Displayed host
	case "366": // JOINED
		h.handleJoined(m)

	case "JOIN":
		if h.isOurNick(m.Prefix.Name) {
			log.Trace().Msgf("%v: JOIN %v", h.network.Server, m)
		}

	case "QUIT":
		if h.isOurNick(m.Prefix.Name) {
			log.Trace().Msgf("%v: QUIT %v", h.network.Server, m)
		}

	case "433":
		// TODO: handle nick in use
		log.Debug().Msgf("%v: NICK IN USE: %v", h.network.Server, m)

	case "448", "473", "475", "477":
		// TODO: handle join failed
		log.Debug().Msgf("%v: JOIN FAILED %v: %v", h.network.Server, m.Params[1], m)

	case "900": // Invite bot logged in
		log.Debug().Msgf("%v: %v", h.network.Server, m.Trailing())

	case "KICK":
		log.Debug().Msgf("%v: KICK: %v", h.network.Server, m)

	case "MODE":
		err := h.handleMode(m)
		if err != nil {
			log.Error().Err(err).Msgf("error MODE change: %v", m)
		}

	case "INVITE":
		// TODO: handle invite
		log.Debug().Msgf("%v: INVITE: %v", h.network.Server, m)

	case "PART":
		// TODO: handle parted
		if h.isOurNick(m.Prefix.Name) {
			log.Debug().Msgf("%v: PART: %v", h.network.Server, m)
		}

	case "PRIVMSG":
		err := h.onMessage(m)
		if err != nil {
			log.Error().Msgf("error on message %v", err)
		}

	case "CAP":
		log.Debug().Msgf("%v: CAP: %v", h.network.Server, m)

	case "NOTICE":
		log.Trace().Msgf("%v: %v", h.network.Server, m)

	case "PING":
		err := h.handlePing(m)
		if err != nil {
			log.Error().Stack().Err(err)
		}

		//case "372":
		//	log.Debug().Msgf("372: %v", m)
		//default:
		//	log.Trace().Msgf("%v: %v", h.network.Server, m)
	}
	return
}

func (h *Handler) isOurNick(nick string) bool {
	return h.network.NickServ.Account == nick
}

func (h *Handler) setConnectionStatus() {
	// set connected since now
	h.connectedSince = time.Now()
	h.connected = true
}

func (h *Handler) resetConnectionStatus() {
	// set connected false if we loose connection or stop
	h.connectedSince = time.Time{}
	h.connected = false

	// loop over channelHealth and reset each one
	for _, h := range h.channelHealth {
		if h != nil {
			h.resetMonitoring()
		}
	}
}

func (h *Handler) GetNetwork() *domain.IrcNetwork {
	return h.network
}

func (h *Handler) UpdateNetwork(network *domain.IrcNetwork) {
	h.network = network
}

func (h *Handler) SetNetwork(network *domain.IrcNetwork) {
	h.network = network
}

func (h *Handler) Stop() {
	h.cancel()

	if !h.isStopped() {
		close(h.stopped)
	}

	if h.conn != nil {
		h.conn.Close()
	}
}

func (h *Handler) isStopped() bool {
	select {
	case <-h.stopped:
		return true
	default:
		return false
	}
}

func (h *Handler) Restart() error {
	h.cancel()

	if !h.isStopped() {
		close(h.stopped)
	}

	if h.conn != nil {
		h.conn.Close()
	}

	time.Sleep(2 * time.Second)

	return h.Run()
}

func (h *Handler) onConnect(channels []domain.IrcChannel) error {
	identified := false

	time.Sleep(2 * time.Second)

	if h.network.NickServ.Password != "" {
		err := h.handleNickServPRIVMSG(h.network.NickServ.Account, h.network.NickServ.Password)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error nickserv: %v", h.network.Name)
			return err
		}
		identified = true
	}

	time.Sleep(3 * time.Second)

	if h.network.InviteCommand != "" {
		err := h.handleConnectCommands(h.network.InviteCommand)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error sending connect command %v to network: %v", h.network.InviteCommand, h.network.Name)
			return err
		}

		time.Sleep(2 * time.Second)
	}

	if !identified {
		for _, channel := range channels {
			err := h.HandleJoinChannel(channel.Name, channel.Password)
			if err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}
	}

	return nil
}

func (h *Handler) OnJoin(msg string) (interface{}, error) {
	return nil, nil
}

func (h *Handler) onMessage(msg *irc.Message) error {
	// parse announce
	channel := &msg.Params[0]
	announcer := &msg.Name
	message := msg.Trailing()

	// check if message is from a valid channel, if not return
	validChannel := h.isValidChannel(*channel)
	if !validChannel {
		return nil
	}

	// check if message is from announce bot, if not return
	validAnnouncer := h.isValidAnnouncer(*announcer)
	if !validAnnouncer {
		return nil
	}

	// clean message
	cleanedMsg := cleanMessage(message)
	log.Debug().Msgf("%v: %v %v: %v", h.network.Server, *channel, *announcer, cleanedMsg)

	if err := h.sendToAnnounceProcessor(*channel, cleanedMsg); err != nil {
		log.Error().Stack().Err(err).Msgf("could not queue line: %v", cleanedMsg)
		return err
	}

	return nil
}

func (h *Handler) sendToAnnounceProcessor(channel string, msg string) error {
	channel = strings.ToLower(channel)

	// check if queue exists
	queue, ok := h.announceProcessors[channel]
	if !ok {
		return fmt.Errorf("queue '%v' not found", channel)
	}

	// if it exists, add msg
	err := queue.AddLineToQueue(channel, msg)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not queue line: %v", msg)
		return err
	}

	v, ok := h.channelHealth[channel]
	if !ok {
		return nil
	}

	v.SetLastAnnounce()

	return nil
}

func (h *Handler) sendPrivMessage(msg string) error {
	msg = strings.TrimLeft(msg, "/")
	privMsg := fmt.Sprintf("PRIVMSG %h", msg)

	err := h.client.Write(privMsg)
	if err != nil {
		log.Error().Err(err).Msgf("could not send priv msg: %v", msg)
		return err
	}

	return nil
}

func (h *Handler) HandleJoinChannel(channel string, password string) error {
	// support channel password
	params := []string{channel}
	if password != "" {
		params = append(params, password)
	}

	m := irc.Message{
		Command: "JOIN",
		Params:  params,
	}

	log.Trace().Msgf("%v: sending %v", h.network.Server, m.String())

	time.Sleep(1 * time.Second)

	err := h.client.Write(m.String())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error handling join: %v", m.String())
		return err
	}

	return nil
}

func (h *Handler) HandlePartChannel(channel string) error {
	m := irc.Message{
		Command: "PART",
		Params:  []string{channel},
	}

	log.Debug().Msgf("%v: %v", h.network.Server, m.String())

	time.Sleep(1 * time.Second)

	err := h.client.Write(m.String())
	if err != nil {
		log.Error().Err(err).Msgf("error handling part: %v", m.String())
		return err
	}

	// reset monitoring status
	v, ok := h.channelHealth[channel]
	if !ok {
		return nil
	}

	v.resetMonitoring()

	// TODO remove announceProcessor

	log.Info().Msgf("Left channel '%v' on network '%h'", channel, h.network.Server)

	return nil
}

func (h *Handler) handleJoined(msg *irc.Message) {
	log.Debug().Msgf("%v: JOINED: %v", h.network.Server, msg.Params[1])

	// get channel
	channel := &msg.Params[1]

	// set monitoring on current channelHealth, or add new
	v, ok := h.channelHealth[strings.ToLower(*channel)]
	if ok {
		v.SetMonitoring()
	} else if v == nil {
		h.channelHealth[*channel] = &channelHealth{
			name:            *channel,
			monitoring:      true,
			monitoringSince: time.Now(),
		}
	}

	log.Info().Msgf("%v: Monitoring channel %v", h.network.Server, msg.Params[1])
}

func (h *Handler) handleConnectCommands(msg string) error {
	connectCommand := strings.ReplaceAll(msg, "/msg", "")
	connectCommands := strings.Split(connectCommand, ",")

	for _, command := range connectCommands {
		cmd := strings.TrimSpace(command)

		m := irc.Message{
			Command: "PRIVMSG",
			Params:  strings.Split(cmd, " "),
		}

		log.Debug().Msgf("%v: sending connect command", h.network.Server)

		err := h.client.Write(m.String())
		if err != nil {
			log.Error().Err(err).Msgf("error handling invite: %v", m.String())
			return err
		}
	}

	return nil
}

func (h *Handler) handlePRIVMSG(msg string) error {
	msg = strings.TrimLeft(msg, "/")

	m := irc.Message{
		Command: "PRIVMSG",
		Params:  []string{msg},
	}
	log.Debug().Msgf("%v: Handle privmsg: %v", h.network.Server, m.String())

	err := h.client.Write(m.String())
	if err != nil {
		log.Error().Err(err).Msgf("error handling PRIVMSG: %v", m.String())
		return err
	}

	return nil
}

func (h *Handler) handleNickServPRIVMSG(nick, password string) error {
	m := irc.Message{
		Command: "PRIVMSG",
		Params:  []string{"NickServ", "IDENTIFY", nick, password},
	}

	log.Debug().Msgf("%v: NickServ: %v", h.network.Server, m.String())

	err := h.client.Write(m.String())
	if err != nil {
		log.Error().Err(err).Msgf("error identifying with nickserv: %v", m.String())
		return err
	}

	return nil
}

func (h *Handler) HandleNickServIdentify(nick, password string) error {
	m := irc.Message{
		Command: "PRIVMSG",
		Params:  []string{"NickServ", "IDENTIFY", nick, password},
	}

	log.Debug().Msgf("%v: NickServ: %v", h.network.Server, m.String())

	err := h.client.Write(m.String())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error identifying with nickserv: %v", m.String())
		return err
	}

	return nil
}

func (h *Handler) HandleNickChange(nick string) error {
	m := irc.Message{
		Command: "NICK",
		Params:  []string{nick},
	}

	log.Debug().Msgf("%v: Nick change: %v", h.network.Server, m.String())

	err := h.client.Write(m.String())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error changing nick: %v", m.String())
		return err
	}

	return nil
}

func (h *Handler) handleMode(msg *irc.Message) error {
	log.Debug().Msgf("%v: MODE: %v %v", h.network.Server, msg.User, msg.Trailing())

	time.Sleep(2 * time.Second)

	if h.network.NickServ.Password != "" && !strings.Contains(msg.String(), h.client.CurrentNick()) || !strings.Contains(msg.String(), "+r") {
		log.Trace().Msgf("%v: MODE: Not correct permission yet: %v", h.network.Server, msg.String())
		return nil
	}

	for _, ch := range h.network.Channels {
		err := h.HandleJoinChannel(ch.Name, ch.Password)
		if err != nil {
			log.Error().Err(err).Msgf("error joining channel: %v", ch.Name)
			continue
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func (h *Handler) handlePing(msg *irc.Message) error {
	log.Trace().Msgf("%v: %v", h.network.Server, msg)

	pong := irc.Message{
		Command: "PONG",
		Params:  msg.Params,
	}

	log.Trace().Msgf("%v: %v", h.network.Server, pong.String())

	err := h.client.Write(pong.String())
	if err != nil {
		log.Error().Err(err).Msgf("error PING PONG response: %v", pong.String())
		return err
	}

	h.setLastPing()

	return nil
}

// check if announcer is one from the list in the definition
func (h *Handler) isValidAnnouncer(nick string) bool {
	_, ok := h.validAnnouncers[nick]
	if !ok {
		return false
	}

	return true
}

// check if channel is one from the list in the definition
func (h *Handler) isValidChannel(channel string) bool {
	_, ok := h.validChannels[strings.ToLower(channel)]
	if !ok {
		return false
	}

	return true
}

func (h *Handler) setLastPing() {
	h.lastPing = time.Now()
}

func (h *Handler) GetLastPing() time.Time {
	return h.lastPing
}

// irc line can contain lots of extra stuff like color so lets clean that
func cleanMessage(message string) string {
	var regexMessageClean = `\x0f|\x1f|\x02|\x03(?:[\d]{1,2}(?:,[\d]{1,2})?)?`

	rxp, err := regexp.Compile(regexMessageClean)
	if err != nil {
		log.Error().Err(err).Msgf("error compiling regex: %v", regexMessageClean)
		return ""
	}

	return rxp.ReplaceAllString(message, "")
}
