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
	StoreNetwork(network *domain.IrcNetwork) error
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

		// TODO use network.Server? Need a way to use multiple indexers for one network if same nick
		s.handlers[network.Name] = handler
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
	if existingHandler, found := s.handlers[network.Name]; found {
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

		// find indexer definitions for network and add
		definitions := s.indexerService.GetIndexersByIRCNetwork(network.Server)

		// init new irc handler
		handler := NewHandler(network, s.filterService, s.releaseService, definitions)

		s.lock.Lock()
		s.handlers[network.Name] = handler
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

func (s *service) StopNetwork(name string) error {
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
	if err = s.StopNetwork(network.Name); err != nil {
		return err
	}

	if err = s.repo.DeleteNetwork(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *service) StoreNetwork(network *domain.IrcNetwork) error {
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

	// stop or start network
	if network.Enabled {
		err := s.startNetwork(*network)
		if err != nil {
			log.Error().Err(err).Msgf("could not start network: %+v", network.Name)
			return fmt.Errorf("could not start network: %v", network.Name)
		}

	} else {
		err := s.StopNetwork(network.Name)
		if err != nil {
			log.Error().Err(err).Msgf("could not stop network: %+v", network.Name)
			return fmt.Errorf("could not stop network: %v", network.Name)
		}
	}

	// stop or start network
	//if !network.Enabled {
	//	log.Debug().Msgf("stopping network: %+v", network.Name)
	//
	//	err := s.StopNetwork(network.Name)
	//	if err != nil {
	//		log.Error().Err(err).Msgf("could not stop network: %+v", network.Name)
	//		return fmt.Errorf("could not stop network: %v", network.Name)
	//	}
	//} else {
	//	log.Debug().Msgf("starting network: %+v", network.Name)
	//
	//	err := s.startNetwork(*network)
	//	if err != nil {
	//		log.Error().Err(err).Msgf("could not start network: %+v", network.Name)
	//		return fmt.Errorf("could not start network: %v", network.Name)
	//	}
	//}

	return nil
}

func (s *service) StoreChannel(networkID int64, channel *domain.IrcChannel) error {
	if err := s.repo.StoreChannel(networkID, channel); err != nil {
		return err
	}

	return nil
}
