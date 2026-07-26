// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alphadose/haxmap"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type ircRepo interface {
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	StoreChannel(ctx context.Context, networkID int64, channel *domain.IrcChannel) error
	UpdateChannel(channel *domain.IrcChannel) error
	UpdateInviteCommand(networkID int64, invite string) error
	StoreNetworkChannels(ctx context.Context, networkID int64, channels []domain.IrcChannel) error
	CheckExistingNetwork(ctx context.Context, network *domain.IrcNetwork) (*domain.IrcNetwork, error)
	FindActiveNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	ListChannels(networkID int64) ([]domain.IrcChannel, error)
	GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
}

type indexerService interface {
	GetIndexersByIRCNetwork(server string) []*domain.IndexerDefinition
}

type notificationSender interface {
	Send(event domain.NotificationEvent, payload domain.NotificationPayload)
}

type releaseService interface {
	Process(ctx context.Context, release *domain.Release)
}

type proxyService interface {
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
}

type sseServer interface {
	Publish(id string, event *sse.Event)
	CreateStreamWithOpts(id string, opts sse.StreamOpts) *sse.Stream
	RemoveStream(id string)
}

type Service struct {
	log zerolog.Logger
	sse sseServer

	repo                ircRepo
	releaseService      releaseService
	indexerService      indexerService
	notificationService notificationSender
	proxyService        proxyService

	networkCache    *haxmap.Map[int64, *domain.IrcNetwork]
	networkHandlers *haxmap.Map[int64, *Handler]

	stopWG sync.WaitGroup
	lock   sync.RWMutex
}

func NewService(log zerolog.Logger, sse sseServer, repo ircRepo, releaseSvc releaseService, indexerSvc indexerService, notificationSvc notificationSender, proxySvc proxyService) *Service {
	return &Service{
		log:                 log.With().Str("module", "irc").Logger(),
		sse:                 sse,
		repo:                repo,
		releaseService:      releaseSvc,
		indexerService:      indexerSvc,
		notificationService: notificationSvc,
		proxyService:        proxySvc,
		networkCache:        haxmap.New[int64, *domain.IrcNetwork](),
		networkHandlers:     haxmap.New[int64, *Handler](),
	}
}

func (s *Service) StartHandlers() {
	ctx := context.Background()
	networks, err := s.repo.FindActiveNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
	}

	for _, network := range networks {
		if !network.Enabled {
			continue
		}

		if network.UseProxy && network.ProxyId != 0 {
			networkProxy, err := s.proxyService.FindByID(ctx, network.ProxyId)
			if err != nil {
				s.log.Error().Err(err).Str("server", network.Server).Msg("failed to get proxy for network")
				continue
			}
			network.Proxy = networkProxy
		}

		channels, err := s.repo.ListChannels(network.ID)
		if err != nil {
			s.log.Error().Err(err).Str("server", network.Server).Msg("failed to list channels for network")
		}

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		network.Channels = channels

		// init new irc handler
		handler := NewHandler(s.log, s.sse, network, definitions, s.releaseService, s.notificationService)

		s.networkHandlers.Set(network.ID, handler)

		s.log.Debug().Str("network", network.Name).Msg("starting network")

		go func(network domain.IrcNetwork) {
			if err := handler.Run(); err != nil {
				s.log.Error().Err(err).Str("network", network.Name).Msg("failed to start irc handler")
			}
		}(network)
	}
}

func (s *Service) StopHandlers() {
	s.log.Info().Msg("stopping all irc handlers..")

	for _, handler := range s.networkHandlers.Iterator() {
		s.log.Info().Str("network", handler.network.Name).Msg("stop network")
		handler.Stop()
	}

	s.log.Info().Msg("stopped all irc handlers")
}

func (s *Service) startNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers already, if so start it
	if existingHandler, found := s.networkHandlers.Get(network.ID); found {
		s.log.Debug().Str("network", network.Name).Msg("starting network")

		if existingHandler.Stopped() {
			go func(handler *Handler) {
				if err := handler.Run(); err != nil {
					s.log.Error().Err(err).Str("network", handler.network.Name).Msg("failed to start irc handler")
				}
			}(existingHandler)
		}

		return nil
	}

	// if not found in handlers, lets add it and run it
	channels, err := s.repo.ListChannels(network.ID)
	if err != nil {
		s.log.Error().Err(err).Str("server", network.Server).Msg("failed to list channels for network")
	}

	// find indexer definitions for network and add
	definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

	network.Channels = channels

	// init new irc handler
	handler := NewHandler(s.log, s.sse, network, definitions, s.releaseService, s.notificationService)

	s.networkHandlers.Set(network.ID, handler)

	s.log.Debug().Str("network", network.Name).Msg("starting network")

	go func(network domain.IrcNetwork) {
		if err := handler.Run(); err != nil {
			s.log.Error().Err(err).Str("network", network.Name).Msg("failed to start irc handler")
		}
	}(network)

	return nil
}

func (s *Service) checkIfNetworkRestartNeeded(network *domain.IrcNetwork) error {
	handler, found := s.networkHandlers.Get(network.ID)
	if !found {
		if err := s.startNetwork(*network); err != nil {
			s.log.Error().Err(err).Str("network", network.Name).Msg("failed to start network")
		}

		return nil
	}

	s.log.Debug().Str("server", network.Server).Msg("decide if irc network handler needs restart or updating")

	if handler.Stopped() {
		s.log.Debug().Str("server", network.Server).Msg("handler stopped, skipping")
		return nil
	}

	currentNetwork := handler.GetNetwork()

	// if server, tls, invite command, port : changed - restart
	// if nickserv account, nickserv password : changed - stay connected, and change those
	// if channels len : changes - join or leave
	if diff, shouldRestart := currentNetwork.DetermineIfRestartIsRequired(network); shouldRestart {
		s.log.Debug().Interface("diff", diff).Str("server", network.Server).Msg("fields changed, restarting network")
		s.log.Info().Str("server", network.Server).Msg("restarting network")

		// we need to reinitialize with the new network config
		handler.UpdateNetwork(network)

		go func() {
			if err := handler.Restart(); err != nil {
				s.log.Error().Stack().Err(err).Str("network", handler.network.Name).Msg("failed to restart network")
			}
		}()

		// return now since the restart will read the network again
		return nil
	}

	// if nick is different lets try change it
	if currentNetwork.Nick != network.Nick {
		s.log.Debug().Msg("changing nick")

		if err := handler.NickChange(network.Nick); err != nil {
			s.log.Error().Err(err).Str("nick", network.Nick).Msg("failed to change nick")
		}
	}

	// TODO refactor channel join/part mess below

	// join or leave channels
	// loop over currentNetwork channels,
	var expectedChannels = make(map[string]struct{}, 0)
	var handlerChannels = make(map[string]struct{}, 0)
	var channelsToLeave = make([]string, 0)
	var channelsToJoin = make([]domain.IrcChannel, 0)
	var channelsToUpdate = make([]domain.IrcChannel, 0)

	// create map of expected channels (keyed lowercase to match handler storage)
	for _, channel := range network.Channels {
		expectedChannels[strings.ToLower(channel.Name)] = struct{}{}
	}

	// check current channels of currentNetwork against expected
	for _, handlerChan := range currentNetwork.Channels {
		name := strings.ToLower(handlerChan.Name)
		handlerChannels[name] = struct{}{}

		if _, ok := expectedChannels[name]; ok {
			// 	if currentNetwork channel matches network channel next
			continue
		}

		// if not expected, leave
		channelsToLeave = append(channelsToLeave, handlerChan.Name)
	}

	// check new channels against currentNetwork: join the new ones, reconcile the
	// config (password/enabled) of the ones we already track
	for _, channel := range network.Channels {
		if _, ok := handlerChannels[strings.ToLower(channel.Name)]; ok {
			channelsToUpdate = append(channelsToUpdate, channel)
			continue
		}

		// if expected channel not in currentNetwork channels, add to join
		// use channel struct for extra info
		channelsToJoin = append(channelsToJoin, channel)
	}

	// leave channels
	for _, leaveChannel := range channelsToLeave {
		s.log.Debug().Str("server", network.Server).Str("channel", leaveChannel).Msg("part channel")

		handler.RemoveChannel(leaveChannel)
	}

	// join channels. AddChannel registers the Channel + state machine before
	// sending JOIN so the JOIN echo is not treated as an unwanted channel.
	for _, joinChannel := range channelsToJoin {
		s.log.Debug().Str("server", network.Server).Str("channel", joinChannel.Name).Msg("join new channel")

		handler.AddChannel(joinChannel)
	}

	// reconcile config changes (e.g. a channel password / +k key) on the fly,
	// without restarting the network
	for _, updateChannel := range channelsToUpdate {
		handler.UpdateChannel(updateChannel)
	}

	// update network for currentNetwork
	// TODO move all this restart logic inside currentNetwork to let it decide what to do
	handler.SetNetwork(network)

	// find indexer definitions for network and add
	definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

	handler.InitIndexers(definitions)

	return nil
}

func (s *Service) RestartNetwork(ctx context.Context, networkID int64) error {
	network, err := s.repo.GetNetworkByID(ctx, networkID)
	if err != nil {
		return err
	}

	if !network.Enabled {
		return errors.New("network disabled, could not restart")
	}

	return s.restartNetwork(*network)
}

func (s *Service) restartNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers, if so restart it
	if err := s.StopNetwork(network.ID); err != nil {
		return err
	}

	return s.startNetwork(network)
}

func (s *Service) StopAndRemoveNetwork(networkID int64) error {
	handler, found := s.networkHandlers.Get(networkID)
	if found {
		handler.Stop()

		// remove from handlers
		s.networkHandlers.Del(networkID)

		s.log.Debug().Int64("network_id", networkID).Msg("stopped network")
	}

	return nil
}

func (s *Service) StopNetwork(networkID int64) error {
	handler, found := s.networkHandlers.Get(networkID)
	if found {
		handler.Stop()
		s.log.Debug().Str("server", handler.network.Server).Msg("stopped network")

	}

	return nil
}

func (s *Service) GetNetworkByID(ctx context.Context, networkID int64) (*domain.IrcNetwork, error) {
	network, err := s.repo.GetNetworkByID(ctx, networkID)
	if err != nil {
		s.log.Error().Err(err).Int64("network_id", networkID).Msg("failed to get network")
		return nil, err
	}

	channels, err := s.repo.ListChannels(network.ID)
	if err != nil {
		s.log.Error().Err(err).Str("server", network.Server).Msg("failed to list channels")
		return nil, err
	}
	network.Channels = append(network.Channels, channels...)

	return network, nil
}

func (s *Service) ManualProcessAnnounce(ctx context.Context, req *domain.IRCManualProcessRequest) error {
	network, err := s.repo.GetNetworkByID(ctx, req.NetworkId)
	if err != nil {
		s.log.Error().Err(err).Int64("network_id", req.NetworkId).Msg("failed to get network")
		return err
	}

	handler, found := s.networkHandlers.Get(network.ID)
	if !found {
		return errors.New("could not find irc handler with id: %d", network.ID)
	}

	// send to channels announce processor
	channel, foundChannel := handler.channels.Get(req.Channel)

	if !foundChannel {
		return errors.New("could not find channel: %s", req.Channel)
	}

	if err := channel.QueueAnnounceLine(req.Message); err != nil {
		return errors.Wrap(err, "could not send manual announce to processor")
	}

	return nil
}

func (s *Service) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	networks, err := s.repo.ListNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
		return nil, err
	}

	ret := make([]domain.IrcNetwork, len(networks))

	for idx, n := range networks {
		channels, err := s.repo.ListChannels(n.ID)
		if err != nil {
			s.log.Error().Err(err).Str("server", n.Server).Msg("failed to list channels")
			return nil, err
		}
		n.Channels = channels

		ret[idx] = n
	}

	return ret, nil
}

func (s *Service) listNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	if s.networkCache.Len() > 0 {
		s.log.Trace().Int("count", int(s.networkCache.Len())).Msg("found networks in cache")

		ret := make([]domain.IrcNetwork, s.networkCache.Len())
		idx := 0
		for _, net := range s.networkCache.Iterator() {
			ret[idx] = *net
			idx++
		}

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
			s.log.Error().Err(err).Str("server", ircNetwork.Server).Msg("failed to list channels")
			return nil, err
		}

		ircNetwork.Channels = channels

		s.networkCache.Set(ircNetwork.ID, &ircNetwork)

		ret[idx] = ircNetwork
	}

	return ret, nil
}

func (s *Service) GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error) {
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
			TLSSkipVerify:    n.TLSSkipVerify,
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
			Channels:         []domain.IrcChannelWithHealth{},
		}

		if n.Enabled {
			handler, found := s.networkHandlers.Get(n.ID)
			if found {
				handler.ReportStatus(&netw)

				for _, channel := range handler.channels.Iterator() {
					snap := channel.Snapshot()

					state := ""
					if sm := channel.StateMachine(); sm != nil {
						state = sm.CurrentState().String()
					}

					ch := domain.IrcChannelWithHealth{
						ID:               snap.ID,
						Enabled:          snap.Enabled,
						Name:             snap.Name,
						Password:         snap.Password,
						Detached:         false,
						State:            state,
						Monitoring:       snap.Monitoring,
						MonitoringSince:  snap.MonitoringSince,
						LastAnnounce:     snap.LastAnnounce,
						ConnectionErrors: snap.ConnectionErrors,
					}

					netw.Channels = append(netw.Channels, ch)
				}

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
				}

				netw.Channels = append(netw.Channels, ch)
			}
		}

		ret[networkIdx] = netw
	}

	return ret, nil
}

func (s *Service) GetMessageHistory(_ context.Context, networkID int64, channel string) ([]domain.IrcMessage, error) {
	handler, found := s.networkHandlers.Get(networkID)
	if !found {
		return nil, domain.ErrIRCNetworkHandlerNotFound
	}

	channelInstance, ok := handler.channels.Get(channel)
	if !ok {
		return nil, errors.New("could not find channel")
	}

	messages := channelInstance.Messages.GetMessages()

	return messages, nil
}

func (s *Service) DeleteNetwork(ctx context.Context, networkID int64) error {
	network, err := s.GetNetworkByID(ctx, networkID)
	if err != nil {
		s.log.Error().Err(err).Int64("network_id", networkID).Msg("could not find network before delete")
		return err
	}

	s.log.Debug().Int64("network_id", networkID).Str("network", network.Name).Msg("delete network")

	// Remove network and handler
	if err = s.StopAndRemoveNetwork(network.ID); err != nil {
		s.log.Error().Err(err).Str("network", network.Name).Msg("could not stop and delete network")
		return err
	}

	if err = s.repo.DeleteNetwork(ctx, networkID); err != nil {
		s.log.Error().Err(err).Str("network", network.Name).Msg("could not delete network")
		return err
	}

	s.networkCache.Del(networkID)

	return nil
}

func (s *Service) UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error {
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

	s.log.Debug().Str("network", network.Name).Msg("update network")

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
					s.log.Error().Str("channel", channel.Name).Msg("could not find channel in existing network")
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
			s.log.Error().Err(err).Str("server", network.Server).Msg("failed to get proxy for network")
			return errors.Wrap(err, "could not get proxy for network: %s", network.Server)
		}
		network.Proxy = networkProxy
	}

	s.networkCache.Set(network.ID, network)

	// stop or start network
	// TODO get current state to see if enabled or not?
	if network.Enabled {
		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		if err := s.checkIfNetworkRestartNeeded(network); err != nil {
			s.log.Error().Err(err).Str("network", network.Name).Msg("could not restart network")
			return errors.New("could not restart network: %s", network.Name)
		}

	} else {
		// take into account multiple channels per network
		if err := s.StopAndRemoveNetwork(network.ID); err != nil {
			s.log.Error().Err(err).Str("network", network.Name).Msg("could not stop network")
			return errors.New("could not stop network: %s", network.Name)
		}
	}

	return nil
}

func (s *Service) StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	existingNetwork, err := s.repo.CheckExistingNetwork(ctx, network)
	if err != nil {
		s.log.Error().Err(err).Msg("could not check for existing network")
		return err
	}

	if existingNetwork == nil {
		if err := s.repo.StoreNetwork(ctx, network); err != nil {
			return err
		}
		s.log.Debug().Interface("network", network).Msg("store network")

		if network.Channels != nil {
			for _, channel := range network.Channels {
				if err := s.repo.StoreChannel(ctx, network.ID, &channel); err != nil {
					s.log.Error().Err(err).Msg("irc.storeChannel: error executing query")
					return errors.Wrap(err, "error storing channel on network")
				}
			}
		}

		s.networkCache.Set(network.ID, network)

		// attach proxy
		network.Proxy = nil
		if network.UseProxy && network.ProxyId != 0 {
			networkProxy, err := s.proxyService.FindByID(ctx, network.ProxyId)
			if err != nil {
				s.log.Error().Err(err).Str("server", network.Server).Msg("failed to get proxy for network")
				return errors.Wrap(err, "could not get proxy for network: %s", network.Server)
			}
			network.Proxy = networkProxy
		}

		// if network is enabled, start it immediately
		if network.Enabled {
			if err := s.startNetwork(*network); err != nil {
				s.log.Error().Err(err).Str("network", network.Name).Msg("could not start network")
				return errors.New("could not start network: %s", network.Name)
			}
		}

		return nil
	}

	// get channels for existing network
	existingChannels, err := s.repo.ListChannels(existingNetwork.ID)
	if err != nil {
		s.log.Error().Err(err).Str("server", existingNetwork.Server).Msg("failed to list channels for network")
	}
	existingNetwork.Channels = existingChannels

	s.networkCache.Set(network.ID, existingNetwork)

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
			s.log.Error().Err(err).Str("network", existingNetwork.Name).Msg("could not restart network")
			return errors.New("could not restart network: %s", existingNetwork.Name)
		}
	}

	return nil
}

func (s *Service) StoreChannel(ctx context.Context, networkID int64, channel *domain.IrcChannel) error {
	if err := s.repo.StoreChannel(ctx, networkID, channel); err != nil {
		return err
	}

	return nil
}

func (s *Service) SendCmd(_ context.Context, req *domain.SendIrcCmdRequest) error {
	handler, found := s.networkHandlers.Get(req.NetworkId)
	if !found {
		return errors.New("could not find irc handler with id: %d", req.NetworkId)
	}

	if err := handler.SendMsg(req.Channel, req.Message); err != nil {
		s.log.Error().Err(err).Str("channel", req.Channel).Str("msg", req.Message).Msg("could not send message to channel")
	}

	return nil
}
