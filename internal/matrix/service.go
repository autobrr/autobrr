// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/matrix"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type matrixRepo interface {
	ListNetworks(ctx context.Context) ([]domain.MatrixNetwork, error)
	FindActiveNetworks(ctx context.Context) ([]domain.MatrixNetwork, error)
	FindNetworksByHomeserver(ctx context.Context, homeserver string) ([]domain.MatrixNetwork, error)
	GetNetworkByID(ctx context.Context, id int64) (*domain.MatrixNetwork, error)
	StoreNetwork(ctx context.Context, network *domain.MatrixNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.MatrixNetwork) error
	DeleteNetwork(ctx context.Context, id int64) error

	tokenStore
}

type indexerService interface {
	GetIndexersByMatrixHomeserver(homeserver string) []*domain.IndexerDefinition
}

type releaseService interface {
	Process(ctx context.Context, release *domain.Release)
}

// Service manages Matrix networks, their sync handlers and device logins.
type Service struct {
	log        zerolog.Logger
	sse        ssePublisher
	repo       matrixRepo
	releaseSvc releaseService
	indexerSvc indexerService
	httpClient *http.Client

	// m serializes handler lifecycle changes and guards handlers and logins
	m        sync.Mutex
	handlers map[int64]*Handler
	logins   map[int64]*deviceLogin
	stopped  bool

	wg sync.WaitGroup
}

func NewService(log zerolog.Logger, sse ssePublisher, repo matrixRepo, releaseSvc releaseService, indexerSvc indexerService) *Service {
	return &Service{
		log:        log.With().Str("module", "matrix").Logger(),
		sse:        sse,
		repo:       repo,
		releaseSvc: releaseSvc,
		indexerSvc: indexerSvc,
		// the client timeout must outlast the sync long-poll
		httpClient: &http.Client{Timeout: syncTimeout + 60*time.Second, Transport: sharedhttp.Transport},
		handlers:   map[int64]*Handler{},
		logins:     map[int64]*deviceLogin{},
	}
}

// StartHandlers starts a handler for every enabled network.
func (s *Service) StartHandlers(ctx context.Context) error {
	networks, err := s.repo.FindActiveNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not find active matrix networks")
		return err
	}

	s.m.Lock()
	defer s.m.Unlock()

	for _, network := range networks {
		s.startHandler(network)
	}

	return nil
}

// StopHandlers stops every handler and pending login.
func (s *Service) StopHandlers(ctx context.Context) {
	s.m.Lock()

	s.stopped = true

	for id, login := range s.logins {
		login.cancel()
		delete(s.logins, id)
	}

	for id, handler := range s.handlers {
		handler.Stop()
		delete(s.handlers, id)
	}

	s.m.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		s.log.Warn().Msg("matrix logins did not stop before shutdown deadline")
	}
}

// startHandler replaces the network's handler. The caller holds s.m.
func (s *Service) startHandler(network domain.MatrixNetwork) {
	s.stopHandler(network.ID)

	if !network.Enabled || s.stopped {
		return
	}

	definitions := s.indexerSvc.GetIndexersByMatrixHomeserver(network.Homeserver)

	handler := newHandler(s.log, s.sse, s.repo, s.httpClient, s.releaseSvc, network, definitions)
	handler.Start()

	s.handlers[network.ID] = handler

	s.log.Debug().Str("network", network.Name).Int64("network_id", network.ID).Int("indexers", len(definitions)).Msg("started matrix network")
}

// stopHandler stops and removes the network's handler. The caller holds s.m.
func (s *Service) stopHandler(id int64) {
	handler, ok := s.handlers[id]
	if !ok {
		return
	}

	handler.Stop()
	delete(s.handlers, id)
}

// RestartNetwork reconnects a network with its stored settings.
func (s *Service) RestartNetwork(ctx context.Context, id int64) error {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		return err
	}

	s.m.Lock()
	defer s.m.Unlock()

	s.startHandler(*network)

	return nil
}

// ListNetworks returns every network with its runtime health.
func (s *Service) ListNetworks(ctx context.Context) ([]domain.MatrixNetwork, error) {
	networks, err := s.repo.ListNetworks(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not list matrix networks")
		return nil, err
	}

	s.m.Lock()
	defer s.m.Unlock()

	for i := range networks {
		s.attachHealth(&networks[i])
	}

	return networks, nil
}

// GetNetworkByID returns a network with its runtime health.
func (s *Service) GetNetworkByID(ctx context.Context, id int64) (*domain.MatrixNetwork, error) {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.m.Lock()
	defer s.m.Unlock()

	s.attachHealth(network)

	return network, nil
}

// attachHealth fills in handler and login state. The caller holds s.m.
func (s *Service) attachHealth(network *domain.MatrixNetwork) {
	health := &domain.MatrixNetworkHealth{}

	if handler, ok := s.handlers[network.ID]; ok {
		networkHealth, roomHealth := handler.Health()
		health = &networkHealth

		for i := range network.Rooms {
			if rh, ok := roomHealth[network.Rooms[i].Name]; ok {
				network.Rooms[i].Health = &rh
			}
		}
	} else if network.Enabled && !network.HasCredentials() {
		health.Error = reasonNotLoggedIn
	}

	if login, ok := s.logins[network.ID]; ok {
		state := login.snapshot()
		health.Login = &state
	}

	network.Health = health
}

// StoreNetwork adds a network, or merges its rooms into the only existing network
// on the same homeserver, so adding a second indexer reuses the logged in account.
func (s *Service) StoreNetwork(ctx context.Context, network *domain.MatrixNetwork) error {
	if err := network.Validate(); err != nil {
		return err
	}

	existing, err := s.repo.FindNetworksByHomeserver(ctx, network.Homeserver)
	if err != nil {
		s.log.Error().Err(err).Str("homeserver", network.Homeserver).Msg("could not find matrix networks")
		return err
	}

	if len(existing) == 1 && network.ID == 0 {
		current := existing[0]

		for _, room := range network.Rooms {
			if !slices.ContainsFunc(current.Rooms, func(r domain.MatrixRoom) bool { return r.Name == room.Name }) {
				current.Rooms = append(current.Rooms, room)
			}
		}

		if err := s.repo.UpdateNetwork(ctx, &current); err != nil {
			s.log.Error().Err(err).Int64("network_id", current.ID).Msg("could not add rooms to matrix network")
			return err
		}

		*network = current

		return s.RestartNetwork(ctx, current.ID)
	}

	if network.AccessToken != "" {
		network.AuthMethod = domain.MatrixAuthMethodToken
	}

	if err := s.repo.StoreNetwork(ctx, network); err != nil {
		s.log.Error().Err(err).Str("network", network.Name).Msg("could not store matrix network")
		return err
	}

	return s.RestartNetwork(ctx, network.ID)
}

// UpdateNetwork updates the settings and rooms of a network and restarts it.
func (s *Service) UpdateNetwork(ctx context.Context, network *domain.MatrixNetwork) error {
	if err := network.Validate(); err != nil {
		return err
	}

	current, err := s.repo.GetNetworkByID(ctx, network.ID)
	if err != nil {
		return err
	}

	// a different homeserver means a different account; the old token must not be sent to it
	if current.Homeserver != network.Homeserver && current.HasCredentials() {
		if err := s.repo.UpdateTokens(ctx, network.ID, domain.MatrixTokens{}); err != nil {
			s.log.Error().Err(err).Int64("network_id", network.ID).Msg("could not clear matrix tokens")
			return err
		}
	}

	if err := s.repo.UpdateNetwork(ctx, network); err != nil {
		s.log.Error().Err(err).Int64("network_id", network.ID).Msg("could not update matrix network")
		return err
	}

	return s.RestartNetwork(ctx, network.ID)
}

// DeleteNetwork stops and deletes a network.
func (s *Service) DeleteNetwork(ctx context.Context, id int64) error {
	s.m.Lock()
	s.stopHandler(id)
	s.cancelLogin(id)
	s.m.Unlock()

	if err := s.repo.DeleteNetwork(ctx, id); err != nil {
		s.log.Error().Err(err).Int64("network_id", id).Msg("could not delete matrix network")
		return err
	}

	return nil
}

// SetAccessToken stores a pasted access token after checking it against the homeserver.
func (s *Service) SetAccessToken(ctx context.Context, req *domain.MatrixSetTokenRequest) error {
	network, err := s.repo.GetNetworkByID(ctx, req.NetworkID)
	if err != nil {
		return err
	}

	client := matrix.NewClient(network.Homeserver, s.httpClient)
	client.SetAccessToken(req.AccessToken)

	whoami, err := client.Whoami(ctx)
	if err != nil {
		return errors.Wrap(err, "could not verify access token")
	}

	s.m.Lock()
	defer s.m.Unlock()

	s.stopHandler(network.ID)
	s.cancelLogin(network.ID)

	tokens := domain.MatrixTokens{
		AuthMethod:  domain.MatrixAuthMethodToken,
		UserID:      whoami.UserID,
		DeviceID:    whoami.DeviceID,
		AccessToken: req.AccessToken,
	}

	if err := s.repo.UpdateTokens(ctx, network.ID, tokens); err != nil {
		s.log.Error().Err(err).Int64("network_id", network.ID).Msg("could not store matrix access token")
		return err
	}

	applyTokens(network, tokens)
	s.startHandler(*network)

	return nil
}

// Logout stops a network and forgets its credentials.
func (s *Service) Logout(ctx context.Context, id int64) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.stopHandler(id)
	s.cancelLogin(id)

	if err := s.repo.UpdateTokens(ctx, id, domain.MatrixTokens{}); err != nil {
		s.log.Error().Err(err).Int64("network_id", id).Msg("could not clear matrix tokens")
		return err
	}

	return nil
}

// GetMessageHistory returns the recorded messages of a room. A network that is not
// running has no history.
func (s *Service) GetMessageHistory(_ context.Context, networkID int64, room string) ([]domain.MatrixMessage, error) {
	s.m.Lock()
	handler, ok := s.handlers[networkID]
	s.m.Unlock()

	if !ok {
		return make([]domain.MatrixMessage, 0), nil
	}

	return handler.History(room)
}

// ManualProcessAnnounce feeds a message into a room's announce processor.
func (s *Service) ManualProcessAnnounce(_ context.Context, req *domain.MatrixManualProcessRequest) error {
	s.m.Lock()
	defer s.m.Unlock()

	handler, ok := s.handlers[req.NetworkID]
	if !ok {
		return errors.New("matrix network %d is not running", req.NetworkID)
	}

	return handler.QueueAnnounce(req.Room, req.Message)
}

func applyTokens(network *domain.MatrixNetwork, tokens domain.MatrixTokens) {
	network.AuthMethod = tokens.AuthMethod
	network.UserID = tokens.UserID
	network.DeviceID = tokens.DeviceID
	network.AccessToken = tokens.AccessToken
	network.RefreshToken = tokens.RefreshToken
	network.TokenExpiresAt = tokens.TokenExpiresAt
	network.OAuthClientID = tokens.OAuthClientID
	network.OAuthTokenEndpoint = tokens.OAuthTokenEndpoint
}
