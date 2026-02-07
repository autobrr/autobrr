// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package download_client

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/arr/lidarr"
	"github.com/autobrr/autobrr/pkg/arr/radarr"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/porla"
	"github.com/autobrr/autobrr/pkg/sabnzbd"
	"github.com/autobrr/autobrr/pkg/transmission"
	"github.com/autobrr/autobrr/pkg/whisparr"

	"github.com/autobrr/go-deluge"
	"github.com/autobrr/go-qbittorrent"
	"github.com/autobrr/go-rtorrent"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/icholy/digest"
	"github.com/rs/zerolog"
)

type Service interface {
	List(ctx context.Context) ([]domain.DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error)
	Store(ctx context.Context, client *domain.DownloadClient) error
	Update(ctx context.Context, client *domain.DownloadClient) error
	Delete(ctx context.Context, clientID int32) error
	Test(ctx context.Context, client domain.DownloadClient) error

	GetArrTags(ctx context.Context, id int32) ([]*domain.ArrTag, error)
	GetClient(ctx context.Context, clientId int32) (*domain.DownloadClient, error)
}

type service struct {
	log       zerolog.Logger
	repo      domain.DownloadClientRepo
	subLogger *log.Logger

	cache *ClientCache
	m     sync.RWMutex
}

func NewService(log logger.Logger, repo domain.DownloadClientRepo) Service {
	s := &service{
		log:  log.With().Str("module", "download_client").Logger(),
		repo: repo,

		cache: NewClientCache(),
		m:     sync.RWMutex{},
	}

	s.subLogger = zstdlog.NewStdLoggerWithLevel(s.log.With().Logger(), zerolog.TraceLevel)

	return s
}

func (s *service) List(ctx context.Context) ([]domain.DownloadClient, error) {
	clients, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not list download clients")
		return nil, err
	}

	return clients, nil
}

func (s *service) FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error) {
	client := s.cache.Get(id)
	if client != nil {
		return client, nil
	}

	s.log.Trace().Msgf("cache miss for client id %d, continue to repo lookup", id)

	client, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find download client by id: %v", id)
		return nil, err
	}

	return client, nil
}

func (s *service) GetArrTags(ctx context.Context, id int32) ([]*domain.ArrTag, error) {
	data := make([]*domain.ArrTag, 0)

	client, err := s.GetClient(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find download client by id: %v", id)
		return data, nil
	}

	switch client.Type {
	case "RADARR":
		arrClient := client.Client.(*radarr.Client)
		tags, err := arrClient.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not get tags from radarr: %v", id)
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

	case "SONARR":
		arrClient := client.Client.(*sonarr.Client)
		tags, err := arrClient.GetTags(ctx)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not get tags from sonarr: %v", id)
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

func (s *service) Store(ctx context.Context, client *domain.DownloadClient) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	// store
	err := s.repo.Store(ctx, client)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store download client: %+v", client)
		return err
	}

	s.cache.Set(client.ID, client)

	return err
}

func (s *service) Update(ctx context.Context, client *domain.DownloadClient) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	existingClient, err := s.FindByID(ctx, client.ID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find download client by id: %v", client.ID)
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
		s.log.Error().Err(err).Msgf("could not update download client: %+v", client)
		return err
	}

	s.cache.Set(client.ID, client)

	return err
}

func (s *service) Delete(ctx context.Context, clientID int32) error {
	if err := s.repo.Delete(ctx, clientID); err != nil {
		s.log.Error().Err(err).Msgf("could not delete download client: %v", clientID)
		return err
	}

	s.cache.Pop(clientID)

	return nil
}

func (s *service) Test(ctx context.Context, client domain.DownloadClient) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	// check for existing client to get settings from
	if client.ID > 0 {
		existingClient, err := s.FindByID(ctx, client.ID)
		if err != nil {
			s.log.Error().Err(err).Msgf("could not find download client by id: %v", client.ID)
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
func (s *service) GetClient(ctx context.Context, clientId int32) (*domain.DownloadClient, error) {
	l := s.log.With().Str("cache", "download-client").Logger()

	client := s.cache.Get(clientId)
	if client == nil {
		l.Trace().Msgf("cache miss for client id %d, continue to repo lookup", clientId)

		var err error
		client, err = s.repo.FindByID(ctx, clientId)
		if err != nil {
			return nil, errors.Wrap(err, "could not find client repo.FindByID")
		}
	}

	// if we have the client return it
	if client.Client != nil {
		l.Trace().Msgf("cache hit for client id %d %s", clientId, client.Name)
		return client, nil
	}

	l.Trace().Msgf("init cache client id %d %s", clientId, client.Name)

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
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "Porla").Str("client", client.Name).Logger(), zerolog.TraceLevel),
		})

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
		scheme := "http"
		if client.TLS {
			scheme = "https"
		}

		transmissionURL, err := url.Parse(fmt.Sprintf("%s://%s:%d/transmission/rpc", scheme, client.Host, client.Port))
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
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "Lidarr").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeRadarr:
		client.Client = radarr.New(radarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "Radarr").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeReadarr:
		client.Client = readarr.New(readarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "Readarr").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeSonarr:
		client.Client = sonarr.New(sonarr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "Sonarr").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeWhisparr:
		client.Client = whisparr.New(whisparr.Config{
			Hostname:      client.Host,
			APIKey:        client.Settings.APIKey,
			Log:           zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "Whisparr").Str("client", client.Name).Logger(), zerolog.TraceLevel),
			BasicAuth:     client.Settings.Auth.Enabled,
			Username:      client.Settings.Auth.Username,
			Password:      client.Settings.Auth.Password,
			TLSSkipVerify: client.TLSSkipVerify,
		})

	case domain.DownloadClientTypeSabnzbd:
		client.Client = sabnzbd.New(sabnzbd.Options{
			Addr:      client.Host,
			ApiKey:    client.Settings.APIKey,
			Log:       nil,
			BasicUser: client.Settings.Auth.Username,
			BasicPass: client.Settings.Auth.Password,
		})
	}

	l.Trace().Msgf("set cache client id %d %s", clientId, client.Name)

	s.cache.Set(clientId, client)

	return client, nil
}
