// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type Service interface {
	StartHandlers()
	StopHandlers()
	StopNetwork(key handlerKey) error
	RestartNetwork(ctx context.Context, id int64) error
	ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error)
	GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	StoreChannel(networkID int64, channel *domain.IrcChannel) error
}

type service struct {
	stopWG sync.WaitGroup
	lock   sync.RWMutex

	log                 zerolog.Logger
	repo                domain.IrcRepo
	releaseService      release.Service
	indexerService      indexer.Service
	notificationService notification.Service
	indexerMap          map[string]string
	handlers            map[handlerKey]*Handler
}

func NewService(log logger.Logger, repo domain.IrcRepo, releaseSvc release.Service, indexerSvc indexer.Service, notificationSvc notification.Service) Service {
	return &service{
		log:                 log.With().Str("module", "irc").Logger(),
		repo:                repo,
		releaseService:      releaseSvc,
		indexerService:      indexerSvc,
		notificationService: notificationSvc,
		handlers:            make(map[handlerKey]*Handler),
	}
}

type handlerKey struct {
	server string
	nick   string
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

		// check if already in handlers
		//v, ok := s.handlers[network.Name]

		s.lock.Lock()
		channels, err := s.repo.ListChannels(network.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network %q", network.Server)
		}
		network.Channels = channels

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		// init new irc handler
		handler := NewHandler(s.log, network, definitions, s.releaseService, s.notificationService)

		// use network.Server + nick to use multiple indexers with different nick per network
		// this allows for multiple handlers to one network
		s.handlers[handlerKey{network.Server, network.Nick}] = handler
		s.lock.Unlock()

		s.log.Debug().Msgf("starting network: %+v", network.Name)

		go func(network domain.IrcNetwork) {
			if err := handler.Run(); err != nil {
				s.log.Error().Err(err).Msgf("failed to start handler for network %q", network.Name)
			}
		}(network)
	}
}

func (s *service) StopHandlers() {
	for _, handler := range s.handlers {
		s.log.Info().Msgf("stopping network: %+v", handler.network.Name)
		handler.Stop()
	}

	s.log.Info().Msg("stopped all irc handlers")
}

func (s *service) startNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers already, if so start it
	if existingHandler, found := s.handlers[handlerKey{network.Server, network.Nick}]; found {
		s.log.Debug().Msgf("starting network: %+v", network.Name)

		if !existingHandler.client.Connected() {
			go func(handler *Handler) {
				if err := handler.Run(); err != nil {
					s.log.Error().Err(err).Msgf("failed to start existingHandler for network %q", handler.network.Name)
				}
			}(existingHandler)
		}
	} else {
		// if not found in handlers, lets add it and run it

		s.lock.Lock()
		channels, err := s.repo.ListChannels(network.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network %q", network.Server)
		}
		network.Channels = channels

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		// init new irc handler
		handler := NewHandler(s.log, network, definitions, s.releaseService, s.notificationService)

		s.handlers[handlerKey{network.Server, network.Nick}] = handler
		s.lock.Unlock()

		s.log.Debug().Msgf("starting network: %+v", network.Name)

		go func(network domain.IrcNetwork) {
			if err := handler.Run(); err != nil {
				s.log.Error().Err(err).Msgf("failed to start handler for network %q", network.Name)
			}
		}(network)
	}

	return nil
}

func (s *service) checkIfNetworkRestartNeeded(network *domain.IrcNetwork) error {
	// look if we have the network in handlers, if so restart it
	if existingHandler, found := s.handlers[handlerKey{network.Server, network.Nick}]; found {
		s.log.Debug().Msgf("irc: decide if irc network handler needs restart or updating: %+v", network.Server)

		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		if existingHandler.client.Connected() {
			handler := existingHandler.GetNetwork()
			restartNeeded := false

			if handler.Server != network.Server {
				restartNeeded = true
			} else if handler.Port != network.Port {
				restartNeeded = true
			} else if handler.TLS != network.TLS {
				restartNeeded = true
			} else if handler.InviteCommand != network.InviteCommand {
				restartNeeded = true
			}
			if restartNeeded {
				s.log.Info().Msgf("irc: restarting network: %+v", network.Server)

				// we need to reinitialize with new network config
				existingHandler.UpdateNetwork(network)

				// todo reset channelHealth?

				go func() {
					if err := existingHandler.Restart(); err != nil {
						s.log.Error().Stack().Err(err).Msgf("failed to restart network %q", existingHandler.network.Name)
					}
				}()

				// return now since the restart will read the network again
				return nil
			}

			if handler.Nick != network.Nick {
				s.log.Debug().Msg("changing nick")

				if err := existingHandler.NickChange(network.Nick); err != nil {
					s.log.Error().Stack().Err(err).Msgf("failed to change nick %q", network.Nick)
				}
			} else if handler.Auth.Password != network.Auth.Password {
				s.log.Debug().Msg("nickserv: changing password")

				if err := existingHandler.NickServIdentify(network.Auth.Password); err != nil {
					s.log.Error().Stack().Err(err).Msgf("failed to identify with nickserv %q", network.Nick)
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
				s.log.Debug().Msgf("%v: part channel %v", network.Server, leaveChannel)

				if err := existingHandler.PartChannel(leaveChannel); err != nil {
					s.log.Error().Stack().Err(err).Msgf("failed to leave channel: %q", leaveChannel)
				}
			}

			// join channels
			for _, joinChannel := range channelsToJoin {
				s.log.Debug().Msgf("%v: join new channel %v", network.Server, joinChannel)

				if err := existingHandler.JoinChannel(joinChannel.Name, joinChannel.Password); err != nil {
					s.log.Error().Stack().Err(err).Msgf("failed to join channel: %q", joinChannel.Name)
				}
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
			s.log.Error().Stack().Err(err).Msgf("failed to start network: %q", network.Name)
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
	hk := handlerKey{network.Server, network.Nick}
	if err := s.StopNetworkIfRunning(hk); err != nil {
		return err
	}

	return s.startNetwork(network)
}

func (s *service) StopNetwork(key handlerKey) error {
	if handler, found := s.handlers[key]; found {
		handler.Stop()
		s.log.Debug().Msgf("stopped network: %+v", key.server)
	}

	return nil
}

func (s *service) StopAndRemoveNetwork(key handlerKey) error {
	if handler, found := s.handlers[key]; found {
		handler.Stop()

		// remove from handlers
		delete(s.handlers, key)
		s.log.Debug().Msgf("stopped network: %+v", key)
	}

	return nil
}

func (s *service) StopNetworkIfRunning(key handlerKey) error {
	if handler, found := s.handlers[key]; found {
		handler.Stop()
		s.log.Debug().Msgf("stopped network: %+v", key.server)
	}

	return nil
}

func (s *service) GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error) {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to get network: %v", id)
		return nil, err
	}

	channels, err := s.repo.ListChannels(network.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to list channels for network %q", network.Server)
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

	var ret []domain.IrcNetwork

	for _, n := range networks {
		channels, err := s.repo.ListChannels(n.ID)
		if err != nil {
			s.log.Error().Msgf("failed to list channels for network %q: %v", n.Server, err)
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

	var ret []domain.IrcNetworkWithHealth

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
			Connected:        false,
			Channels:         []domain.ChannelWithHealth{},
			ConnectionErrors: []string{},
		}

		handler, ok := s.handlers[handlerKey{n.Server, n.Nick}]
		if ok {
			handler.m.RLock()

			// only set connected and connected since if we have an active handler and connection
			if handler.client.Connected() {

				netw.Connected = handler.connectedSince != time.Time{}
				netw.ConnectedSince = handler.connectedSince

				// current and preferred nick is only available if the network is connected
				netw.CurrentNick = handler.CurrentNick()
				netw.PreferredNick = handler.PreferredNick()
			}
			netw.Healthy = handler.Healthy()

			// if we have any connection errors like bad nickserv auth add them here
			if len(handler.connectionErrors) > 0 {
				netw.ConnectionErrors = handler.connectionErrors
			}

			handler.m.RUnlock()
		}

		channels, err := s.repo.ListChannels(n.ID)
		if err != nil {
			s.log.Error().Msgf("failed to list channels for network %q: %v", n.Server, err)
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
		s.log.Error().Stack().Err(err).Msgf("could not find network before delete: %v", network.Name)
		return err
	}

	s.log.Debug().Msgf("delete network: %v", id)

	// Remove network and handler
	//if err = s.StopNetwork(network.Server); err != nil {
	if err = s.StopAndRemoveNetwork(handlerKey{network.Server, network.Nick}); err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not stop and delete network: %v", network.Name)
		return err
	}

	if err = s.repo.DeleteNetwork(ctx, id); err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not delete network: %v", network.Name)
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
	s.log.Debug().Msgf("irc.service: update network: %+v", network)

	// stop or start network
	// TODO get current state to see if enabled or not?
	if network.Enabled {
		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		err := s.checkIfNetworkRestartNeeded(network)
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("could not restart network: %+v", network.Name)
			return errors.New("could not restart network: %v", network.Name)
		}

	} else {
		// take into account multiple channels per network
		err := s.StopAndRemoveNetwork(handlerKey{network.Server, network.Nick})
		if err != nil {
			s.log.Error().Stack().Err(err).Msgf("could not stop network: %+v", network.Name)
			return errors.New("could not stop network: %v", network.Name)
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
		if err := s.repo.StoreNetwork(network); err != nil {
			return err
		}
		s.log.Debug().Msgf("store network: %+v", network)

		if network.Channels != nil {
			for _, channel := range network.Channels {
				if err := s.repo.StoreChannel(network.ID, &channel); err != nil {
					s.log.Error().Stack().Err(err).Msg("irc.storeChannel: error executing query")
					return errors.Wrap(err, "error storing channel on network")
					//return err
				}
			}
		}

		return nil
	}

	// get channels for existing network
	existingChannels, err := s.repo.ListChannels(existingNetwork.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to list channels for network %q", existingNetwork.Server)
	}
	existingNetwork.Channels = existingChannels

	if network.Channels != nil {
		for _, channel := range network.Channels {
			// add channels. Make sure it doesn't delete before
			if err := s.repo.StoreChannel(existingNetwork.ID, &channel); err != nil {
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

		err := s.checkIfNetworkRestartNeeded(existingNetwork)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not restart network: %+v", existingNetwork.Name)
			return errors.New("could not restart network: %v", existingNetwork.Name)
		}
	}

	return nil
}

func (s *service) StoreChannel(networkID int64, channel *domain.IrcChannel) error {
	if err := s.repo.StoreChannel(networkID, channel); err != nil {
		return err
	}

	return nil
}
