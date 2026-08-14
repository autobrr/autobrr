// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package download_client

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/aria2"
	"github.com/autobrr/autobrr/pkg/arr/chaptarr"
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
	"github.com/autobrr/autobrr/pkg/transmission"

	"github.com/autobrr/go-deluge"
	"github.com/autobrr/go-qbittorrent"
	"github.com/autobrr/go-rtorrent"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/icholy/digest"
	"github.com/rs/zerolog"
)

type downloadClientRepo interface {
	List(ctx context.Context) ([]domain.DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error)
	Store(ctx context.Context, client *domain.DownloadClient) error
	Update(ctx context.Context, client *domain.DownloadClient) error
	Delete(ctx context.Context, clientID int32) error
}
type Service struct {
	log       zerolog.Logger
	repo      downloadClientRepo
	subLogger *log.Logger

	cache *ClientCache
	m     sync.RWMutex
}

func NewService(log zerolog.Logger, repo downloadClientRepo) *Service {
	s := &Service{
		log:  log.With().Str("module", "download_client").Logger(),
		repo: repo,

		cache: NewClientCache(),
		m:     sync.RWMutex{},
	}

	s.subLogger = zstdlog.NewStdLoggerWithLevel(s.log.With().Logger(), zerolog.TraceLevel)

	return s
}

func (s *Service) List(ctx context.Context) ([]domain.DownloadClient, error) {
	clients, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not list download clients")
		return nil, err
	}

	return clients, nil
}

func (s *Service) FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error) {
	cachedClient := s.cache.Get(id)
	if cachedClient != nil {
		return cachedClient, nil
	}

	s.log.Trace().Int32("client_id", id).Msg("cache miss for client, continue to repo lookup")

	client, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Int32("client_id", id).Msg("could not find download client by id")
		return nil, err
	}

	return client, nil
}

func (s *Service) GetArrTags(ctx context.Context, id int32) ([]*domain.ArrTag, error) {
	data := make([]*domain.ArrTag, 0)

	client, err := s.GetClient(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Int32("client_id", id).Msg("could not find download client")
		return data, nil
	}

	switch client.Type {
	case domain.DownloadClientTypeRadarr:
		arrClient := client.Client.(*radarr.Client)
		tags, err := arrClient.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from radarr")
			return data, nil
		}

		for _, tag := range tags {
			emt := &domain.ArrTag{
				ID:    tag.ID,
				Label: tag.Label,
			}
			data = append(data, emt)
		}

		return data, nil

	case domain.DownloadClientTypeSonarr:
		arrClient := client.Client.(*sonarr.Client)
		tags, err := arrClient.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from sonarr")
			return data, nil
		}

		for _, tag := range tags {
			emt := &domain.ArrTag{
				ID:    tag.ID,
				Label: tag.Label,
			}
			data = append(data, emt)
		}

		return data, nil

	case domain.DownloadClientTypeWhisparr, domain.DownloadClientTypeWhisparrV3:
		arrClient := client.Client.(*whisparr.Client)
		tags, err := arrClient.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from whisparr")
			return data, nil
		}

		for _, tag := range tags {
			emt := &domain.ArrTag{
				ID:    tag.ID,
				Label: tag.Label,
			}
			data = append(data, emt)
		}

		return data, nil

	case domain.DownloadClientTypeSportarr:
		arrClient := client.Client.(*sportarr.Client)
		tags, err := arrClient.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Int32("client_id", id).Msg("could not get tags from sportarr")
			return data, nil
		}

		for _, tag := range tags {
			emt := &domain.ArrTag{
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

func (s *Service) Store(ctx context.Context, client *domain.DownloadClient) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	// store
	err := s.repo.Store(ctx, client)
	if err != nil {
		s.log.Error().Err(err).Interface("client", client).Msg("could not store download client")
		return err
	}

	s.cache.Set(client.ID, client)

	return err
}

func (s *Service) Update(ctx context.Context, client *domain.DownloadClient) error {
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

	s.cache.Set(client.ID, client)

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

	s.cache.Pop(clientID)

	return nil
}

func (s *Service) Test(ctx context.Context, client domain.DownloadClient) error {
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

// GetClient get client from cache or repo and attach downloadClient implementation
func (s *Service) GetClient(ctx context.Context, clientId int32) (*domain.DownloadClient, error) {
	l := s.log.With().Str("cache", "download-client").Int32("client_id", clientId).Logger()

	client := s.cache.Get(clientId)
	if client == nil {
		l.Trace().Msg("cache miss for client, continue to repo lookup")

		var err error
		client, err = s.repo.FindByID(ctx, clientId)
		if err != nil {
			return nil, errors.Wrap(err, "could not find client repo.FindByID")
		}
	}

	// if we have the client return it
	if client.Client != nil {
		l.Trace().Str("client", client.Name).Msg("cache hit for client")
		return client, nil
	}

	l.Trace().Str("client", client.Name).Msg("init cache client")

	switch client.Type {
	case domain.DownloadClientTypeQbittorrent:
		clientHost, err := client.BuildLegacyHost()
		if err != nil {
			return nil, errors.Wrap(err, "error building qBittorrent host url: %v", client.Host)
		}

		client.Client = qbittorrent.NewClient(qbittorrent.Config{
			Host:          clientHost,
			Username:      client.Username,
			Password:      client.Password,
			APIKey:        client.Settings.APIKey,
			TLSSkipVerify: client.TLSSkipVerify,
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "qBittorrent").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			BasicUser:     client.Settings.Auth.Username,
			BasicPass:     client.Settings.Auth.Password,
		})

	case domain.DownloadClientTypePorla:
		client.Client = porla.NewClient(porla.Config{
			Hostname:      client.Host,
			AuthToken:     client.Settings.APIKey,
			TLSSkipVerify: client.TLSSkipVerify,
			BasicUser:     client.Settings.Auth.Username,
			BasicPass:     client.Settings.Auth.Password,
			Log:           s.log.With().Str("type", "Porla").Str("client", client.Name).Logger(),
		})

	case domain.DownloadClientTypeAria2:
		ar, err := aria2.NewClient(aria2.Config{
			Host:          client.Host,
			Secret:        client.Settings.APIKey,
			TLSSkipVerify: client.TLSSkipVerify,
			BasicUser:     client.Settings.Auth.Username,
			BasicPass:     client.Settings.Auth.Password,
			Log:           s.log.With().Str("type", "Aria2").Str("client", client.Name).Logger(),
		})
		if err != nil {
			return nil, errors.Wrap(err, "error creating aria2 client: %s", client.Host)
		}
		client.Client = ar

	case domain.DownloadClientTypeDelugeV1:
		client.Client = deluge.NewV1(deluge.Settings{
			Hostname:             client.Host,
			Port:                 uint(client.Port),
			Login:                client.Username,
			Password:             client.Password,
			DebugServerResponses: true,
			ReadWriteTimeout:     time.Second * 60,
		})

	case domain.DownloadClientTypeDelugeV2:
		client.Client = deluge.NewV2(deluge.Settings{
			Hostname:             client.Host,
			Port:                 uint(client.Port),
			Login:                client.Username,
			Password:             client.Password,
			DebugServerResponses: true,
			ReadWriteTimeout:     time.Second * 60,
		})

	case domain.DownloadClientTypeTransmission:
		clientHost, err := client.BuildLegacyHost()
		if err != nil {
			return nil, errors.Wrap(err, "error building Transmission host url: %v", client.Host)
		}

		transmissionURL, err := url.Parse(clientHost)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse transmission url")
		}

		tbt, err := transmission.New(transmissionURL, &transmission.Config{
			UserAgent:     "autobrr",
			Username:      client.Username,
			Password:      client.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})
		if err != nil {
			return nil, errors.Wrap(err, "error logging into transmission client: %s", client.Host)
		}
		client.Client = tbt

	case domain.DownloadClientTypeRTorrent:
		if client.Settings.Auth.Type == domain.DownloadClientAuthTypeDigest {
			cfg := rtorrent.Config{
				Addr:          client.Host,
				TLSSkipVerify: client.TLSSkipVerify,
				BasicUser:     client.Settings.Auth.Username,
				BasicPass:     client.Settings.Auth.Password,
				Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "rTorrent").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			}

			httpClient := &http.Client{
				Transport: &digest.Transport{
					Username: client.Settings.Auth.Username,
					Password: client.Settings.Auth.Password,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: client.TLSSkipVerify},
					},
				},
			}

			// override client
			client.Client = rtorrent.NewClientWithOpts(cfg, rtorrent.WithCustomClient(httpClient))

		} else {
			client.Client = rtorrent.NewClient(rtorrent.Config{
				Addr:          client.Host,
				TLSSkipVerify: client.TLSSkipVerify,
				BasicUser:     client.Settings.Auth.Username,
				BasicPass:     client.Settings.Auth.Password,
				Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "rTorrent").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			})
		}

	case domain.DownloadClientTypeLidarr:
		client.Client = lidarr.New(lidarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           s.log.With().Str("type", "Lidarr").Str("client", client.Name).Logger(),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeRadarr:
		client.Client = radarr.New(radarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           s.log.With().Str("type", "Radarr").Str("client", client.Name).Logger(),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeReadarr:
		client.Client = readarr.New(readarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           s.log.With().Str("type", "Readarr").Str("client", client.Name).Logger(),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeSonarr:
		client.Client = sonarr.New(sonarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           s.log.With().Str("type", "Sonarr").Str("client", client.Name).Logger(),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeWhisparr, domain.DownloadClientTypeWhisparrV3:
		client.Client = whisparr.New(whisparr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Version:       whisparrVersion(client.Type),
			Log:           s.log.With().Str("type", "Whisparr").Str("client", client.Name).Logger(),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeSportarr:
		client.Client = sportarr.New(sportarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           s.log.With().Str("type", "Sportarr").Str("client", client.Name).Logger(),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeChaptarr:
		client.Client = chaptarr.New(chaptarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           s.log.With().Str("type", "Chaptarr").Str("client", client.Name).Logger(),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeSabnzbd:
		client.Client = sabnzbd.New(sabnzbd.Options{
			Addr:      client.Host,
			ApiKey:    client.Settings.APIKey,
			Log:       s.log.With().Str("type", "Sabnzbd").Str("client", client.Name).Logger(),
			BasicUser: client.Settings.Auth.Username,
			BasicPass: client.Settings.Auth.Password,
		})

	case domain.DownloadClientTypeNzbget:
		client.Client = nzbget.New(nzbget.Options{
			Host:     client.Host,
			Username: client.Username,
			Password: client.Password,
			Log:      s.log.With().Str("type", "Nzbget").Str("client", client.Name).Logger(),
		})
	}

	l.Trace().Str("client", client.Name).Msg("set cache client")

	s.cache.Set(clientId, client)

	return client, nil
}
