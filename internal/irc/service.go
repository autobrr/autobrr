package irc

import (
	"context"
	"fmt"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/rs/zerolog/log"
)

type Service interface {
	StartHandlers()
	StopHandlers()
	StopNetwork(name string) error
	ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	GetNetworkByID(id int64) (*domain.IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	StoreChannel(networkID int64, channel *domain.IrcChannel) error
}

type service struct {
	repo           domain.IrcRepo
	filterService  filter.Service
	indexerService indexer.Service
	releaseService release.Service
	indexerMap     map[string]string
	handlers       map[string]*Handler

	stopWG sync.WaitGroup
	lock   sync.Mutex
}

func NewService(repo domain.IrcRepo, filterService filter.Service, indexerSvc indexer.Service, releaseSvc release.Service) Service {
	return &service{
		repo:           repo,
		filterService:  filterService,
		indexerService: indexerSvc,
		releaseService: releaseSvc,
		handlers:       make(map[string]*Handler),
	}
}

func (s *service) StartHandlers() {
	networks, err := s.repo.FindActiveNetworks(context.Background())
	if err != nil {
		log.Error().Msgf("failed to list networks: %v", err)
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
			log.Error().Err(err).Msgf("failed to list channels for network %q", network.Server)
		}
		network.Channels = channels

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		// init new irc handler
		handler := NewHandler(network, s.filterService, s.releaseService, definitions)

		// TODO use network.Server? + nick? Need a way to use multiple indexers for one network if same nick
		s.handlers[network.Server] = handler
		s.lock.Unlock()

		log.Debug().Msgf("starting network: %+v", network.Name)

		s.stopWG.Add(1)

		go func() {
			if err := handler.Run(); err != nil {
				log.Error().Err(err).Msgf("failed to start handler for network %q", network.Name)
			}
		}()

		s.stopWG.Done()
	}
}

func (s *service) StopHandlers() {
	for _, handler := range s.handlers {
		log.Info().Msgf("stopping network: %+v", handler.network.Name)
		handler.Stop()
	}

	log.Info().Msg("stopped all irc handlers")
}

func (s *service) startNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers already, if so start it
	if existingHandler, found := s.handlers[network.Server]; found {
		log.Debug().Msgf("starting network: %+v", network.Name)

		if existingHandler.conn != nil {
			go func() {
				if err := existingHandler.Run(); err != nil {
					log.Error().Err(err).Msgf("failed to start existingHandler for network %q", existingHandler.network.Name)
				}
			}()
		}
	} else {
		// if not found in handlers, lets add it and run it

		s.lock.Lock()
		channels, err := s.repo.ListChannels(network.ID)
		if err != nil {
			log.Error().Err(err).Msgf("failed to list channels for network %q", network.Server)
		}
		network.Channels = channels

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		// init new irc handler
		handler := NewHandler(network, s.filterService, s.releaseService, definitions)

		s.handlers[network.Server] = handler
		s.lock.Unlock()

		log.Debug().Msgf("starting network: %+v", network.Name)

		s.stopWG.Add(1)

		go func() {
			if err := handler.Run(); err != nil {
				log.Error().Err(err).Msgf("failed to start handler for network %q", network.Name)
			}
		}()

		s.stopWG.Done()
	}

	return nil
}

func (s *service) checkIfNetworkRestartNeeded(network *domain.IrcNetwork) error {
	// look if we have the network in handlers, if so restart it
	// TODO check if we need to add indexerDefinitions etc
	if existingHandler, found := s.handlers[network.Server]; found {
		log.Debug().Msgf("decide if irc network handler needs restart or updating: %+v", network.Name)

		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		if existingHandler.conn != nil {
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
				log.Info().Msgf("irc: restarting network: %+v", network.Name)

				// we need to reinitialize with new network config
				existingHandler.UpdateNetwork(network)

				go func() {
					if err := existingHandler.Restart(); err != nil {
						log.Error().Stack().Err(err).Msgf("failed to restart network %q", existingHandler.network.Name)
					}
				}()

				// return now since the restart will read the network again OR FIXME
				return nil
			}

			if handler.NickServ.Account != network.NickServ.Account {
				log.Debug().Msg("changing nick")

				err := existingHandler.HandleNickChange(network.NickServ.Account)
				if err != nil {
					log.Error().Stack().Err(err).Msgf("failed to change nick %q", network.NickServ.Account)
				}
			} else if handler.NickServ.Password != network.NickServ.Password {
				log.Debug().Msg("nickserv: changing password")

				err := existingHandler.HandleNickServIdentify(network.NickServ.Account, network.NickServ.Password)
				if err != nil {
					log.Error().Stack().Err(err).Msgf("failed to identify with nickserv %q", network.NickServ.Account)
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

			for _, channel := range network.Channels {
				_, ok := handlerChannels[channel.Name]
				if ok {
					continue
				}

				// 	if expected channel not in handler channels, add to join
				// use channel struct for extra info
				channelsToJoin = append(channelsToJoin, channel)

				// TODO if not in network.Channels, or not enabled, leave channel
			}

			// leave channels
			for _, leaveChannel := range channelsToLeave {
				err := existingHandler.HandlePartChannel(leaveChannel)
				if err != nil {
					log.Error().Stack().Err(err).Msgf("failed to leave channel: %q", leaveChannel)
				}
			}

			// join channels
			for _, joinChannel := range channelsToJoin {
				// TODO handle invite commands before?
				err := existingHandler.HandleJoinChannel(joinChannel.Name, joinChannel.Password)
				if err != nil {
					log.Error().Stack().Err(err).Msgf("failed to join channel: %q", joinChannel.Name)
				}
			}
		}
	} else {
		err := s.startNetwork(*network)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("failed to start network: %q", network.Name)
		}
	}

	return nil
}

func (s *service) restartNetwork(network domain.IrcNetwork) error {
	// look if we have the network in handlers, if so restart it
	if existingHandler, found := s.handlers[network.Server]; found {
		log.Info().Msgf("restarting network: %+v", network.Name)

		if existingHandler.conn != nil {
			go func() {
				if err := existingHandler.Restart(); err != nil {
					log.Error().Err(err).Msgf("failed to restart network %q", existingHandler.network.Name)
				}
			}()
		}
	}

	// TODO handle full restart

	return nil
}

func (s *service) StopNetwork(name string) error {
	if handler, found := s.handlers[name]; found {
		handler.Stop()
		log.Debug().Msgf("stopped network: %+v", name)
	}

	return nil
}

func (s *service) StopAndRemoveNetwork(name string) error {
	if handler, found := s.handlers[name]; found {
		handler.Stop()

		// remove from handlers
		delete(s.handlers, name)
		log.Debug().Msgf("stopped network: %+v", name)
	}

	return nil
}

func (s *service) StopNetworkIfRunning(name string) error {
	if handler, found := s.handlers[name]; found {
		handler.Stop()
		log.Debug().Msgf("stopped network: %+v", name)
	}

	return nil
}

func (s *service) GetNetworkByID(id int64) (*domain.IrcNetwork, error) {
	network, err := s.repo.GetNetworkByID(id)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get network: %v", id)
		return nil, err
	}

	channels, err := s.repo.ListChannels(network.ID)
	if err != nil {
		log.Error().Err(err).Msgf("failed to list channels for network %q", network.Server)
		return nil, err
	}
	network.Channels = append(network.Channels, channels...)

	return network, nil
}

func (s *service) ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error) {
	networks, err := s.repo.ListNetworks(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("failed to list networks: %v", err)
		return nil, err
	}

	var ret []domain.IrcNetwork

	for _, n := range networks {
		channels, err := s.repo.ListChannels(n.ID)
		if err != nil {
			log.Error().Msgf("failed to list channels for network %q: %v", n.Server, err)
			return nil, err
		}
		n.Channels = append(n.Channels, channels...)

		ret = append(ret, n)
	}

	return ret, nil
}

func (s *service) DeleteNetwork(ctx context.Context, id int64) error {
	network, err := s.GetNetworkByID(id)
	if err != nil {
		return err
	}

	log.Debug().Msgf("delete network: %v", id)

	// Remove network and handler
	//if err = s.StopNetwork(network.Server); err != nil {
	if err = s.StopAndRemoveNetwork(network.Server); err != nil {
		return err
	}

	if err = s.repo.DeleteNetwork(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *service) UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error {

	if network.Channels != nil {
		if err := s.repo.StoreNetworkChannels(ctx, network.ID, network.Channels); err != nil {
			return err
		}

		//for _, channel := range network.Channels {
		//	if err := s.repo.StoreChannel(existingNetwork.ID, &channel); err != nil {
		//		return err
		//	}
		//}
	}

	if err := s.repo.UpdateNetwork(ctx, network); err != nil {
		return err
	}
	log.Debug().Msgf("irc.service: update network: %+v", network)

	// stop or start network
	// TODO get current state to see if enabled or not?
	if network.Enabled {
		// if it's only channels affected, simply leave or join channels instead of restart
		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave
		//err := s.restartNetwork(*network)

		err := s.checkIfNetworkRestartNeeded(network)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("could not restart network: %+v", network.Name)
			return fmt.Errorf("could not restart network: %v", network.Name)
		}

	} else {
		// TODO take into account multiple channels per network
		//err := s.StopNetwork(network.Server)
		err := s.StopAndRemoveNetwork(network.Server)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("could not stop network: %+v", network.Name)
			return fmt.Errorf("could not stop network: %v", network.Name)
		}
	}

	return nil
}

func (s *service) StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error {
	existingNetwork, err := s.repo.CheckExistingNetwork(ctx, network)
	if err != nil {
		log.Error().Err(err).Msg("could not check for existing network")
		return err
	}

	if existingNetwork == nil {
		if err := s.repo.StoreNetwork(network); err != nil {
			return err
		}
		log.Debug().Msgf("store network: %+v", network)

		if network.Channels != nil {
			for _, channel := range network.Channels {
				if err := s.repo.StoreChannel(network.ID, &channel); err != nil {
					return err
				}
			}
		}

		return nil
	}

	if network.Channels != nil {
		for _, channel := range network.Channels {
			// TODO store or add. Make sure it doesn't delete before
			if err := s.repo.StoreChannel(existingNetwork.ID, &channel); err != nil {
				return err
			}
		}

		// append channels to existing network
		network.Channels = append(network.Channels, existingNetwork.Channels...)
	}

	if existingNetwork.Enabled {
		// TODO if it's only channels affected, simply leave or join channels instead of restart
		// decideToRestartJoinOrLeaveChannel()
		// if server, tls, invite command, port : changed - restart
		// if nickserv account, nickserv password : changed - stay connected, and change those
		// if channels len : changes - join or leave

		//err := s.restartNetwork(*network)
		err := s.checkIfNetworkRestartNeeded(network)
		if err != nil {
			log.Error().Err(err).Msgf("could not restart network: %+v", network.Name)
			return fmt.Errorf("could not restart network: %v", network.Name)
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
