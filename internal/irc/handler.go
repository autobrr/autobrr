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

	"github.com/rs/zerolog/log"
	"gopkg.in/irc.v3"
)

var (
	connectTimeout = 15 * time.Second
)

type Handler struct {
	network         *domain.IrcNetwork
	announceService announce.Service

	conn    net.Conn
	ctx     context.Context
	stopped chan struct{}
	cancel  context.CancelFunc
}

func NewHandler(network domain.IrcNetwork, announceService announce.Service) *Handler {
	return &Handler{
		conn:            nil,
		ctx:             nil,
		stopped:         make(chan struct{}),
		network:         &network,
		announceService: announceService,
	}
}

func (s *Handler) Run() error {
	//log.Debug().Msgf("server %+v", s.network)

	if s.network.Addr == "" {
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

	addr := s.network.Addr

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
		Nick: s.network.Nick,
		User: s.network.Nick,
		Name: s.network.Nick,
		Pass: s.network.Pass,
		Handler: irc.HandlerFunc(func(c *irc.Client, m *irc.Message) {
			switch m.Command {
			case "001":
				// 001 is a welcome event, so we join channels there
				err := s.onConnect(c, s.network.Channels)
				if err != nil {
					log.Error().Msgf("error joining channels %v", err)
				}

			case "366":
				// TODO: handle joined
				log.Debug().Msgf("JOINED: %v", m)

			case "433":
				// TODO: handle nick in use
				log.Debug().Msgf("NICK IN USE: %v", m)

			case "448", "475", "477":
				// TODO: handle join failed
				log.Debug().Msgf("JOIN FAILED: %v", m)

			case "KICK":
				log.Debug().Msgf("KICK: %v", m)

			case "MODE":
				// TODO: handle mode change
				log.Debug().Msgf("MODE CHANGE: %v", m)

			case "INVITE":
				// TODO: handle invite
				log.Debug().Msgf("INVITE: %v", m)

			case "PART":
				// TODO: handle parted
				log.Debug().Msgf("PART: %v", m)

			case "PRIVMSG":
				err := s.onMessage(m)
				if err != nil {
					log.Error().Msgf("error on message %v", err)
				}
			}
		}),
	}

	// Create the client
	client := irc.NewClient(s.conn, config)

	// Connect
	err = client.RunContext(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("could not connect to %v", addr)
		return err
	}

	return nil
}

func (s *Handler) GetNetwork() *domain.IrcNetwork {
	return s.network
}

func (s *Handler) Stop() {
	s.cancel()

	//if !s.isStopped() {
	//	close(s.stopped)
	//}

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

func (s *Handler) onConnect(client *irc.Client, channels []domain.IrcChannel) error {
	// TODO check commands like nickserv before joining

	for _, command := range s.network.ConnectCommands {
		cmd := strings.TrimLeft(command, "/")

		log.Info().Msgf("send connect command: %v to network: %s", cmd, s.network.Name)

		err := client.Write(cmd)
		if err != nil {
			log.Error().Err(err).Msgf("error sending connect command %v to network: %v", command, s.network.Name)
			continue
			//return err
		}

		time.Sleep(1 * time.Second)
	}

	for _, ch := range channels {
		myChan := fmt.Sprintf("JOIN %s", ch.Name)

		// handle channel password
		if ch.Password != "" {
			myChan = fmt.Sprintf("JOIN %s %s", ch.Name, ch.Password)
		}

		err := client.Write(myChan)
		if err != nil {
			log.Error().Err(err).Msgf("error joining channel: %v", ch.Name)
			continue
			//return err
		}

		log.Info().Msgf("Monitoring channel %s", ch.Name)

		time.Sleep(1 * time.Second)
	}

	return nil
}

func (s *Handler) OnJoin(msg string) (interface{}, error) {
	return nil, nil
}

func (s *Handler) onMessage(msg *irc.Message) error {
	log.Debug().Msgf("msg: %v", msg)

	// parse announce
	channel := &msg.Params[0]
	announcer := &msg.Name
	message := msg.Trailing()
	// TODO add network

	// add correlationID and tracing

	announceID := fmt.Sprintf("%v:%v:%v", s.network.Addr, *channel, *announcer)

	// clean message
	cleanedMsg := cleanMessage(message)

	go func() {
		err := s.announceService.Parse(announceID, cleanedMsg)
		if err != nil {
			log.Error().Err(err).Msgf("could not parse line: %v", cleanedMsg)
		}
	}()

	return nil
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
