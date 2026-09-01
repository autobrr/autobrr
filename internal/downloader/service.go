// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package downloader

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/aria2"
	"github.com/autobrr/autobrr/pkg/arr/lidarr"
	"github.com/autobrr/autobrr/pkg/arr/radarr"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"
	"github.com/autobrr/autobrr/pkg/arr/sportarr"
	"github.com/autobrr/autobrr/pkg/arr/whisparr"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/nzbget"
	"github.com/autobrr/autobrr/pkg/porla"
	"github.com/autobrr/autobrr/pkg/sabnzbd"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
	"github.com/autobrr/autobrr/pkg/transmission"

	"github.com/autobrr/go-deluge"
	"github.com/autobrr/go-qbittorrent"
	"github.com/autobrr/go-rtorrent"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/icholy/digest"
	"github.com/rs/zerolog"
)

type downloaderRepo interface {
	List(ctx context.Context) ([]domain.Downloader, error)
	FindByID(ctx context.Context, clientID int32) (*domain.Downloader, error)
	Store(ctx context.Context, client *domain.Downloader) error
	Update(ctx context.Context, client *domain.Downloader) error
	Delete(ctx context.Context, clientID int32) error
}
type Service struct {
	log  zerolog.Logger
	repo downloaderRepo

	instances *Cache
}

func NewService(log zerolog.Logger, repo downloaderRepo) *Service {
	s := &Service{
		log:  log.With().Str("module", "downloader").Logger(),
		repo: repo,

		instances: NewCache(),
	}

	return s
}

func (s *Service) List(ctx context.Context) ([]domain.Downloader, error) {
	clients, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not list download clients")
		return nil, err
	}

	return clients, nil
}

func (s *Service) FindByID(ctx context.Context, clientID int32) (*domain.Downloader, error) {
	cachedClient, ok := s.instances.Get(clientID)
	if ok {
		return cachedClient.Config(), nil
	}

	s.log.Trace().Int32("client_id", clientID).Msg("cache miss for client, continue to repo lookup")

	client, err := s.repo.FindByID(ctx, clientID)
	if err != nil {
		s.log.Error().Err(err).Int32("client_id", clientID).Msg("could not find download client by id")
		return nil, err
	}

	return client, nil
}

func (s *Service) GetArrTags(ctx context.Context, id int32) ([]domain.ArrTag, error) {
	data := make([]domain.ArrTag, 0)

	instance, err := s.GetInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	cfg := instance.Config()

	switch cfg.Type {
	case domain.DownloaderTypeRadarr:
		client, err := ClientAs[*radarr.Client](instance)
		if err != nil {
			return nil, err
		}

		tags, err := client.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from radarr")
			return data, nil
		}

		for _, tag := range tags {
			emt := domain.ArrTag{
				ID:    tag.ID,
				Label: tag.Label,
			}
			data = append(data, emt)
		}

		return data, nil

	case domain.DownloaderTypeSonarr:
		client, err := ClientAs[*sonarr.Client](instance)
		if err != nil {
			return nil, err
		}

		tags, err := client.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from sonarr")
			return data, nil
		}

		for _, tag := range tags {
			emt := domain.ArrTag{
				ID:    tag.ID,
				Label: tag.Label,
			}
			data = append(data, emt)
		}

		return data, nil

	case domain.DownloaderTypeWhisparr, domain.DownloaderTypeWhisparrV3:
		client, err := ClientAs[*whisparr.Client](instance)
		if err != nil {
			return nil, err
		}

		tags, err := client.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from whisparr")
			return data, nil
		}

		for _, tag := range tags {
			emt := domain.ArrTag{
				ID:    tag.ID,
				Label: tag.Label,
			}
			data = append(data, emt)
		}

		return data, nil

	case domain.DownloaderTypeSportarr:
		client, err := ClientAs[*sportarr.Client](instance)
		if err != nil {
			return nil, err
		}

		tags, err := client.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from sportarr")
			return data, nil
		}

		for _, tag := range tags {
			emt := domain.ArrTag{
				ID:    tag.ID,
				Label: tag.Label,
			}
			data = append(data, emt)
		}

		return data, nil

	default:
		return data, nil
	}
}

func (s *Service) Store(ctx context.Context, client *domain.Downloader) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	// store
	if err := s.repo.Store(ctx, client); err != nil {
		s.log.Error().Err(err).Interface("client", client).Msg("could not store download client")
		return err
	}

	instance, err := s.initInstance(client)
	if err != nil {
		s.log.Error().Err(err).Interface("client", client).Msg("could not init instance")
		return err
	}
	s.instances.Set(client.ID, instance)

	return nil
}

func (s *Service) Update(ctx context.Context, client *domain.Downloader) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	existingClient, err := s.FindByID(ctx, client.ID)
	if err != nil {
		s.log.Error().Err(err).Int32("client_id", client.ID).Msg("could not find download client")
		return err
	}

	if domain.IsRedactedString(client.Password) {
		client.Password = existingClient.Password
	}

	if domain.IsRedactedString(client.Settings.APIKey) {
		client.Settings.APIKey = existingClient.Settings.APIKey
	}

	if domain.IsRedactedString(client.Settings.Auth.Password) {
		client.Settings.Auth.Password = existingClient.Settings.Auth.Password
	}

	if domain.IsRedactedString(client.Settings.Basic.Password) {
		client.Settings.Basic.Password = existingClient.Settings.Basic.Password
	}

	// update
	if err := s.repo.Update(ctx, client); err != nil {
		s.log.Error().Err(err).Interface("client", client).Msg("could not update download client")
		return err
	}

	instance, err := s.initInstance(client)
	if err != nil {
		s.log.Error().Err(err).Interface("client", client).Msg("could not initialize download client")
		return err
	}
	s.instances.Set(client.ID, instance)

	return err
}

func (s *Service) Delete(ctx context.Context, clientID int32) error {
	_, err := s.FindByID(ctx, clientID)
	if err != nil {
		s.log.Error().Err(err).Int32("client_id", clientID).Msg("could not find download client")
		return err
	}

	if err := s.repo.Delete(ctx, clientID); err != nil {
		s.log.Error().Err(err).Int32("client_id", clientID).Msg("could not delete download client")
		return err
	}

	s.instances.Delete(clientID)

	return nil
}

func (s *Service) Test(ctx context.Context, client *domain.Downloader) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	// check for existing client to get settings from
	if client.ID > 0 {
		existingClient, err := s.FindByID(ctx, client.ID)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", client.ID).Msg("could not find download client")
			return err
		}

		if domain.IsRedactedString(client.Password) {
			client.Password = existingClient.Password
		}
		if domain.IsRedactedString(client.Settings.APIKey) {
			client.Settings.APIKey = existingClient.Settings.APIKey
		}
		if domain.IsRedactedString(client.Settings.Auth.Password) {
			client.Settings.Auth.Password = existingClient.Settings.Auth.Password
		}
		if domain.IsRedactedString(client.Settings.Basic.Password) {
			client.Settings.Basic.Password = existingClient.Settings.Basic.Password
		}
	}

	// test
	if err := s.testConnection(ctx, client); err != nil {
		s.log.Error().Err(err).Msg("client connection test error")
		return err
	}

	return nil
}

func (s *Service) initInstance(cfg *domain.Downloader) (*Instance, error) {
	instance := &Instance{
		config: cfg,
	}

	switch cfg.Type {
	case domain.DownloaderTypeAria2:
		clientCfg := aria2.Config{
			Host:          cfg.Host,
			Secret:        cfg.Settings.APIKey,
			TLSSkipVerify: cfg.TLSSkipVerify,
			BasicUser:     cfg.Settings.Auth.Username,
			BasicPass:     cfg.Settings.Auth.Password,
			Log:           s.log.With().Str("type", "Aria2").Str("client", cfg.Name).Logger(),
		}
		if cfg.Settings.Auth.Enabled {
			clientCfg.BasicUser = cfg.Settings.Auth.Username
			clientCfg.BasicPass = cfg.Settings.Auth.Password
		}

		client, err := aria2.NewClient(clientCfg)
		if err != nil {
			return nil, errors.Wrap(err, "error creating aria2 client: %s", cfg.Host)
		}
		instance.client = client

	case domain.DownloaderTypeDelugeV1:
		clientCfg := deluge.Settings{
			Hostname:             cfg.Host,
			Port:                 uint(cfg.Port),
			Login:                cfg.Username,
			Password:             cfg.Password,
			DebugServerResponses: true,
			ReadWriteTimeout:     time.Second * 60,
		}
		instance.client = deluge.NewV1(clientCfg)

	case domain.DownloaderTypeDelugeV2:
		clientCfg := deluge.Settings{
			Hostname:             cfg.Host,
			Port:                 uint(cfg.Port),
			Login:                cfg.Username,
			Password:             cfg.Password,
			DebugServerResponses: true,
			ReadWriteTimeout:     time.Second * 60,
		}
		instance.client = deluge.NewV2(clientCfg)

	case domain.DownloaderTypeQbittorrent:
		clientHost, err := cfg.BuildLegacyHost()
		if err != nil {
			return nil, errors.Wrap(err, "error building qBittorrent host url: %v", cfg.Host)
		}

		clientCfg := qbittorrent.Config{
			Host:          clientHost,
			Username:      cfg.Username,
			Password:      cfg.Password,
			APIKey:        cfg.Settings.APIKey,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "qBittorrent").Str("client", cfg.Name).Logger(), zerolog.TraceLevel),
		}

		if cfg.Settings.Auth.Enabled {
			clientCfg.BasicUser = cfg.Settings.Auth.Username
			clientCfg.BasicPass = cfg.Settings.Auth.Password
		}

		instance.client = qbittorrent.NewClient(clientCfg)

	case domain.DownloaderTypePorla:
		clientCfg := porla.Config{
			Hostname:      cfg.Host,
			AuthToken:     cfg.Settings.APIKey,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           s.log.With().Str("type", "Porla").Str("client", cfg.Name).Logger(),
		}

		if cfg.Settings.Auth.Enabled {
			clientCfg.BasicUser = cfg.Settings.Auth.Username
			clientCfg.BasicPass = cfg.Settings.Auth.Password
		}

		instance.client = porla.NewClient(clientCfg)

	case domain.DownloaderTypeRTorrent:
		rtCfg := rtorrent.Config{
			Addr:          cfg.Host,
			TLSSkipVerify: cfg.TLSSkipVerify,
			BasicUser:     cfg.Settings.Auth.Username,
			BasicPass:     cfg.Settings.Auth.Password,
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "rTorrent").Str("client", cfg.Name).Logger(), zerolog.TraceLevel),
		}

		//if cfg.Settings.Auth.Enabled {
		//	rtCfg.BasicUser = cfg.Settings.Auth.Username
		//	rtCfg.BasicPass = cfg.Settings.Auth.Password
		//}

		var opts []rtorrent.OptFunc
		if cfg.Settings.Auth.Type == domain.DownloaderAuthTypeDigest {
			transport := &digest.Transport{
				Username:  cfg.Settings.Auth.Username,
				Password:  cfg.Settings.Auth.Password,
				Transport: sharedhttp.Transport,
			}

			if cfg.TLSSkipVerify {
				transport.Transport = sharedhttp.TransportTLSInsecure
			}

			httpClient := &http.Client{Transport: transport}

			opts = append(opts, rtorrent.WithCustomClient(httpClient))
		}

		instance.client = rtorrent.NewClientWithOpts(rtCfg, opts...)

	case domain.DownloaderTypeTransmission:
		clientHost, err := cfg.BuildLegacyHost()
		if err != nil {
			return nil, errors.Wrap(err, "error building Transmission host url: %v", cfg.Host)
		}

		transmissionURL, err := url.Parse(clientHost)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse transmission url")
		}

		clientCfg := &transmission.Config{
			UserAgent:     "autobrr",
			Username:      cfg.Username,
			Password:      cfg.Password,
			TLSSkipVerify: cfg.TLSSkipVerify,
		}

		client, err := transmission.New(transmissionURL, clientCfg)
		if err != nil {
			return nil, errors.Wrap(err, "error logging into transmission client: %s", cfg.Host)
		}
		instance.client = client

	case domain.DownloaderTypeSabnzbd:
		clientCfg := sabnzbd.Options{
			Addr:   cfg.Host,
			ApiKey: cfg.Settings.APIKey,
			Log:    s.log.With().Str("type", "Sabnzbd").Str("client", cfg.Name).Logger(),
		}

		if cfg.Settings.Auth.Enabled {
			clientCfg.BasicUser = cfg.Settings.Auth.Username
			clientCfg.BasicPass = cfg.Settings.Auth.Password
		}

		instance.client = sabnzbd.New(clientCfg)

	case domain.DownloaderTypeNzbget:
		clientCfg := nzbget.Options{
			Host:     cfg.Host,
			Username: cfg.Username,
			Password: cfg.Password,
			Log:      s.log.With().Str("type", "Nzbget").Str("client", cfg.Name).Logger(),
		}
		instance.client = nzbget.New(clientCfg)

	case domain.DownloaderTypeLidarr:
		clientCfg := lidarr.Config{
			Hostname:      cfg.Host,
			APIKey:        cfg.Settings.APIKey,
			BasicAuth:     cfg.Settings.Auth.Enabled,
			Username:      cfg.Settings.Auth.Username,
			Password:      cfg.Settings.Auth.Password,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           s.log.With().Str("type", "Lidarr").Str("client", cfg.Name).Logger(),
		}
		instance.client = lidarr.New(clientCfg)

	case domain.DownloaderTypeRadarr:
		clientCfg := radarr.Config{
			Hostname:      cfg.Host,
			APIKey:        cfg.Settings.APIKey,
			BasicAuth:     cfg.Settings.Auth.Enabled,
			Username:      cfg.Settings.Auth.Username,
			Password:      cfg.Settings.Auth.Password,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           s.log.With().Str("type", "Radarr").Str("client", cfg.Name).Logger(),
		}
		instance.client = radarr.New(clientCfg)

	case domain.DownloaderTypeReadarr:
		clientCfg := readarr.Config{
			Hostname:      cfg.Host,
			APIKey:        cfg.Settings.APIKey,
			BasicAuth:     cfg.Settings.Auth.Enabled,
			Username:      cfg.Settings.Auth.Username,
			Password:      cfg.Settings.Auth.Password,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           s.log.With().Str("type", "Readarr").Str("client", cfg.Name).Logger(),
		}
		instance.client = readarr.New(clientCfg)

	case domain.DownloaderTypeSonarr:
		clientCfg := sonarr.Config{
			Hostname:      cfg.Host,
			APIKey:        cfg.Settings.APIKey,
			BasicAuth:     cfg.Settings.Auth.Enabled,
			Username:      cfg.Settings.Auth.Username,
			Password:      cfg.Settings.Auth.Password,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           s.log.With().Str("type", "Sonarr").Str("client", cfg.Name).Logger(),
		}
		instance.client = sonarr.New(clientCfg)

	case domain.DownloaderTypeSportarr:
		clientCfg := sportarr.Config{
			Hostname:      cfg.Host,
			APIKey:        cfg.Settings.APIKey,
			BasicAuth:     cfg.Settings.Auth.Enabled,
			Username:      cfg.Settings.Auth.Username,
			Password:      cfg.Settings.Auth.Password,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           s.log.With().Str("type", "Sportarr").Str("client", cfg.Name).Logger(),
		}
		instance.client = sportarr.New(clientCfg)

	case domain.DownloaderTypeWhisparr, domain.DownloaderTypeWhisparrV3:
		clientCfg := whisparr.Config{
			Hostname:      cfg.Host,
			APIKey:        cfg.Settings.APIKey,
			Version:       whisparrVersion(cfg.Type),
			BasicAuth:     cfg.Settings.Auth.Enabled,
			Username:      cfg.Settings.Auth.Username,
			Password:      cfg.Settings.Auth.Password,
			TLSSkipVerify: cfg.TLSSkipVerify,
			Log:           s.log.With().Str("type", "Whisparr").Str("client", cfg.Name).Logger(),
		}
		instance.client = whisparr.New(clientCfg)
	}

	return instance, nil
}

// GetInstance get downloader instance from cache or repo and attach client implementation
func (s *Service) GetInstance(ctx context.Context, clientId int32) (*Instance, error) {
	cachedClient, ok := s.instances.Get(clientId)
	if ok {
		return cachedClient, nil
	}

	cfg, err := s.repo.FindByID(ctx, clientId)
	if err != nil {
		s.log.Error().Err(err).Int32("client_id", clientId).Msg("could not find download client")
		return nil, err
	}

	instance, err := s.initInstance(cfg)
	if err != nil {
		s.log.Error().Err(err).Int32("client_id", clientId).Msg("could not initialize download client")
		return nil, err
	}

	s.instances.Set(clientId, instance)

	return instance, nil
}
