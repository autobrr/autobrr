package irc

import (
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog/log"
)

var (
	connectTimeout = 15 * time.Second
)

type channelHealth struct {
	m sync.RWMutex

	name            string
	monitoring      bool
	monitoringSince time.Time
	lastAnnounce    time.Time
}

// SetLastAnnounce set last announce to now
func (h *channelHealth) SetLastAnnounce() {
	h.m.Lock()
	h.lastAnnounce = time.Now()
	h.m.Unlock()
}

// SetMonitoring set monitoring and time
func (h *channelHealth) SetMonitoring() {
	h.m.Lock()
	h.monitoring = true
	h.monitoringSince = time.Now()
	h.m.Unlock()
}

// resetMonitoring remove monitoring and time
func (h *channelHealth) resetMonitoring() {
	h.m.Lock()
	h.monitoring = false
	h.monitoringSince = time.Time{}
	h.m.Unlock()
}

type Handler struct {
	network            *domain.IrcNetwork
	releaseSvc         release.Service
	announceProcessors map[string]announce.Processor
	definitions        map[string]*domain.IndexerDefinition

	client *ircevent.Connection
	m      sync.RWMutex

	lastPing       time.Time
	connected      bool
	connectedSince time.Time
	// tODO disconnectedTime

	validAnnouncers map[string]struct{}
	validChannels   map[string]struct{}
	channelHealth   map[string]*channelHealth
}

func NewHandler(network domain.IrcNetwork, definitions []*domain.IndexerDefinition, releaseSvc release.Service) *Handler {
	h := &Handler{
		client:             nil,
		network:            &network,
		releaseSvc:         releaseSvc,
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

			h.announceProcessors[channel] = announce.NewAnnounceProcessor(h.releaseSvc, definition)

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
	addr := fmt.Sprintf("%v:%d", h.network.Server, h.network.Port)

	h.client = &ircevent.Connection{
		Nick:          h.network.NickServ.Account,
		User:          h.network.NickServ.Account,
		RealName:      h.network.NickServ.Account,
		Password:      h.network.Pass,
		Server:        addr,
		KeepAlive:     4 * time.Minute,
		Timeout:       1 * time.Minute,
		ReconnectFreq: 15 * time.Second,
		Version:       "autobrr",
		QuitMessage:   "bye from autobrr",
		Debug:         true,
		Log:           logger.StdLeveledLogger,
	}

	if h.network.TLS {
		h.client.UseTLS = true
		h.client.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	h.client.AddConnectCallback(h.onConnect)
	h.client.AddCallback("MODE", h.handleMode)
	h.client.AddCallback("INVITE", h.handleInvite)
	h.client.AddCallback("366", h.handleJoined)
	h.client.AddCallback("PART", h.handlePart)
	h.client.AddCallback("PRIVMSG", h.onMessage)

	if err := h.client.Connect(); err != nil {
		log.Error().Stack().Err(err).Msgf("%v: connect error", h.network.Server)

		// reset connection status on handler and channels
		h.resetConnectionStatus()

		//return err
	}

	// set connected since now
	h.setConnectionStatus()

	h.client.Loop()

	return nil
}

func (h *Handler) isOurNick(nick string) bool {
	return h.network.NickServ.Account == nick
}

func (h *Handler) setConnectionStatus() {
	h.m.Lock()
	// set connected since now
	h.connectedSince = time.Now()
	h.connected = true
	h.m.Unlock()
}

func (h *Handler) resetConnectionStatus() {
	h.m.Lock()
	// set connected false if we loose connection or stop
	h.connectedSince = time.Time{}
	h.connected = false

	// loop over channelHealth and reset each one
	for _, h := range h.channelHealth {
		if h != nil {
			h.resetMonitoring()
		}
	}

	h.m.Unlock()
}

func (h *Handler) GetNetwork() *domain.IrcNetwork {
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

func (h *Handler) Stop() {
	log.Debug().Msgf("%v: Disconnecting...", h.network.Server)
	h.client.Quit()
}

func (h *Handler) Restart() error {
	log.Debug().Msgf("%v: Restarting network...", h.network.Server)

	h.client.Quit()

	time.Sleep(4 * time.Second)

	return h.Run()
}

func (h *Handler) onConnect(m ircmsg.Message) {
	identified := false

	time.Sleep(4 * time.Second)

	if h.network.NickServ.Password != "" {
		err := h.HandleNickServIdentify(h.network.NickServ.Password)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error nickserv: %v", h.network.Name)
			return
		}
		identified = true
	}

	time.Sleep(4 * time.Second)

	if h.network.InviteCommand != "" {
		err := h.handleConnectCommands(h.network.InviteCommand)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("error sending connect command %v to network: %v", h.network.InviteCommand, h.network.Name)
			return
		}

		time.Sleep(1 * time.Second)
	}

	if !identified {
		for _, channel := range h.network.Channels {
			err := h.HandleJoinChannel(channel.Name, channel.Password)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("error joining channels %v", err)
				return
			}
		}
	}

}

func (h *Handler) onMessage(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}
	// parse announce
	announcer := msg.Nick()
	channel := msg.Params[0]
	message := msg.Params[1]

	// check if message is from a valid channel, if not return
	validChannel := h.isValidChannel(channel)
	if !validChannel {
		return
	}

	// check if message is from announce bot, if not return
	validAnnouncer := h.isValidAnnouncer(announcer)
	if !validAnnouncer {
		return
	}

	// clean message
	cleanedMsg := cleanMessage(message)
	log.Debug().Msgf("%v: %v %v: %v", h.network.Server, channel, announcer, cleanedMsg)

	if err := h.sendToAnnounceProcessor(channel, cleanedMsg); err != nil {
		log.Error().Stack().Err(err).Msgf("could not queue line: %v", cleanedMsg)
		return
	}

	return
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

func (h *Handler) HandleJoinChannel(channel string, password string) error {
	m := ircmsg.Message{
		Command: "JOIN",
		Params:  []string{channel},
	}

	// support channel password
	if password != "" {
		m.Params = []string{channel, password}
	}

	log.Debug().Msgf("%v: sending JOIN command %v", h.network.Server, strings.Join(m.Params, " "))

	err := h.client.SendIRCMessage(m)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error handling join: %v", channel)
		return err
	}

	return nil
}

func (h *Handler) handlePart(msg ircmsg.Message) {
	if !h.isOurNick(msg.Nick()) {
		log.Debug().Msgf("%v: MODE OTHER USER: %+v", h.network.Server, msg)
		return
	}

	channel := msg.Params[0]

	log.Debug().Msgf("%v: PART channel %v", h.network.Server, channel)

	err := h.client.Part(channel)
	if err != nil {
		log.Error().Err(err).Msgf("error handling part: %v", channel)
		return
	}

	// reset monitoring status
	v, ok := h.channelHealth[channel]
	if !ok {
		return
	}

	v.resetMonitoring()

	// TODO remove announceProcessor

	log.Info().Msgf("%v: Left channel '%v'", h.network.Server, channel)

	return
}

func (h *Handler) HandlePartChannel(channel string) error {
	log.Debug().Msgf("%v: PART channel %v", h.network.Server, channel)

	err := h.client.Part(channel)
	if err != nil {
		log.Error().Err(err).Msgf("error handling part: %v", channel)
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

func (h *Handler) handleJoined(msg ircmsg.Message) {
	if !h.isOurNick(msg.Params[0]) {
		log.Debug().Msgf("%v: OTHER USER JOINED: %+v", h.network.Server, msg)
		return
	}

	// get channel
	channel := msg.Params[1]

	log.Debug().Msgf("%v: JOINED: %v", h.network.Server, msg.Params[1])

	// set monitoring on current channelHealth, or add new
	v, ok := h.channelHealth[strings.ToLower(channel)]
	if ok {
		v.SetMonitoring()
	} else if v == nil {
		h.AddChannelHealth(channel)
	}

	valid := h.isValidChannel(channel)
	if valid {
		log.Info().Msgf("%v: Monitoring channel %v", h.network.Server, msg.Params[1])
		return
	}
}

func (h *Handler) handleConnectCommands(msg string) error {
	connectCommand := strings.ReplaceAll(msg, "/msg", "")
	connectCommands := strings.Split(connectCommand, ",")

	for _, command := range connectCommands {
		cmd := strings.TrimSpace(command)

		m := ircmsg.Message{
			Command: "PRIVMSG",
			Params:  strings.Split(cmd, " "),
		}

		log.Debug().Msgf("%v: sending connect command", h.network.Server)

		err := h.client.SendIRCMessage(m)
		if err != nil {
			log.Error().Err(err).Msgf("error handling invite: %v", m)
			return err
		}
	}

	return nil
}

func (h *Handler) handleInvite(msg ircmsg.Message) {
	if len(msg.Params) < 2 {
		return
	}

	// get channel
	channel := msg.Params[1]

	log.Debug().Msgf("%v: INVITE from %v, joining %v", h.network.Server, msg.Nick(), channel)

	err := h.client.Join(channel)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error handling join: %v", channel)
		return
	}

	return
}

func (h *Handler) HandleNickServIdentify(password string) error {
	m := ircmsg.Message{
		Command: "PRIVMSG",
		Params:  []string{"NickServ", "IDENTIFY", password},
	}

	log.Debug().Msgf("%v: NickServ: %v", h.network.Server, m)

	err := h.client.SendIRCMessage(m)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error identifying with nickserv: %v", m)
		return err
	}

	return nil
}

func (h *Handler) HandleNickChange(nick string) error {
	log.Debug().Msgf("%v: Nick change: %v", h.network.Server, nick)

	h.client.SetNick(nick)

	return nil
}

func (h *Handler) handleMode(msg ircmsg.Message) {
	log.Debug().Msgf("%v: MODE: %+v", h.network.Server, msg)

	if !h.isOurNick(msg.Params[0]) {
		log.Debug().Msgf("%v: MODE OTHER USER: %+v", h.network.Server, msg)
		return
	}

	time.Sleep(5 * time.Second)

	if h.network.NickServ.Password != "" && !strings.Contains(msg.Params[0], h.client.Nick) || !strings.Contains(msg.Params[1], "+r") {
		log.Trace().Msgf("%v: MODE: Not correct permission yet: %v", h.network.Server, msg.Params)
		return
	}

	for _, ch := range h.network.Channels {
		err := h.HandleJoinChannel(ch.Name, ch.Password)
		if err != nil {
			log.Error().Err(err).Msgf("error joining channel: %v", ch.Name)
			continue
		}

		time.Sleep(1 * time.Second)
	}

	return
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
