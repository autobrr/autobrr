// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package downloader

import (
	"context"

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

	"github.com/autobrr/go-deluge"
	"github.com/autobrr/go-qbittorrent"
	"github.com/autobrr/go-rtorrent"
	"github.com/hekmon/transmissionrpc/v3"
)

func (s *Service) testConnection(ctx context.Context, cfg *domain.Downloader) error {
	instance, err := s.initInstance(cfg)
	if err != nil {
		return err
	}

	switch cfg.Type {
	case domain.DownloaderTypeQbittorrent:
		return s.testConnectionQbittorrent(ctx, instance)

	case domain.DownloaderTypeDelugeV1, domain.DownloaderTypeDelugeV2:
		return s.testConnectionDeluge(ctx, instance)

	case domain.DownloaderTypeRTorrent:
		return s.testConnectionRTorrent(ctx, instance)

	case domain.DownloaderTypeTransmission:
		return s.testConnectionTransmission(ctx, instance)

	case domain.DownloaderTypePorla:
		return s.testConnectionPorla(ctx, instance)

	case domain.DownloaderTypeAria2:
		return s.testConnectionAria2(ctx, instance)

	case domain.DownloaderTypeRadarr:
		return s.testConnectionRadarr(ctx, instance)

	case domain.DownloaderTypeSonarr:
		return s.testConnectionSonarr(ctx, instance)

	case domain.DownloaderTypeLidarr:
		return s.testConnectionLidarr(ctx, instance)

	case domain.DownloaderTypeWhisparr, domain.DownloaderTypeWhisparrV3:
		return s.testConnectionWhisparr(ctx, instance)

	case domain.DownloaderTypeReadarr:
		return s.testConnectionReadarr(ctx, instance)

	case domain.DownloaderTypeSportarr:
		return s.testConnectionSportarr(ctx, instance)

	case domain.DownloaderTypeSabnzbd:
		return s.testConnectionSabnzbd(ctx, instance)

	case domain.DownloaderTypeNzbget:
		return s.testConnectionNzbget(ctx, instance)

	default:
		return errors.New("unsupported client: %s", cfg.Type)
	}
}

func (s *Service) testConnectionQbittorrent(ctx context.Context, instance *Instance) error {
	client, err := ClientAs[*qbittorrent.Client](instance)
	if err != nil {
		return err
	}

	cfg := instance.Config()

	if err := client.LoginCtx(ctx); err != nil {
		return errors.Wrap(err, "error logging into client: %v", cfg.Host)
	}

	if _, err := client.GetTorrentsCtx(ctx, qbittorrent.TorrentFilterOptions{Filter: qbittorrent.TorrentFilterAll}); err != nil {
		return errors.Wrap(err, "error getting torrents: %v", cfg.Host)
	}

	s.log.Debug().Msg("test client connection for qBittorrent: success")

	return nil
}

func (s *Service) testConnectionDeluge(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	switch cfg.Type {
	case domain.DownloaderTypeDelugeV1:
		client, err := ClientAs[*deluge.Client](instance)
		if err != nil {
			return err
		}

		// perform connection to Deluge server
		if err := client.Connect(ctx); err != nil {
			return errors.Wrap(err, "error logging into client: %v", cfg.Host)
		}
		defer client.Close()

		// print daemon version
		version, err := client.DaemonVersion(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get daemon version: %v", cfg.Host)
		}
		s.log.Debug().Str("version", version).Msg("test client connection for Deluge: success")

	case domain.DownloaderTypeDelugeV2:
		client, err := ClientAs[*deluge.ClientV2](instance)
		if err != nil {
			return err
		}

		// perform connection to Deluge server
		if err := client.Connect(ctx); err != nil {
			return errors.Wrap(err, "error logging into client: %v", cfg.Host)
		}
		defer client.Close()

		// print daemon version
		version, err := client.DaemonVersion(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get daemon version: %v", cfg.Host)
		}
		s.log.Debug().Str("version", version).Msg("test client connection for Deluge: success")

	default:
		return errors.New("unsupported deluge client version: %s", cfg.Type)
	}

	return nil
}

func (s *Service) testConnectionRTorrent(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*rtorrent.Client](instance)
	if err != nil {
		return err
	}

	name, err := client.Name(ctx)
	if err != nil {
		return errors.Wrap(err, "error logging into client: %s", cfg.Host)
	}

	s.log.Debug().Str("name", name).Msg("test client connection for rTorrent: success")

	return nil
}

func (s *Service) testConnectionTransmission(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*transmissionrpc.Client](instance)
	if err != nil {
		return err
	}

	ok, version, _, err := client.RPCVersion(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting rpc info: %v", cfg.Host)
	}

	if !ok {
		return errors.Wrap(err, "error getting rpc info: %v", cfg.Host)
	}

	s.log.Debug().Int64("version", version).Msg("test client connection for Transmission: success")

	return nil
}

func (s *Service) testConnectionRadarr(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*radarr.Client](instance)
	if err != nil {
		return err
	}

	if _, err := client.Test(ctx); err != nil {
		return errors.Wrap(err, "radarr: connection test failed: %v", cfg.Host)
	}

	s.log.Debug().Msg("test client connection for Radarr: success")

	return nil
}

func (s *Service) testConnectionSportarr(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*sportarr.Client](instance)
	if err != nil {
		return err
	}

	status, err := client.Test(ctx)
	if err != nil {
		return errors.Wrap(err, "sportarr: connection test failed: %v", cfg.Host)
	}

	// The native release/push route ships with 4.0.1024; older versions
	// answer system/status fine but 404 the push, so fail the test early
	// with an actionable message instead of failing on the first release.
	if !status.SupportsNativeAPI() {
		return errors.New("sportarr: version %s is too old, autobrr needs Sportarr %s or newer", status.Version, sportarr.MinimumVersion)
	}

	s.log.Debug().Str("version", status.Version).Msg("test client connection for Sportarr: success")

	return nil
}

func (s *Service) testConnectionSonarr(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*sonarr.Client](instance)
	if err != nil {
		return err
	}

	if _, err := client.Test(ctx); err != nil {
		return errors.Wrap(err, "sonarr: connection test failed: %v", cfg.Host)
	}

	s.log.Debug().Msg("test client connection for Sonarr: success")

	return nil
}

func (s *Service) testConnectionLidarr(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*lidarr.Client](instance)
	if err != nil {
		return err
	}

	if _, err := client.Test(ctx); err != nil {
		return errors.Wrap(err, "lidarr: connection test failed: %v", cfg.Host)
	}

	s.log.Debug().Msg("test client connection for Lidarr: success")

	return nil
}

// whisparrVersion maps the client type to the Whisparr major version it talks
// to. v2 is Sonarr based and serves series, v3 is Radarr based and serves
// movies, so the two are separate client types.
func whisparrVersion(clientType domain.DownloaderType) int {
	if clientType == domain.DownloaderTypeWhisparrV3 {
		return whisparr.VersionV3
	}

	return whisparr.VersionV2
}

func (s *Service) testConnectionWhisparr(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*whisparr.Client](instance)
	if err != nil {
		return err
	}

	status, err := client.Test(ctx)
	if err != nil {
		return errors.Wrap(err, "whisparr: connection test failed: %v", cfg.Host)
	}

	s.log.Debug().Str("version", status.Version).Msg("test client connection for whisparr: success")

	return nil
}

func (s *Service) testConnectionReadarr(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*readarr.Client](instance)
	if err != nil {
		return err
	}

	if _, err := client.Test(ctx); err != nil {
		return errors.Wrap(err, "readarr: connection test failed: %v", cfg.Host)
	}

	s.log.Debug().Msg("test client connection for readarr: success")

	return nil
}

func (s *Service) testConnectionPorla(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*porla.Client](instance)
	if err != nil {
		return err
	}

	version, err := client.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "porla: failed to get version: %v", cfg.Host)
	}

	commitHash := version.Commitish

	if len(commitHash) > 8 {
		commitHash = commitHash[:8]
	}

	s.log.Debug().Str("version", version.Version).Str("commit", commitHash).Msg("test client connection for porla: success")

	return nil
}

func (s *Service) testConnectionAria2(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*aria2.Client](instance)
	if err != nil {
		return err
	}

	version, err := client.GetVersion(ctx)
	if err != nil {
		return errors.Wrap(err, "aria2: failed to get version: %s", cfg.Host)
	}

	s.log.Debug().Str("version", version.Version).Msg("test client connection for aria2: success")

	return nil
}

func (s *Service) testConnectionSabnzbd(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*sabnzbd.Client](instance)
	if err != nil {
		return err
	}

	version, err := client.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting version from sabnzbd: %s", cfg.Host)
	}

	s.log.Debug().Str("version", version.Version).Msg("test client connection for sabnzbd: success")

	return nil
}

func (s *Service) testConnectionNzbget(ctx context.Context, instance *Instance) error {
	cfg := instance.Config()

	client, err := ClientAs[*nzbget.Client](instance)
	if err != nil {
		return err
	}

	version, err := client.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting version from nzbget: %s", cfg.Host)
	}

	s.log.Debug().Str("version", version).Msg("test client connection for nzbget: success")

	return nil
}
