// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alphadose/haxmap"
	"github.com/ergochat/irc-go/ircevent"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type ircRepo interface {
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	ToggleNetworkEnabled(ctx context.Context, id int64, enabled bool) error
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

type eventBus interface {
	EmitIRC(ctx context.Context, event events.IRCEvent)
	OnProxy(handler func(ctx context.Context, event events.ProxyChangeEvent) error) func()
}

type Service struct {
	log      zerolog.Logger
	eventBus eventBus
	sse      sseServer

	repo           ircRepo
	releaseService releaseService
	indexerService indexerService
	proxyService   proxyService

	networkCache    *haxmap.Map[int64, *domain.IrcNetwork]
	networkHandlers *haxmap.Map[int64, *Handler]

	// unhealthyNetworks is owned by the health monitor goroutine
	unhealthyNetworks map[int64]*unhealthyEpisode
	unhealthySnapshot atomic.Pointer[[]reportedNetwork]
	stopHealthMonitor context.CancelFunc

	// lifecycleM serializes starting, restarting and removing handlers. Each of
	// those looks a handler up and then acts on it, and two of them interleaved on
	// one network either connect two handlers or revive one that was just stopped.
	// It is a one-slot channel rather than a sync.Mutex so StopHandlers can give
	// up waiting for it at the shutdown deadline.
	lifecycleM chan struct{}
	stopped    atomic.Bool

	// wg tracks the background connects and the health monitor, so shutdown can
	// wait for them
	wg sync.WaitGroup
}

func NewService(log zerolog.Logger, eventBus eventBus, sse sseServer, repo ircRepo, releaseSvc releaseService, indexerSvc indexerService, proxySvc proxyService) *Service {
	s := &Service{
		log:             log.With().Str("module", "irc").Logger(),
		eventBus:        eventBus,
		sse:             sse,
		repo:            repo,
		releaseService:  releaseSvc,
		indexerService:  indexerSvc,
		proxyService:    proxySvc,
		networkCache:    haxmap.New[int64, *domain.IrcNetwork](),
		networkHandlers: haxmap.New[int64, *Handler](),
		lifecycleM:      make(chan struct{}, 1),

		unhealthyNetworks: map[int64]*unhealthyEpisode{},
	}

	s.setupEventListeners()

	return s
}

func (s *Service) setupEventListeners() {
	s.eventBus.OnProxy(func(ctx context.Context, event events.ProxyChangeEvent) error {
		if event.Usage == nil {
			return nil
		}

		return s.onProxyChanged(ctx, event)
	})
}

// onProxyChanged reloads every network that routed through the changed proxy and lets the
// restart check pick up the difference: a deleted proxy leaves a detached row and the network
// reconnects direct, an updated one attaches fresh dial settings, and a disabled one takes the
// network down with it rather than letting it fall back to a direct connection.
func (s *Service) onProxyChanged(ctx context.Context, event events.ProxyChangeEvent) error {
	for _, item := range event.Usage.IrcNetworks {
		network, err := s.GetNetworkByID(ctx, item.ID)
		if err != nil {
			s.log.Error().Err(err).Int64("network_id", item.ID).Int64("proxy_id", event.ProxyID).Msg("could not load network for changed proxy")
			continue
		}

		if err := s.attachProxy(ctx, network); err != nil {
			s.log.Error().Err(err).Str("network", network.Name).Int64("proxy_id", network.ProxyId).Msg("could not get changed proxy for network")
			continue
		}

		if network.Enabled && network.Proxy != nil && !network.Proxy.Enabled {
			if err := s.disableNetworkForProxy(ctx, network); err != nil {
				s.log.Error().Err(err).Str("network", network.Name).Int64("proxy_id", event.ProxyID).Msg("could not disable network for disabled proxy")
			}
			continue
		}

		s.networkCache.Set(network.ID, network)

		if !network.Enabled {
			continue
		}

		if err := s.checkIfNetworkRestartNeeded(network); err != nil {
			s.log.Error().Err(err).Str("network", network.Name).Int64("proxy_id", event.ProxyID).Msg("could not restart network for changed proxy")
		}
	}

	return nil
}

// attachProxy resolves the proxy a network is configured with. Every path that hands a network
// to a handler goes through here: Handler.Run refuses to connect when use_proxy is set but no
// Proxy is attached, so a missed attach shows up as an error instead of a direct connection.
func (s *Service) attachProxy(ctx context.Context, network *domain.IrcNetwork) error {
	network.Proxy = nil

	if !network.UseProxy || network.ProxyId == 0 {
		return nil
	}

	networkProxy, err := s.proxyService.FindByID(ctx, network.ProxyId)
	if err != nil {
		return errors.Wrap(err, "could not get proxy for network: %s", network.Server)
	}

	network.Proxy = networkProxy

	return nil
}

// disableNetworkForProxy takes a network down when its proxy is disabled and persists it as
// disabled, so it stays down until the user re-enables it instead of reconnecting direct.
func (s *Service) disableNetworkForProxy(ctx context.Context, network *domain.IrcNetwork) error {
	s.log.Warn().Str("network", network.Name).Str("proxy", network.Proxy.Name).Msg("proxy disabled, disabling network")

	if err := s.StopAndRemoveNetwork(network.ID); err != nil {
		return err
	}

	if err := s.repo.ToggleNetworkEnabled(ctx, network.ID, false); err != nil {
		return err
	}

	network.Enabled = false
	s.networkCache.Set(network.ID, network)

	return nil
}

func (s *Service) StartHandlers() {
	ctx := context.Background()
	networks, err := s.repo.FindActiveNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list networks")
	}

	s.lifecycleM <- struct{}{}
	defer func() { <-s.lifecycleM }()

	for _, network := range networks {
		if !network.Enabled {
			continue
		}

		// still register the handler: Run refuses to connect without the proxy and records
		// why, so the network shows as unhealthy instead of silently missing
		if err := s.attachProxy(ctx, &network); err != nil {
			s.log.Error().Err(err).Str("server", network.Server).Msg("failed to get proxy for network")
		}

		s.startNetwork(network)
	}

	if s.stopped.Load() {
		return
	}

	healthCtx, cancel := context.WithCancel(ctx)
	s.stopHealthMonitor = cancel

	s.wg.Go(func() {
		s.monitorHealth(healthCtx)
	})
}

// StopHandlers stops every network and waits, until ctx is done, for the
// connects in flight to give up and the connections to close.
func (s *Service) StopHandlers(ctx context.Context) {
	s.log.Debug().Msg("stopping all irc handlers..")

	// set before waiting for the lock, so an operation holding it does not
	// connect anything new
	s.stopped.Store(true)

	select {
	case s.lifecycleM <- struct{}{}:
	case <-ctx.Done():
		s.log.Warn().Err(ctx.Err()).Msg("irc lifecycle busy, shutting down without stopping handlers")
		return
	}

	if s.stopHealthMonitor != nil {
		s.stopHealthMonitor()
	}

	clients := make([]*ircevent.Connection, 0, s.networkHandlers.Len())
	for _, handler := range s.networkHandlers.Iterator() {
		s.log.Debug().Str("network", handler.GetNetwork().Name).Msg("stop network")

		if client := handler.stop(); client != nil {
			clients = append(clients, client)
		}
	}
	<-s.lifecycleM

	closed := make(chan struct{})
	go func() {
		s.wg.Wait()
		for _, client := range clients {
			client.Wait()
		}
		close(closed)
	}()

	select {
	case <-closed:
		s.log.Debug().Msg("stopped all irc handlers")

	case <-ctx.Done():
		s.log.Warn().Err(ctx.Err()).Msg("irc connections did not close before shutdown")
	}
}

// startHandler claims handler and connects it in the background.
// The caller must hold s.lifecycleM.
func (s *Service) startHandler(handler *Handler) {
	if s.stopped.Load() {
		return
	}

	connectGen, err := handler.claimConnect()
	if err != nil {
		s.log.Debug().Err(err).Str("network", handler.GetNetwork().Name).Msg("network already connecting")
		return
	}

	s.wg.Go(func() {
		if err := handler.connect(connectGen); err != nil && !errors.Is(err, clientManuallyDisconnected) {
			s.log.Error().Err(err).Str("network", handler.GetNetwork().Name).Msg("failed to start irc handler")
		}
	})
}

// startNetwork starts the handler of network, building it first if there is none.
// The caller must hold s.lifecycleM.
func (s *Service) startNetwork(network domain.IrcNetwork) {
	if s.stopped.Load() {
		return
	}

	if existingHandler, found := s.networkHandlers.Get(network.ID); found {
		s.log.Debug().Str("network", network.Name).Msg("starting network")

		if existingHandler.Stopped() {
			s.startHandler(existingHandler)
		}

		return
	}

	channels, err := s.repo.ListChannels(network.ID)
	if err != nil {
		s.log.Error().Err(err).Str("server", network.Server).Msg("failed to list channels for network")
	}

	network.Channels = channels

	definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

	handler := NewHandler(s.log, s.eventBus, s.sse, network, definitions, s.releaseService)

	s.networkHandlers.Set(network.ID, handler)

	s.log.Debug().Str("network", network.Name).Msg("starting network")

	s.startHandler(handler)
}

// restartHandler reconnects handler with the config of network.
// The caller must hold s.lifecycleM.
func (s *Service) restartHandler(handler *Handler, network *domain.IrcNetwork, definitions []*domain.IndexerDefinition) {
	handler.StopAndWait(restartQuitTimeout)

	if s.stopped.Load() {
		return
	}

	handler.UpdateNetwork(network)
	handler.reloadChannels(definitions)

	s.startHandler(handler)
}

func (s *Service) checkIfNetworkRestartNeeded(network *domain.IrcNetwork) error {
	s.lifecycleM <- struct{}{}
	defer func() { <-s.lifecycleM }()

	handler, found := s.networkHandlers.Get(network.ID)
	if !found {
		s.startNetwork(*network)

		return nil
	}

	s.log.Debug().Str("server", network.Server).Msg("decide if irc network handler needs restart or updating")

	definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

	// a handler in the map is only stopped after a fatal failure (ban, auth, TLS,
	// flapping), and the edit may be what fixes it
	if handler.Stopped() {
		s.log.Debug().Str("server", network.Server).Msg("handler stopped, starting with updated config")

		s.restartHandler(handler, network, definitions)

		return nil
	}

	currentNetwork := handler.GetNetwork()

	// if server, tls, invite command, port : changed - restart
	// if nickserv account, nickserv password : changed - stay connected, and change those
	// if channels len : changes - join or leave
	if diff, shouldRestart := currentNetwork.DetermineIfRestartIsRequired(network); shouldRestart {
		s.log.Debug().Interface("diff", diff).Str("server", network.Server).Msg("fields changed, restarting network")
		s.log.Info().Str("server", network.Server).Msg("restarting network")

		s.restartHandler(handler, network, definitions)

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

	handler.InitIndexers(definitions)

	return nil
}

func (s *Service) RestartNetwork(ctx context.Context, networkID int64) error {
	// load the config under the lock: an edit or disable that lands between a
	// load outside it and the restart would be reverted
	s.lifecycleM <- struct{}{}
	defer func() { <-s.lifecycleM }()

	network, err := s.GetNetworkByID(ctx, networkID)
	if err != nil {
		return err
	}

	if !network.Enabled {
		return errors.New("network disabled, could not restart")
	}

	if err := s.attachProxy(ctx, network); err != nil {
		return err
	}

	handler, found := s.networkHandlers.Get(network.ID)
	if !found {
		s.startNetwork(*network)

		return nil
	}

	s.restartHandler(handler, network, s.indexerService.GetIndexersByIRCNetwork(network.Server))

	return nil
}

func (s *Service) StopAndRemoveNetwork(networkID int64) error {
	s.lifecycleM <- struct{}{}
	defer func() { <-s.lifecycleM }()

	handler, found := s.networkHandlers.Get(networkID)
	if found {
		handler.Stop()

		s.networkHandlers.Del(networkID)

		s.log.Debug().Int64("network_id", networkID).Msg("stopped network")
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
		return errors.Wrap(domain.ErrRecordNotFound, "could not find irc handler with id: %d", network.ID)
	}

	// send to channels announce processor
	channel, foundChannel := handler.channels.Get(req.Channel)
	if !foundChannel {
		return errors.Wrap(domain.ErrIRCChannelNotFound, "channel: %s", req.Channel)
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
				slices.SortStableFunc(netw.Channels, func(a, b domain.IrcChannelWithHealth) int {
					return strings.Compare(a.Name, b.Name)
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
	if err := network.Auth.Validate(); err != nil {
		return err
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

	if err := s.attachProxy(ctx, network); err != nil {
		s.log.Error().Err(err).Str("server", network.Server).Msg("failed to get proxy for network")
		return err
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
	if err := network.Auth.Validate(); err != nil {
		return err
	}

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
			for i := range network.Channels {
				if err := s.repo.StoreChannel(ctx, network.ID, &network.Channels[i]); err != nil {
					s.log.Error().Err(err).Msg("irc.storeChannel: error executing query")
					return errors.Wrap(err, "error storing channel on network")
				}
			}
		}

		s.networkCache.Set(network.ID, network)

		if err := s.attachProxy(ctx, network); err != nil {
			s.log.Error().Err(err).Str("server", network.Server).Msg("failed to get proxy for network")
			return err
		}

		// if network is enabled, start it immediately
		if network.Enabled {
			s.lifecycleM <- struct{}{}
			s.startNetwork(*network)
			<-s.lifecycleM
		}

		return nil
	}

	// get channels for existing network
	existingChannels, err := s.repo.ListChannels(existingNetwork.ID)
	if err != nil {
		s.log.Error().Err(err).Str("server", existingNetwork.Server).Msg("failed to list channels for network")
	}
	existingNetwork.Channels = existingChannels

	if err := s.attachProxy(ctx, existingNetwork); err != nil {
		s.log.Error().Err(err).Str("server", existingNetwork.Server).Msg("failed to get proxy for network")
		return err
	}

	if network.Channels != nil {
		// store through the slice element so the id from the insert reaches the
		// channels handed to the handler and cache, not a loop copy
		for i := range network.Channels {
			if err := s.repo.StoreChannel(ctx, existingNetwork.ID, &network.Channels[i]); err != nil {
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

	// cache only once every write landed, so a failed one cannot leave
	// unsaved channels or invite commands in the cache
	s.networkCache.Set(existingNetwork.ID, existingNetwork)

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
	_, err := s.GetNetworkByID(ctx, networkID)
	if err != nil {
		s.log.Error().Err(err).Msg("could not find existing network")
		return err
	}

	if err := s.repo.StoreChannel(ctx, networkID, channel); err != nil {
		return err
	}

	return nil
}

func (s *Service) SendCmd(_ context.Context, req *domain.SendIrcCmdRequest) error {
	handler, found := s.networkHandlers.Get(req.NetworkId)
	if !found {
		return errors.Wrap(domain.ErrRecordNotFound, "could not find irc handler with id: %d", req.NetworkId)
	}

	if err := handler.SendMsg(req.Channel, req.Message); err != nil {
		s.log.Error().Err(err).Str("channel", req.Channel).Str("msg", req.Message).Msg("could not send message to channel")
	}

	return nil
}
