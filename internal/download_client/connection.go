// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package download_client

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
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

func (s *Service) testConnection(ctx context.Context, client domain.DownloadClient) error {
	switch client.Type {
	case domain.DownloadClientTypeQbittorrent:
		return s.testQbittorrentConnection(ctx, client)

	case domain.DownloadClientTypeDelugeV1, domain.DownloadClientTypeDelugeV2:
		return s.testDelugeConnection(ctx, client)

	case domain.DownloadClientTypeRTorrent:
		return s.testRTorrentConnection(ctx, client)

	case domain.DownloadClientTypeTransmission:
		return s.testTransmissionConnection(ctx, client)

	case domain.DownloadClientTypePorla:
		return s.testPorlaConnection(client)

	case domain.DownloadClientTypeAria2:
		return s.testAria2Connection(ctx, client)

	case domain.DownloadClientTypeRadarr:
		return s.testRadarrConnection(ctx, client)

	case domain.DownloadClientTypeSonarr:
		return s.testSonarrConnection(ctx, client)

	case domain.DownloadClientTypeLidarr:
		return s.testLidarrConnection(ctx, client)

	case domain.DownloadClientTypeWhisparr, domain.DownloadClientTypeWhisparrV3:
		return s.testWhisparrConnection(ctx, client)

	case domain.DownloadClientTypeReadarr:
		return s.testReadarrConnection(ctx, client)

	case domain.DownloadClientTypeSportarr:
		return s.testSportarrConnection(ctx, client)

	case domain.DownloadClientTypeChaptarr:
		return s.testChaptarrConnection(ctx, client)

	case domain.DownloadClientTypeSabnzbd:
		return s.testSabnzbdConnection(ctx, client)

	case domain.DownloadClientTypeNzbget:
		return s.testNzbgetConnection(ctx, client)

	default:
		return errors.New("unsupported client: %s", client.Type)
	}
}

func (s *Service) testQbittorrentConnection(ctx context.Context, client domain.DownloadClient) error {
	clientHost, err := client.BuildLegacyHost()
	if err != nil {
		return errors.Wrap(err, "error building qBittorrent host url: %s", client.Host)
	}

	qbtSettings := qbittorrent.Config{
		Host:          clientHost,
		TLSSkipVerify: client.TLSSkipVerify,
		Username:      client.Username,
		Password:      client.Password,
		APIKey:        client.Settings.APIKey,
		Log:           s.subLogger,
	}

	// only set basic auth if enabled
	if client.Settings.Auth.Enabled {
		qbtSettings.BasicUser = client.Settings.Auth.Username
		qbtSettings.BasicPass = client.Settings.Auth.Password
	}

	qbt := qbittorrent.NewClient(qbtSettings)

	if err := qbt.LoginCtx(ctx); err != nil {
		return errors.Wrap(err, "error logging into client: %v", client.Host)
	}

	if _, err := qbt.GetTorrentsCtx(ctx, qbittorrent.TorrentFilterOptions{Filter: qbittorrent.TorrentFilterAll}); err != nil {
		return errors.Wrap(err, "error getting torrents: %v", client.Host)
	}

	s.log.Debug().Msg("test client connection for qBittorrent: success")

	return nil
}

func (s *Service) testDelugeConnection(ctx context.Context, client domain.DownloadClient) error {
	settings := deluge.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     30 * time.Second,
	}

	settings.Logger = zstdlog.NewStdLoggerWithLevel(s.log.With().Logger(), zerolog.TraceLevel)

	var err error
	var version string

	switch client.Type {
	case "DELUGE_V1":
		del := deluge.NewV1(settings)

		// perform connection to Deluge server
		if err := del.Connect(ctx); err != nil {
			return errors.Wrap(err, "error logging into client: %v", client.Host)
		}

		defer del.Close()

		// print daemon version
		version, err = del.DaemonVersion(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get daemon version: %v", client.Host)
		}

	case "DELUGE_V2":
		del := deluge.NewV2(settings)

		// perform connection to Deluge server
		if err := del.Connect(ctx); err != nil {
			return errors.Wrap(err, "error logging into client: %v", client.Host)
		}

		defer del.Close()

		// print daemon version
		version, err = del.DaemonVersion(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get daemon version: %v", client.Host)
		}

	default:
		return errors.New("unsupported deluge client version: %s", client.Type)
	}

	s.log.Debug().Str("version", version).Msg("test client connection for Deluge: success")

	return nil
}

func (s *Service) testRTorrentConnection(ctx context.Context, client domain.DownloadClient) error {
	cfg := rtorrent.Config{
		Addr:          client.Host,
		TLSSkipVerify: client.TLSSkipVerify,
		BasicUser:     client.Settings.Auth.Username,
		BasicPass:     client.Settings.Auth.Password,
		Log:           s.subLogger,
	}

	// create client
	rt := rtorrent.NewClient(cfg)

	if client.Settings.Auth.Type == domain.DownloadClientAuthTypeDigest {
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
		rt = rtorrent.NewClientWithOpts(cfg, rtorrent.WithCustomClient(httpClient))
	}

	name, err := rt.Name(ctx)
	if err != nil {
		return errors.Wrap(err, "error logging into client: %s", client.Host)
	}

	s.log.Debug().Str("name", name).Msg("test client connection for rTorrent: success")

	return nil
}

func (s *Service) testTransmissionConnection(ctx context.Context, client domain.DownloadClient) error {
	clientHost, err := client.BuildLegacyHost()
	if err != nil {
		return errors.Wrap(err, "error building Transmission host url: %v", client.Host)
	}

	transmissionURL, err := url.Parse(clientHost)
	if err != nil {
		return errors.Wrap(err, "could not parse transmission url")
	}

	tbt, err := transmission.New(transmissionURL, &transmission.Config{
		UserAgent:     "autobrr",
		Username:      client.Username,
		Password:      client.Password,
		TLSSkipVerify: client.TLSSkipVerify,
	})
	if err != nil {
		return errors.Wrap(err, "error logging into client: %v", client.Host)
	}

	ok, version, _, err := tbt.RPCVersion(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting rpc info: %v", client.Host)
	}

	if !ok {
		return errors.Wrap(err, "error getting rpc info: %v", client.Host)
	}

	s.log.Debug().Int64("version", version).Msg("test client connection for Transmission: success")

	return nil
}

func (s *Service) testRadarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := radarr.New(radarr.Config{
		Hostname:      client.Host,
		APIKey:        client.Settings.APIKey,
		BasicAuth:     client.Settings.Auth.Enabled,
		Username:      client.Settings.Auth.Username,
		Password:      client.Settings.Auth.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.log,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "radarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msg("test client connection for Radarr: success")

	return nil
}

func (s *Service) testSportarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := sportarr.New(sportarr.Config{
		Hostname:      client.Host,
		APIKey:        client.Settings.APIKey,
		BasicAuth:     client.Settings.Auth.Enabled,
		Username:      client.Settings.Auth.Username,
		Password:      client.Settings.Auth.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.log,
	})

	status, err := r.Test(ctx)
	if err != nil {
		return errors.Wrap(err, "sportarr: connection test failed: %v", client.Host)
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

func (s *Service) testSonarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := sonarr.New(sonarr.Config{
		Hostname:      client.Host,
		APIKey:        client.Settings.APIKey,
		BasicAuth:     client.Settings.Auth.Enabled,
		Username:      client.Settings.Auth.Username,
		Password:      client.Settings.Auth.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.log,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "sonarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msg("test client connection for Sonarr: success")

	return nil
}

func (s *Service) testLidarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := lidarr.New(lidarr.Config{
		Hostname:      client.Host,
		APIKey:        client.Settings.APIKey,
		BasicAuth:     client.Settings.Auth.Enabled,
		Username:      client.Settings.Auth.Username,
		Password:      client.Settings.Auth.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.log,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "lidarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msg("test client connection for Lidarr: success")

	return nil
}

// whisparrVersion maps the client type to the Whisparr major version it talks
// to. v2 is Sonarr based and serves series, v3 is Radarr based and serves
// movies, so the two are separate client types.
func whisparrVersion(clientType domain.DownloadClientType) int {
	if clientType == domain.DownloadClientTypeWhisparrV3 {
		return whisparr.VersionV3
	}

	return whisparr.VersionV2
}

func (s *Service) testWhisparrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := whisparr.New(whisparr.Config{
		Hostname:      client.Host,
		APIKey:        client.Settings.APIKey,
		Version:       whisparrVersion(client.Type),
		BasicAuth:     client.Settings.Auth.Enabled,
		Username:      client.Settings.Auth.Username,
		Password:      client.Settings.Auth.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.log,
	})

	status, err := r.Test(ctx)
	if err != nil {
		return errors.Wrap(err, "whisparr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Str("version", status.Version).Msg("test client connection for whisparr: success")

	return nil
}

func (s *Service) testReadarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := readarr.New(readarr.Config{
		Hostname:      client.Host,
		APIKey:        client.Settings.APIKey,
		BasicAuth:     client.Settings.Auth.Enabled,
		Username:      client.Settings.Auth.Username,
		Password:      client.Settings.Auth.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.log,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "readarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msg("test client connection for readarr: success")

	return nil
}

func (s *Service) testChaptarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := chaptarr.New(chaptarr.Config{
		Hostname:      client.Host,
		APIKey:        client.Settings.APIKey,
		BasicAuth:     client.Settings.Auth.Enabled,
		Username:      client.Settings.Auth.Username,
		Password:      client.Settings.Auth.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.log,
	})

	status, err := r.Test(ctx)
	if err != nil {
		return errors.Wrap(err, "chaptarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Str("version", status.Version).Msg("test client connection for chaptarr: success")

	return nil
}

func (s *Service) testPorlaConnection(client domain.DownloadClient) error {
	p := porla.NewClient(porla.Config{
		Hostname:      client.Host,
		TLSSkipVerify: client.TLSSkipVerify,
		AuthToken:     client.Settings.APIKey,
		BasicUser:     client.Settings.Auth.Username,
		BasicPass:     client.Settings.Auth.Password,
		Log:           s.log,
	})

	version, err := p.Version()

	if err != nil {
		return errors.Wrap(err, "porla: failed to get version: %v", client.Host)
	}

	commitHash := version.Commitish

	if len(commitHash) > 8 {
		commitHash = commitHash[:8]
	}

	s.log.Debug().Str("version", version.Version).Msg("test client connection for porla: success")

	return nil
}

func (s *Service) testAria2Connection(ctx context.Context, client domain.DownloadClient) error {
	ar, err := aria2.NewClient(aria2.Config{
		Host:          client.Host,
		Secret:        client.Settings.APIKey,
		TLSSkipVerify: client.TLSSkipVerify,
		BasicUser:     client.Settings.Auth.Username,
		BasicPass:     client.Settings.Auth.Password,
		Log:           s.log,
	})
	if err != nil {
		return errors.Wrap(err, "aria2: could not create client: %s", client.Host)
	}

	version, err := ar.GetVersion(ctx)
	if err != nil {
		return errors.Wrap(err, "aria2: failed to get version: %s", client.Host)
	}

	s.log.Debug().Str("version", version.Version).Msg("test client connection for aria2: success")

	return nil
}

func (s *Service) testSabnzbdConnection(ctx context.Context, client domain.DownloadClient) error {
	opts := sabnzbd.Options{
		Addr:      client.Host,
		ApiKey:    client.Settings.APIKey,
		BasicUser: client.Settings.Auth.Username,
		BasicPass: client.Settings.Auth.Password,
		Log:       s.log,
	}

	sab := sabnzbd.New(opts)
	version, err := sab.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting version from sabnzbd")
	}

	s.log.Debug().Str("version", version.Version).Msg("test client connection for sabnzbd: success")

	return nil
}

func (s *Service) testNzbgetConnection(ctx context.Context, client domain.DownloadClient) error {
	nzb := nzbget.New(nzbget.Options{
		Host:     client.Host,
		Username: client.Username,
		Password: client.Password,
		Log:      s.log,
	})

	version, err := nzb.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting version from nzbget")
	}

	s.log.Debug().Str("version", version).Msg("test client connection for nzbget: success")

	return nil
}
