// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type Service interface {
	StartHandlers()
	StopHandlers()
	StopNetwork(id int64) error
	StopAndRemoveNetwork(id int64) error
	StopNetworkIfRunning(id int64) error
	RestartNetwork(ctx context.Context, id int64) error
	ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error)
	GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	StoreChannel(ctx context.Context, networkID int64, channel *domain.IrcChannel) error
	SendCmd(ctx context.Context, req *domain.SendIrcCmdRequest) error
}

type service struct {
	log zerolog.Logger
	sse *sse.Server

	repo                domain.IrcRepo
	releaseService      release.Service
	indexerService      indexer.Service
	notificationService notification.Service
	indexerMap          map[string]string
	handlers            map[int64]*Handler

	stopWG sync.WaitGroup
	lock   sync.RWMutex
}

const sseMaxEntries = 1000

func NewService(log logger.Logger, sse *sse.Server, repo domain.IrcRepo, releaseSvc release.Service, indexerSvc indexer.Service, notificationSvc notification.Service) Service {
	return &service{
		log:                 log.With().Str("module", "irc").Logger(),
		sse:                 sse,
		repo:                repo,
		releaseService:      releaseSvc,
		indexerService:      indexerSvc,
		notificationService: notificationSvc,
		handlers:            make(map[int64]*Handler),
	}
}

func (s *service) StartHandlers() {
	networks, err := s.repo.FindActiveNetworks(context.Background())
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
	}

	for _, network := range networks {
		if !network.Enabled {
			continue
		}

		channels, err := s.repo.ListChannels(network.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network: %s", network.Server)
		}

		for _, channel := range channels {
			// setup SSE stream per channel
			s.createSSEStream(network.ID, channel.Name)
		}

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		s.lock.Lock()
		network.Channels = channels

		// init new irc handler
		handler := NewHandler(s.log, s.sse, network, definitions, s.releaseService, s.notificationService)

		// use network.Server + nick to use multiple indexers with different nick per network
		// this allows for multiple handlers to one network
		s.handlers[network.ID] = handler
		s.lock.Unlock()

		s.log.Debug().Msgf("starting network: %s", network.Name)

		go func(network domain.IrcNetwork) {
			if err := handler.Run(); err != nil {
				s.log.Error().Err(err).Msgf("failed to start handler for network: %s", network.Name)
			}
		}(network)
	}
}

func (s *service) StopHandlers() {
	for _, handler := range s.handlers {
		s.log.Info().Msgf("stopping network: %s", handler.network.Name)
		handler.Stop()
	}

	s.log.Info().Msg("stopped all irc handlers")
}

func (s *service) startNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers already, if so start it
	if existingHandler, found := s.handlers[network.ID]; found {
		s.log.Debug().Msgf("starting network: %s", network.Name)

		if existingHandler.Stopped() {
			go func(handler *Handler) {
				if err := handler.Run(); err != nil {
					s.log.Error().Err(err).Msgf("failed to start existing handler for network: %s", handler.network.Name)
				}
			}(existingHandler)
		}
	} else {
		// if not found in handlers, lets add it and run it
		channels, err := s.repo.ListChannels(network.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network: %s", network.Server)
		}

		for _, channel := range channels {
			// setup SSE stream per channel
			s.createSSEStream(network.ID, channel.Name)
		}

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		s.lock.Lock()
		network.Channels = channels

		// init new irc handler
		handler := NewHandler(s.log, s.sse, network, definitions, s.releaseService, s.notificationService)

		s.handlers[network.ID] = handler
		s.lock.Unlock()

		s.log.Debug().Msgf("starting network: %s", network.Name)

		go func(network domain.IrcNetwork) {
			if err := handler.Run(); err != nil {
				s.log.Error().Err(err).Msgf("failed to start handler for network: %s", network.Name)
			}
		}(network)
	}

	return nil
}

func (s *service) checkIfNetworkRestartNeeded(network *domain.IrcNetwork) error {
	// look if we have the network in handlers, if so restart it
	if existingHandler, found := s.handlers[network.ID]; found {
		s.log.Debug().Msgf("irc: decide if irc network handler needs restart or updating: %s", network.Server)

		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		if !existingHandler.Stopped() {
			handler := existingHandler.GetNetwork()
			restartNeeded := false
			var fieldsChanged []string

			if handler.Server != network.Server {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "server")
			}
			if handler.Port != network.Port {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "port")
			}
			if handler.TLS != network.TLS {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "tls")
			}
			if handler.Pass != network.Pass {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "pass")
			}
			if handler.InviteCommand != network.InviteCommand {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "invite command")
			}
			if handler.UseBouncer != network.UseBouncer {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "use bouncer")
			}
			if handler.BouncerAddr != network.BouncerAddr {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "bouncer addr")
			}
			if handler.Auth.Mechanism != network.Auth.Mechanism {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "auth mechanism")
			}
			if handler.Auth.Account != network.Auth.Account {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "auth account")
			}
			if handler.Auth.Password != network.Auth.Password {
				restartNeeded = true
				fieldsChanged = append(fieldsChanged, "auth password")
			}

			if restartNeeded {
				s.log.Debug().Msgf("irc: fields %+v changed, restarting network: %s", fieldsChanged, network.Server)
				s.log.Info().Msgf("irc: restarting network: %s", network.Server)

				// we need to reinitialize with new network config
				existingHandler.UpdateNetwork(network)

				// todo reset channelHealth?

				go func() {
					if err := existingHandler.Restart(); err != nil {
						s.log.Error().Stack().Err(err).Msgf("failed to restart network: %s", existingHandler.network.Name)
					}
				}()

				// return now since the restart will read the network again
				return nil
			}

			if handler.Nick != network.Nick {
				s.log.Debug().Msg("changing nick")

				if err := existingHandler.NickChange(network.Nick); err != nil {
					s.log.Error().Err(err).Msgf("failed to change nick: %s", network.Nick)
				}
			}

			// join or leave channels
			// loop over handler channels,
			var expectedChannels = make(map[string]struct{}, 0)
			var handlerChannels = make(map[string]struct{}, 0)
			var channelsToLeave = make([]string, 0)
			var channelsToJoin = make([]domain.IrcChannel, 0)

			// create map of expected channels
			for _, channel := range network.Channels {
				expectedChannels[channel.Name] = struct{}{}
			}

			// check current channels of handler against expected
			for _, handlerChan := range handler.Channels {
				handlerChannels[handlerChan.Name] = struct{}{}

				_, ok := expectedChannels[handlerChan.Name]
				if ok {
					// 	if handler channel matches network channel next
					continue
				}

				// if not expected, leave
				channelsToLeave = append(channelsToLeave, handlerChan.Name)
			}

			// check new channels against handler to see which to join
			for _, channel := range network.Channels {
				_, ok := handlerChannels[channel.Name]
				if ok {
					continue
				}

				// if expected channel not in handler channels, add to join
				// use channel struct for extra info
				channelsToJoin = append(channelsToJoin, channel)
			}

			// leave channels
			for _, leaveChannel := range channelsToLeave {
				s.log.Debug().Msgf("%s: part channel %s", network.Server, leaveChannel)

				if err := existingHandler.PartChannel(leaveChannel); err != nil {
					s.log.Error().Err(err).Msgf("failed to leave channel: %s", leaveChannel)
				}

				// create SSE stream for new channel
				s.removeSSEStream(network.ID, leaveChannel)
			}

			// join channels
			for _, joinChannel := range channelsToJoin {
				s.log.Debug().Msgf("%s: join new channel %s", network.Server, joinChannel.Name)

				if err := existingHandler.JoinChannel(joinChannel.Name, joinChannel.Password); err != nil {
					s.log.Error().Err(err).Msgf("failed to join channel: %s", joinChannel.Name)
				}

				// create SSE stream for new channel
				s.createSSEStream(network.ID, joinChannel.Name)
			}

			// update network for handler
			// TODO move all this restart logic inside handler to let it decide what to do
			existingHandler.SetNetwork(network)

			// find indexer definitions for network and add
			definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

			existingHandler.InitIndexers(definitions)
		}
	} else {
		if err := s.startNetwork(*network); err != nil {
			s.log.Error().Err(err).Msgf("failed to start network: %s", network.Name)
		}
	}

	return nil
}

func (s *service) RestartNetwork(ctx context.Context, id int64) error {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		return err
	}

	return s.restartNetwork(*network)
}

func (s *service) restartNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers, if so restart it
	if err := s.StopNetworkIfRunning(network.ID); err != nil {
		return err
	}

	return s.startNetwork(network)
}

func (s *service) StopNetwork(id int64) error {
	if handler, found := s.handlers[id]; found {
		handler.Stop()
		s.log.Debug().Msgf("stopped network: %s", handler.network.Server)
	}

	return nil
}

func (s *service) StopAndRemoveNetwork(id int64) error {
	if handler, found := s.handlers[id]; found {
		// remove SSE streams
		for _, channel := range handler.network.Channels {
			s.removeSSEStream(handler.network.ID, channel.Name)
		}

		handler.Stop()

		// remove from handlers
		delete(s.handlers, id)
		s.log.Debug().Msgf("stopped network: %d", id)
	}

	return nil
}

func (s *service) StopNetworkIfRunning(id int64) error {
	if handler, found := s.handlers[id]; found {
		handler.Stop()
		s.log.Debug().Msgf("stopped network: %s", handler.network.Server)
	}

	return nil
}

func (s *service) GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error) {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to get network: %d", id)
		return nil, err
	}

	channels, err := s.repo.ListChannels(network.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to list channels for network: %s", network.Server)
		return nil, err
	}
	network.Channels = append(network.Channels, channels...)

	return network, nil
}

func (s *service) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	networks, err := s.repo.ListNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
		return nil, err
	}

	ret := make([]domain.IrcNetwork, 0)

	for _, n := range networks {
		channels, err := s.repo.ListChannels(n.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network: %s", n.Server)
			return nil, err
		}
		n.Channels = append(n.Channels, channels...)

		ret = append(ret, n)
	}

	return ret, nil
}

func (s *service) GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error) {
	networks, err := s.repo.ListNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
		return nil, err
	}

	ret := make([]domain.IrcNetworkWithHealth, 0)

	for _, n := range networks {
		netw := domain.IrcNetworkWithHealth{
			ID:               n.ID,
			Name:             n.Name,
			Enabled:          n.Enabled,
			Server:           n.Server,
			Port:             n.Port,
			TLS:              n.TLS,
			Pass:             n.Pass,
			Nick:             n.Nick,
			Auth:             n.Auth,
			InviteCommand:    n.InviteCommand,
			BouncerAddr:      n.BouncerAddr,
			UseBouncer:       n.UseBouncer,
			Connected:        false,
			Channels:         []domain.ChannelWithHealth{},
			ConnectionErrors: []string{},
		}

		handler, ok := s.handlers[n.ID]
		if ok {
			handler.ReportStatus(&netw)
		}

		channels, err := s.repo.ListChannels(n.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network: %s", n.Server)
			return nil, err
		}

		// combine from repo and handler
		for _, channel := range channels {
			ch := domain.ChannelWithHealth{
				ID:       channel.ID,
				Enabled:  channel.Enabled,
				Name:     channel.Name,
				Password: channel.Password,
				Detached: channel.Detached,
				//Monitoring:      false,
				//MonitoringSince: time.Time{},
				//LastAnnounce:    time.Time{},
			}

			// only check if we have a handler
			if handler != nil {
				name := strings.ToLower(channel.Name)

				handler.m.RLock()
				chan1, ok := handler.channelHealth[name]
				if ok {
					chan1.m.RLock()
					ch.Monitoring = chan1.monitoring
					ch.MonitoringSince = chan1.monitoringSince
					ch.LastAnnounce = chan1.lastAnnounce

					chan1.m.RUnlock()
				}
				handler.m.RUnlock()
			}

			netw.Channels = append(netw.Channels, ch)
		}

		ret = append(ret, netw)
	}

	return ret, nil
}

func (s *service) DeleteNetwork(ctx context.Context, id int64) error {
	network, err := s.GetNetworkByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find network before delete: %d", id)
		return err
	}

	s.log.Debug().Msgf("delete network: %d %s", id, network.Name)

	// Remove network and handler
	if err = s.StopAndRemoveNetwork(network.ID); err != nil {
		s.log.Error().Err(err).Msgf("could not stop and delete network: %s", network.Name)
		return err
	}

	if err = s.repo.DeleteNetwork(ctx, id); err != nil {
		s.log.Error().Err(err).Msgf("could not delete network: %s", network.Name)
		return err
	}

	return nil
}

func (s *service) UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	if network.Channels != nil {
		if err := s.repo.StoreNetworkChannels(ctx, network.ID, network.Channels); err != nil {
			return err
		}
	}

	if err := s.repo.UpdateNetwork(ctx, network); err != nil {
		return err
	}
	s.log.Debug().Msgf("irc.service: update network: %s", network.Name)

	// stop or start network
	// TODO get current state to see if enabled or not?
	if network.Enabled {
		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		err := s.checkIfNetworkRestartNeeded(network)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not restart network: %s", network.Name)
			return errors.New("could not restart network: %s", network.Name)
		}

	} else {
		// take into account multiple channels per network
		err := s.StopAndRemoveNetwork(network.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not stop network: %s", network.Name)
			return errors.New("could not stop network: %s", network.Name)
		}
	}

	return nil
}

func (s *service) StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	existingNetwork, err := s.repo.CheckExistingNetwork(ctx, network)
	if err != nil {
		s.log.Error().Err(err).Msg("could not check for existing network")
		return err
	}

	if existingNetwork == nil {
		if err := s.repo.StoreNetwork(ctx, network); err != nil {
			return err
		}
		s.log.Debug().Msgf("store network: %+v", network)

		if network.Channels != nil {
			for _, channel := range network.Channels {
				if err := s.repo.StoreChannel(ctx, network.ID, &channel); err != nil {
					s.log.Error().Err(err).Msg("irc.storeChannel: error executing query")
					return errors.Wrap(err, "error storing channel on network")
				}
			}
		}

		return nil
	}

	// get channels for existing network
	existingChannels, err := s.repo.ListChannels(existingNetwork.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to list channels for network: %s", existingNetwork.Server)
	}
	existingNetwork.Channels = existingChannels

	if network.Channels != nil {
		for _, channel := range network.Channels {
			// add channels. Make sure it doesn't delete before
			if err := s.repo.StoreChannel(ctx, existingNetwork.ID, &channel); err != nil {
				return err
			}
		}

		// append channels to existing network
		existingNetwork.Channels = append(existingNetwork.Channels, network.Channels...)
	}

	// append invite command for existing network
	if network.InviteCommand != "" {
		existingNetwork.InviteCommand = strings.Join([]string{existingNetwork.InviteCommand, network.InviteCommand}, ",")
		if err := s.repo.UpdateInviteCommand(existingNetwork.ID, existingNetwork.InviteCommand); err != nil {
			return err
		}
	}

	if existingNetwork.Enabled {
		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave

		if err := s.checkIfNetworkRestartNeeded(existingNetwork); err != nil {
			s.log.Error().Err(err).Msgf("could not restart network: %s", existingNetwork.Name)
			return errors.New("could not restart network: %s", existingNetwork.Name)
		}
	}

	return nil
}

func (s *service) StoreChannel(ctx context.Context, networkID int64, channel *domain.IrcChannel) error {
	if err := s.repo.StoreChannel(ctx, networkID, channel); err != nil {
		return err
	}

	return nil
}

func (s *service) SendCmd(ctx context.Context, req *domain.SendIrcCmdRequest) error {
	if handler, found := s.handlers[req.NetworkId]; found {
		if err := handler.SendMsg(req.Channel, req.Message); err != nil {
			s.log.Error().Err(err).Msgf("could not send message to channel: %s %s", req.Channel, req.Message)
		}
	}

	return nil
}

func (s *service) createSSEStream(networkId int64, channel string) {
	key := genSSEKey(networkId, channel)

	s.sse.CreateStreamWithOpts(key, sse.StreamOpts{
		MaxEntries: sseMaxEntries,
		AutoReplay: true,
	})
}

func (s *service) removeSSEStream(networkId int64, channel string) {
	key := genSSEKey(networkId, channel)

	s.sse.RemoveStream(key)
}

func genSSEKey(networkId int64, channel string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d%s", networkId, strings.ToLower(channel))))
}
