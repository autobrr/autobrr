// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alphadose/haxmap"
	"github.com/jellydator/ttlcache/v3"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type Service interface {
	ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error)
	GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	StoreChannel(ctx context.Context, networkID int64, channel *domain.IrcChannel) error
	ManualProcessAnnounce(ctx context.Context, req *domain.IRCManualProcessRequest) error

	GetMessageHistory(ctx context.Context, networkID int64, channel string) ([]domain.IrcMessage, error)

	StartHandlers()
	StopHandlers()
	StopAndRemoveNetwork(id int64) error
	StopNetwork(id int64) error
	RestartNetwork(ctx context.Context, id int64) error
	SendCmd(ctx context.Context, req *domain.SendIrcCmdRequest) error
}

type indexerSvc interface {
	GetIndexersByIRCNetwork(server string) []*domain.IndexerDefinition
}

type proxySvc interface {
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
}

type service struct {
	log zerolog.Logger
	sse *sse.Server

	repo                domain.IrcRepo
	releaseService      release.Processor
	indexerService      indexerSvc
	notificationService notification.Sender
	proxyService        proxySvc

	networkCache    *ttlcache.Cache[int64, *domain.IrcNetwork]
	networkHandlers *haxmap.Map[int64, *Handler]

	stopWG sync.WaitGroup
	lock   sync.RWMutex
}

const sseMaxEntries = 1000

func NewService(log logger.Logger, sse *sse.Server, repo domain.IrcRepo, releaseSvc release.Processor, indexerSvc indexerSvc, notificationSvc notification.Sender, proxySvc proxySvc) Service {
	return &service{
		log:                 log.With().Str("module", "irc").Logger(),
		sse:                 sse,
		repo:                repo,
		releaseService:      releaseSvc,
		indexerService:      indexerSvc,
		notificationService: notificationSvc,
		proxyService:        proxySvc,
		networkCache:        ttlcache.New[int64, *domain.IrcNetwork](ttlcache.WithTTL[int64, *domain.IrcNetwork](5 * time.Minute)),
		networkHandlers:     haxmap.New[int64, *Handler](),
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

		if network.UseProxy && network.ProxyId != 0 {
			networkProxy, err := s.proxyService.FindByID(context.Background(), network.ProxyId)
			if err != nil {
				s.log.Error().Err(err).Msgf("failed to get proxy for network: %s", network.Server)
				continue
			}
			network.Proxy = networkProxy
		}

		channels, err := s.repo.ListChannels(network.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network: %s", network.Server)
		}

		//for _, channel := range channels {
		//	// setup SSE stream per channel
		//	//s.createSSEStream(network.ID, channel.Name)
		//}

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		network.Channels = channels

		// init new irc handler
		handler := NewHandler(s.log, s.sse, network, definitions, s.releaseService, s.notificationService)

		s.networkHandlers.Set(network.ID, handler)

		s.log.Debug().Msgf("starting network: %s", network.Name)

		go func(network domain.IrcNetwork) {
			if err := handler.Run(); err != nil {
				s.log.Error().Err(err).Msgf("failed to start handler for network: %s", network.Name)
			}
		}(network)
	}
}

func (s *service) StopHandlers() {
	s.log.Info().Msg("stopping all irc handlers..")

	s.networkHandlers.ForEach(func(i int64, handler *Handler) bool {
		s.log.Info().Msgf("stop network: %s", handler.network.Name)

		handler.Stop()

		return true
	})

	s.log.Info().Msg("stopped all irc handlers")
}

func (s *service) startNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers already, if so start it
	if existingHandler, found := s.networkHandlers.Get(network.ID); found {
		s.log.Debug().Msgf("starting network: %s", network.Name)

		if existingHandler.Stopped() {
			go func(handler *Handler) {
				if err := handler.Run(); err != nil {
					s.log.Error().Err(err).Msgf("failed to start existing handler for network: %s", handler.network.Name)
				}
			}(existingHandler)
		}

		return nil
	}

	// if not found in handlers, lets add it and run it
	channels, err := s.repo.ListChannels(network.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to list channels for network: %s", network.Server)
	}

	//for _, channel := range channels {
	//	// setup SSE stream per channel
	//	s.createSSEStream(network.ID, channel.Name)
	//}

	// find indexer definitions for network and add
	definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

	network.Channels = channels

	// init new irc handler
	handler := NewHandler(s.log, s.sse, network, definitions, s.releaseService, s.notificationService)

	s.networkHandlers.Set(network.ID, handler)

	s.log.Debug().Msgf("starting network: %s", network.Name)

	go func(network domain.IrcNetwork) {
		if err := handler.Run(); err != nil {
			s.log.Error().Err(err).Msgf("failed to start handler for network: %s", network.Name)
		}
	}(network)

	return nil
}

func (s *service) checkIfNetworkRestartNeeded(network *domain.IrcNetwork) error {
	handler, found := s.networkHandlers.Get(network.ID)
	if !found {
		if err := s.startNetwork(*network); err != nil {
			s.log.Error().Err(err).Msgf("failed to start network: %s", network.Name)
		}

		return nil
	}

	s.log.Debug().Msgf("irc: decide if irc network handler needs restart or updating: %s", network.Server)

	if handler.Stopped() {
		s.log.Debug().Msgf("irc: handler stopped, skip: %s", network.Server)
		return nil
	}

	currentNetwork := handler.GetNetwork()

	// if server, tls, invite command, port : changed - restart
	// if nickserv account, nickserv password : changed - stay connected, and change those
	// if channels len : changes - join or leave
	if diff, shouldRestart := DetermineNetworkRestartRequired(*currentNetwork, *network); shouldRestart {
		s.log.Debug().Msgf("irc: fields %+v changed, restarting network: %s", diff, network.Server)
		s.log.Info().Msgf("irc: restarting network: %s", network.Server)

		// we need to reinitialize with new network config
		handler.UpdateNetwork(network)

		go func() {
			if err := handler.Restart(); err != nil {
				s.log.Error().Stack().Err(err).Msgf("failed to restart network: %s", handler.network.Name)
			}
		}()

		// return now since the restart will read the network again
		return nil
	}

	// if nick is different lets try change it
	if currentNetwork.Nick != network.Nick {
		s.log.Debug().Msg("changing nick")

		if err := handler.NickChange(network.Nick); err != nil {
			s.log.Error().Err(err).Msgf("failed to change nick: %s", network.Nick)
		}
	}

	// TODO refactor channel join/part mess below

	// join or leave channels
	// loop over currentNetwork channels,
	var expectedChannels = make(map[string]struct{}, 0)
	var handlerChannels = make(map[string]struct{}, 0)
	var channelsToLeave = make([]string, 0)
	var channelsToJoin = make([]domain.IrcChannel, 0)

	// create map of expected channels
	for _, channel := range network.Channels {
		expectedChannels[channel.Name] = struct{}{}
	}

	// check current channels of currentNetwork against expected
	for _, handlerChan := range currentNetwork.Channels {
		handlerChannels[handlerChan.Name] = struct{}{}

		_, ok := expectedChannels[handlerChan.Name]
		if ok {
			// 	if currentNetwork channel matches network channel next
			continue
		}

		// if not expected, leave
		channelsToLeave = append(channelsToLeave, handlerChan.Name)
	}

	// check new channels against currentNetwork to see which to join
	for _, channel := range network.Channels {
		_, ok := handlerChannels[channel.Name]
		if ok {
			continue
		}

		// if expected channel not in currentNetwork channels, add to join
		// use channel struct for extra info
		channelsToJoin = append(channelsToJoin, channel)
	}

	// leave channels
	for _, leaveChannel := range channelsToLeave {
		s.log.Debug().Msgf("%s: part channel %s", network.Server, leaveChannel)

		if err := handler.PartChannel(leaveChannel); err != nil {
			s.log.Error().Err(err).Msgf("failed to leave channel: %s", leaveChannel)
		}

		// remove SSE stream for channel
		//s.removeSSEStream(network.ID, leaveChannel)
	}

	// join channels
	for _, joinChannel := range channelsToJoin {
		s.log.Debug().Msgf("%s: join new channel %s", network.Server, joinChannel.Name)

		if err := handler.JoinChannel(joinChannel.Name, joinChannel.Password); err != nil {
			s.log.Error().Err(err).Msgf("failed to join channel: %s", joinChannel.Name)
		}

		// create SSE stream for new channel
		//s.createSSEStream(network.ID, joinChannel.Name)
	}

	// update network for currentNetwork
	// TODO move all this restart logic inside currentNetwork to let it decide what to do
	handler.SetNetwork(network)

	// find indexer definitions for network and add
	definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

	handler.InitIndexers(definitions)

	return nil
}

func (s *service) RestartNetwork(ctx context.Context, id int64) error {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		return err
	}

	if !network.Enabled {
		return errors.New("network disabled, could not restart")
	}

	return s.restartNetwork(*network)
}

func (s *service) restartNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers, if so restart it
	if err := s.StopNetwork(network.ID); err != nil {
		return err
	}

	return s.startNetwork(network)
}

func (s *service) StopAndRemoveNetwork(id int64) error {
	if handler, found := s.networkHandlers.Get(id); found {
		// remove SSE streams
		//handler.channels.ForEach(func(_ string, channel *Channel) bool {
		//	s.removeSSEStream(handler.network.ID, channel.Name)
		//	return true
		//})

		handler.Stop()

		// remove from handlers
		s.networkHandlers.Del(id)

		s.log.Debug().Msgf("stopped network: %d", id)
	}

	return nil
}

func (s *service) StopNetwork(id int64) error {
	if handler, found := s.networkHandlers.Get(id); found {
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

func (s *service) ManualProcessAnnounce(ctx context.Context, req *domain.IRCManualProcessRequest) error {
	network, err := s.repo.GetNetworkByID(ctx, req.NetworkId)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to get network: %d", req.NetworkId)
		return err
	}

	handler, found := s.networkHandlers.Get(network.ID)
	if !found {
		return errors.New("could not find irc handler with id: %d", network.ID)
	}

	// send to channels announce processor
	channel, foundChannel := handler.channels.Get(req.Channel)
	if foundChannel {
		err = channel.QueueAnnounceLine(req.Message)
		if err != nil {
			return errors.Wrap(err, "could not send manual announce to processor")
		}
	}

	return nil
}

func (s *service) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	networks, err := s.repo.ListNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
		return nil, err
	}

	ret := make([]domain.IrcNetwork, len(networks))

	for idx, n := range networks {
		channels, err := s.repo.ListChannels(n.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network: %s", n.Server)
			return nil, err
		}
		n.Channels = channels

		ret[idx] = n
	}

	return ret, nil
}

func (s *service) listNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	if s.networkCache.Len() > 0 {
		s.log.Trace().Msgf("found %d networks in cache", s.networkCache.Len())

		ret := make([]domain.IrcNetwork, s.networkCache.Len())
		idx := 0
		s.networkCache.Range(func(item *ttlcache.Item[int64, *domain.IrcNetwork]) bool {
			ret[idx] = *item.Value()
			idx++

			return true
		})

		return ret, nil
	}

	s.log.Trace().Msg("no networks in cache, fetching from db")

	networks, err := s.repo.ListNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
		return nil, err
	}

	ret := make([]domain.IrcNetwork, len(networks))

	for idx, ircNetwork := range networks {
		channels, err := s.repo.ListChannels(ircNetwork.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to list channels for network: %s", ircNetwork.Server)
			return nil, err
		}

		ircNetwork.Channels = channels

		s.networkCache.Set(ircNetwork.ID, &ircNetwork, ttlcache.DefaultTTL)

		ret[idx] = ircNetwork
	}

	return ret, nil
}

func (s *service) GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error) {
	networks, err := s.ListNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
		return nil, err
	}

	ret := make([]domain.IrcNetworkWithHealth, len(networks))

	for networkIdx, n := range networks {
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
			BotMode:          n.BotMode,
			UseProxy:         n.UseProxy,
			ProxyId:          n.ProxyId,
			Connected:        false,
			ConnectionErrors: []string{},
			Bots:             make([]domain.IrcUser, 0),
			Channels:         []domain.IrcChannelWithHealth{},
		}

		if n.Enabled {
			handler, found := s.networkHandlers.Get(n.ID)
			if found {
				handler.ReportStatus(&netw)

				handler.channels.ForEach(func(name string, channel *Channel) bool {
					ch := domain.IrcChannelWithHealth{
						ID:               channel.ID,
						Enabled:          channel.Enabled,
						Name:             channel.Name,
						Password:         channel.Password,
						Detached:         false,
						State:            channel.stateMachine.state.String(),
						Monitoring:       channel.Monitoring,
						MonitoringSince:  channel.MonitoringSince,
						LastAnnounce:     channel.LastAnnounce,
						ConnectionErrors: slices.Clone(channel.ConnectionErrors),
					}

					channel.announcers.ForEach(func(nick string, announcer *domain.IrcUser) bool {
						ch.Announcers = append(ch.Announcers, *announcer)
						return true
					})

					netw.Channels = append(netw.Channels, ch)

					return true
				})

				handler.bots.ForEach(func(name string, user *domain.IrcUser) bool {
					netw.Bots = append(netw.Bots, *user)
					return true
				})

				// sort alphabetically so the ui doesn't jump around randomly between auto-refresh
				sort.SliceStable(netw.Channels, func(i, j int) bool {
					return netw.Channels[i].Name < netw.Channels[j].Name
				})
			}
		} else {
			// combine from repo and handler
			for _, channel := range n.Channels {
				ch := domain.IrcChannelWithHealth{
					ID:               channel.ID,
					Enabled:          channel.Enabled,
					Name:             channel.Name,
					Password:         channel.Password,
					Detached:         channel.Detached,
					Monitoring:       false,
					MonitoringSince:  time.Time{},
					LastAnnounce:     time.Time{},
					ConnectionErrors: []string{},
					Announcers:       []domain.IrcUser{},
				}

				netw.Channels = append(netw.Channels, ch)
			}
		}

		ret[networkIdx] = netw
	}

	return ret, nil
}

func (s *service) GetMessageHistory(_ context.Context, networkID int64, channel string) ([]domain.IrcMessage, error) {
	handler, found := s.networkHandlers.Get(networkID)
	if !found {
		return nil, errors.New("could not find network handler")
	}

	channelInstance, ok := handler.channels.Get(channel)
	if !ok {
		return nil, errors.New("could not find channel")
	}

	messages := channelInstance.Messages.GetMessages()

	return messages, nil
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

	s.networkCache.Delete(id)

	return nil
}

func (s *service) UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	existingNetwork, err := s.GetNetworkByID(ctx, network.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("could not find existing network")
		return err
	}

	if domain.IsRedactedString(network.Pass) {
		network.Pass = existingNetwork.Pass
	}

	if domain.IsRedactedString(network.Auth.Password) {
		network.Auth.Password = existingNetwork.Auth.Password
	}

	s.log.Debug().Msgf("irc.service: update network: %s", network.Name)

	if err := s.repo.UpdateNetwork(ctx, network); err != nil {
		return err
	}

	if network.Channels != nil {
		for idx, channel := range network.Channels {
			if domain.IsRedactedString(channel.Password) {
				index := slices.IndexFunc(existingNetwork.Channels, func(existingChannel domain.IrcChannel) bool {
					return existingChannel.ID == channel.ID
				})
				if index == -1 {
					s.log.Error().Msgf("could not find channel %s in existing network", channel.Name)
					return errors.New("could not find channel in existing network")
				}

				network.Channels[idx].Password = existingNetwork.Channels[index].Password
			}
		}

		if err := s.repo.StoreNetworkChannels(ctx, network.ID, network.Channels); err != nil {
			return err
		}
	}

	network.Proxy = nil

	// attach proxy
	if network.UseProxy && network.ProxyId != 0 {
		networkProxy, err := s.proxyService.FindByID(ctx, network.ProxyId)
		if err != nil {
			s.log.Error().Err(err).Msgf("failed to get proxy for network: %s", network.Server)
			return errors.Wrap(err, "could not get proxy for network: %s", network.Server)
		}
		network.Proxy = networkProxy
	}

	s.networkCache.Set(network.ID, network, ttlcache.DefaultTTL)

	// stop or start network
	// TODO get current state to see if enabled or not?
	if network.Enabled {
		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		if err := s.checkIfNetworkRestartNeeded(network); err != nil {
			s.log.Error().Err(err).Msgf("could not restart network: %s", network.Name)
			return errors.New("could not restart network: %s", network.Name)
		}

	} else {
		// take into account multiple channels per network
		if err := s.StopAndRemoveNetwork(network.ID); err != nil {
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

		s.networkCache.Set(network.ID, network, ttlcache.DefaultTTL)

		return nil
	}

	// get channels for existing network
	existingChannels, err := s.repo.ListChannels(existingNetwork.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("failed to list channels for network: %s", existingNetwork.Server)
	}
	existingNetwork.Channels = existingChannels

	s.networkCache.Set(network.ID, existingNetwork, ttlcache.DefaultTTL)

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
	//if handler, found := s.handlers[req.NetworkId]; found {
	if handler, found := s.networkHandlers.Get(req.NetworkId); found {
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
