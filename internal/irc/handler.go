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
	definitions        []domain.IndexerDefinition

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
		definitions:        definitions,
		announceProcessors: map[string]announce.Processor{},
		validAnnouncers:    map[string]struct{}{},
		validChannels:      map[string]struct{}{},
		channelHealth:      map[string]*channelHealth{},
	}

	// Networks can be shared by multiple indexers but channels are unique
	// so let's add a new AnnounceProcessor per channel
	for _, definition := range definitions {
		// indexers can use multiple channels, but it's not common, but let's handle that anyway.
		for _, channel := range definition.IRC.Channels {
			// some channels are defined in mixed case
			channel = strings.ToLower(channel)

			h.announceProcessors[channel] = announce.NewAnnounceProcessor(definition, filterService, releaseService)

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

	return h
}

func (s *Handler) Run() error {
	//log.Debug().Msgf("server %+v", s.network)

	if s.network.Server == "" {
		return errors.New("addr not set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	dialer := net.Dialer{
		Timeout: connectTimeout,
	}

	var netConn net.Conn
	var err error

	addr := fmt.Sprintf("%v:%v", s.network.Server, s.network.Port)

	// decide to use SSL or not
	if s.network.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: true,
		}

		netConn, err = dialer.DialContext(s.ctx, "tcp", addr)
		if err != nil {
			log.Error().Err(err).Msgf("failed to dial %v", addr)
			return fmt.Errorf("failed to dial %q: %v", addr, err)
		}

		netConn = tls.Client(netConn, tlsConf)
		s.conn = netConn
	} else {
		netConn, err = dialer.DialContext(s.ctx, "tcp", addr)
		if err != nil {
			log.Error().Err(err).Msgf("failed to dial %v", addr)
			return fmt.Errorf("failed to dial %q: %v", addr, err)
		}

		s.conn = netConn
	}

	log.Info().Msgf("Connected to: %v", addr)

	config := irc.ClientConfig{
		Nick: s.network.NickServ.Account,
		User: s.network.NickServ.Account,
		Name: s.network.NickServ.Account,
		Pass: s.network.Pass,
		Handler: irc.HandlerFunc(func(c *irc.Client, m *irc.Message) {
			switch m.Command {
			case "001":
				// 001 is a welcome event, so we join channels there
				err := s.onConnect(s.network.Channels)
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
				s.handleJoined(m)

			case "JOIN":
				log.Trace().Msgf("%v: JOIN %v", s.network.Server, m)

			case "433":
				// TODO: handle nick in use
				log.Debug().Msgf("%v: NICK IN USE: %v", s.network.Server, m)

			case "448", "473", "475", "477":
				// TODO: handle join failed
				log.Debug().Msgf("%v: JOIN FAILED %v: %v", s.network.Server, m.Params[1], m)

			case "900": // Invite bot logged in
				log.Debug().Msgf("%v: %v", s.network.Server, m.Trailing())

			case "KICK":
				log.Debug().Msgf("%v: KICK: %v", s.network.Server, m)

			case "MODE":
				err := s.handleMode(m)
				if err != nil {
					log.Error().Err(err).Msgf("error MODE change: %v", m)
				}

			case "INVITE":
				// TODO: handle invite
				log.Debug().Msgf("%v: INVITE: %v", s.network.Server, m)

			case "PART":
				// TODO: handle parted
				log.Debug().Msgf("%v: PART: %v", s.network.Server, m)

			case "PRIVMSG":
				err := s.onMessage(m)
				if err != nil {
					log.Error().Msgf("error on message %v", err)
				}

			case "CAP":
				log.Debug().Msgf("%v: CAP: %v", s.network.Server, m)

			case "NOTICE":
				log.Trace().Msgf("%v: %v", s.network.Server, m)

			case "PING":
				err := s.handlePing(m)
				if err != nil {
					log.Error().Stack().Err(err)
				}

			//case "372":
			//	log.Debug().Msgf("372: %v", m)
			default:
				log.Trace().Msgf("%v: %v", s.network.Server, m)
			}
		}),
	}

	// Create the client
	client := irc.NewClient(s.conn, config)

	s.client = client

	// set connected since now
	s.setConnectionStatus()

	// Connect
	err = client.RunContext(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("could not connect to %v", addr)

		// reset connection status on handler and channels
		s.resetConnectionStatus()

		return err
	}

	return nil
}

func (s *Handler) setConnectionStatus() {
	// set connected since now
	s.connectedSince = time.Now()
	s.connected = true
}

func (s *Handler) resetConnectionStatus() {
	// set connected false if we loose connection or stop
	s.connectedSince = time.Time{}
	s.connected = false

	// loop over channelHealth and reset each one
	for _, h := range s.channelHealth {
		if h != nil {
			h.resetMonitoring()
		}
	}
}

func (s *Handler) GetNetwork() *domain.IrcNetwork {
	return s.network
}

func (s *Handler) UpdateNetwork(network *domain.IrcNetwork) {
	s.network = network
}

func (s *Handler) Stop() {
	s.cancel()

	if !s.isStopped() {
		close(s.stopped)
	}

	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *Handler) isStopped() bool {
	select {
	case <-s.stopped:
		return true
	default:
		return false
	}
}

func (s *Handler) Restart() error {
	s.cancel()

	if !s.isStopped() {
		close(s.stopped)
	}

	if s.conn != nil {
		s.conn.Close()
	}

	time.Sleep(2 * time.Second)

	return s.Run()
}

func (s *Handler) onConnect(channels []domain.IrcChannel) error {
	identified := false

	time.Sleep(2 * time.Second)

	if s.network.NickServ.Password != "" {
		err := s.handleNickServPRIVMSG(s.network.NickServ.Account, s.network.NickServ.Password)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error nickserv: %v", s.network.Name)
			return err
		}
		identified = true
	}

	time.Sleep(3 * time.Second)

	if s.network.InviteCommand != "" {

		err := s.handleInvitePRIVMSG(s.network.InviteCommand)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error sending connect command %v to network: %v", s.network.InviteCommand, s.network.Name)
			return err
		}

		time.Sleep(2 * time.Second)
	}

	if !identified {
		for _, channel := range channels {
			err := s.HandleJoinChannel(channel.Name, channel.Password)
			if err != nil {
				log.Error().Stack().Err(err)
				return err
			}
		}
	}

	return nil
}

func (s *Handler) OnJoin(msg string) (interface{}, error) {
	return nil, nil
}

func (s *Handler) onMessage(msg *irc.Message) error {
	// parse announce
	channel := &msg.Params[0]
	announcer := &msg.Name
	message := msg.Trailing()

	// check if message is from a valid channel, if not return
	validChannel := s.isValidChannel(*channel)
	if !validChannel {
		return nil
	}

	// check if message is from announce bot, if not return
	validAnnouncer := s.isValidAnnouncer(*announcer)
	if !validAnnouncer {
		return nil
	}

	// clean message
	cleanedMsg := cleanMessage(message)
	log.Debug().Msgf("%v: %v %v: %v", s.network.Server, *channel, *announcer, cleanedMsg)

	if err := s.sendToAnnounceProcessor(*channel, cleanedMsg); err != nil {
		log.Error().Stack().Err(err).Msgf("could not queue line: %v", cleanedMsg)
		return err
	}

	return nil
}

func (s *Handler) sendToAnnounceProcessor(channel string, msg string) error {
	channel = strings.ToLower(channel)

	// check if queue exists
	queue, ok := s.announceProcessors[channel]
	if !ok {
		return fmt.Errorf("queue '%v' not found", channel)
	}

	// if it exists, add msg
	err := queue.AddLineToQueue(channel, msg)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("could not queue line: %v", msg)
		return err
	}

	v, ok := s.channelHealth[channel]
	if !ok {
		return nil
	}

	v.SetLastAnnounce()

	return nil
}

func (s *Handler) sendPrivMessage(msg string) error {
	msg = strings.TrimLeft(msg, "/")
	privMsg := fmt.Sprintf("PRIVMSG %s", msg)

	err := s.client.Write(privMsg)
	if err != nil {
		log.Error().Err(err).Msgf("could not send priv msg: %v", msg)
		return err
	}

	return nil
}

func (s *Handler) HandleJoinChannel(channel string, password string) error {
	// support channel password
	params := []string{channel}
	if password != "" {
		params = append(params, password)
	}

	m := irc.Message{
		Command: "JOIN",
		Params:  params,
	}

	log.Trace().Msgf("%v: sending %v", s.network.Server, m.String())

	time.Sleep(1 * time.Second)

	err := s.client.Write(m.String())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error handling join: %v", m.String())
		return err
	}

	return nil
}

func (s *Handler) HandlePartChannel(channel string) error {
	m := irc.Message{
		Command: "PART",
		Params:  []string{channel},
	}

	log.Debug().Msgf("%v: %v", s.network.Server, m.String())

	time.Sleep(1 * time.Second)

	err := s.client.Write(m.String())
	if err != nil {
		log.Error().Err(err).Msgf("error handling part: %v", m.String())
		return err
	}

	log.Info().Msgf("Left channel '%v' on network '%s'", channel, s.network.Server)

	return nil
}

func (s *Handler) handleJoined(msg *irc.Message) {
	log.Debug().Msgf("%v: JOINED: %v", s.network.Server, msg.String())

	// get channel
	channel := &msg.Params[1]

	// only set values if channel is found in map
	v, ok := s.channelHealth[strings.ToLower(*channel)]
	if ok {
		v.SetMonitoring()
	}

	log.Info().Msgf("%v: Monitoring channel %s", s.network.Server, msg.Params[1])
}

func (s *Handler) handleInvitePRIVMSG(msg string) error {
	msg = strings.TrimPrefix(msg, "/msg")
	split := strings.Split(msg, " ")

	m := irc.Message{
		Command: "PRIVMSG",
		Params:  split,
	}

	log.Info().Msgf("%v: Invite command: %v", s.network.Server, m.String())

	err := s.client.Write(m.String())
	if err != nil {
		log.Error().Err(err).Msgf("error handling invite: %v", m.String())
		return err
	}

	return nil
}

func (s *Handler) handlePRIVMSG(msg string) error {
	msg = strings.TrimLeft(msg, "/")

	m := irc.Message{
		Command: "PRIVMSG",
		Params:  []string{msg},
	}
	log.Debug().Msgf("%v: Handle privmsg: %v", s.network.Server, m.String())

	err := s.client.Write(m.String())
	if err != nil {
		log.Error().Err(err).Msgf("error handling PRIVMSG: %v", m.String())
		return err
	}

	return nil
}

func (s *Handler) handleNickServPRIVMSG(nick, password string) error {
	m := irc.Message{
		Command: "PRIVMSG",
		Params:  []string{"NickServ", "IDENTIFY", nick, password},
	}

	log.Debug().Msgf("%v: NickServ: %v", s.network.Server, m.String())

	err := s.client.Write(m.String())
	if err != nil {
		log.Error().Err(err).Msgf("error identifying with nickserv: %v", m.String())
		return err
	}

	return nil
}

func (s *Handler) HandleNickServIdentify(nick, password string) error {
	m := irc.Message{
		Command: "PRIVMSG",
		Params:  []string{"NickServ", "IDENTIFY", nick, password},
	}

	log.Debug().Msgf("%v: NickServ: %v", s.network.Server, m.String())

	err := s.client.Write(m.String())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error identifying with nickserv: %v", m.String())
		return err
	}

	return nil
}

func (s *Handler) HandleNickChange(nick string) error {
	m := irc.Message{
		Command: "NICK",
		Params:  []string{nick},
	}

	log.Debug().Msgf("%v: Nick change: %v", s.network.Server, m.String())

	err := s.client.Write(m.String())
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error changing nick: %v", m.String())
		return err
	}

	return nil
}

func (s *Handler) handleMode(msg *irc.Message) error {
	log.Debug().Msgf("%v: MODE: %v %v", s.network.Server, msg.User, msg.Trailing())

	time.Sleep(2 * time.Second)

	if s.network.NickServ.Password != "" && !strings.Contains(msg.String(), s.client.CurrentNick()) || !strings.Contains(msg.String(), "+r") {
		log.Trace().Msgf("%v: MODE: Not correct permission yet: %v", s.network.Server, msg.String())
		return nil
	}

	for _, ch := range s.network.Channels {
		err := s.HandleJoinChannel(ch.Name, ch.Password)
		if err != nil {
			log.Error().Err(err).Msgf("error joining channel: %v", ch.Name)
			continue
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func (s *Handler) handlePing(msg *irc.Message) error {
	log.Trace().Msgf("%v: %v", s.network.Server, msg)

	pong := irc.Message{
		Command: "PONG",
		Params:  msg.Params,
	}

	log.Trace().Msgf("%v: %v", s.network.Server, pong.String())

	err := s.client.Write(pong.String())
	if err != nil {
		log.Error().Err(err).Msgf("error PING PONG response: %v", pong.String())
		return err
	}

	s.setLastPing()

	return nil
}

// check if announcer is one from the list in the definition
func (s *Handler) isValidAnnouncer(nick string) bool {
	_, ok := s.validAnnouncers[nick]
	if !ok {
		return false
	}

	return true
}

// check if channel is one from the list in the definition
func (s *Handler) isValidChannel(channel string) bool {
	_, ok := s.validChannels[strings.ToLower(channel)]
	if !ok {
		return false
	}

	return true
}

func (s *Handler) setLastPing() {
	s.lastPing = time.Now()
}

func (s *Handler) GetLastPing() time.Time {
	return s.lastPing
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
