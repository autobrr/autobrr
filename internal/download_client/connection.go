package download_client

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/lidarr"
	"github.com/autobrr/autobrr/pkg/porla"
	"github.com/autobrr/autobrr/pkg/radarr"
	"github.com/autobrr/autobrr/pkg/readarr"
	"github.com/autobrr/autobrr/pkg/sabnzbd"
	"github.com/autobrr/autobrr/pkg/sonarr"
	"github.com/autobrr/autobrr/pkg/whisparr"
	"github.com/autobrr/go-qbittorrent"

	delugeClient "github.com/gdm85/go-libdeluge"
	"github.com/hekmon/transmissionrpc/v2"
	"github.com/mrobinsn/go-rtorrent/rtorrent"
)

func (s *service) testConnection(ctx context.Context, client domain.DownloadClient) error {
	switch client.Type {
	case domain.DownloadClientTypeQbittorrent:
		return s.testQbittorrentConnection(ctx, client)

	case domain.DownloadClientTypeDelugeV1, domain.DownloadClientTypeDelugeV2:
		return s.testDelugeConnection(client)

	case domain.DownloadClientTypeRTorrent:
		return s.testRTorrentConnection(client)

	case domain.DownloadClientTypeTransmission:
		return s.testTransmissionConnection(ctx, client)

	case domain.DownloadClientTypePorla:
		return s.testPorlaConnection(client)

	case domain.DownloadClientTypeRadarr:
		return s.testRadarrConnection(ctx, client)

	case domain.DownloadClientTypeSonarr:
		return s.testSonarrConnection(ctx, client)

	case domain.DownloadClientTypeLidarr:
		return s.testLidarrConnection(ctx, client)

	case domain.DownloadClientTypeWhisparr:
		return s.testWhisparrConnection(ctx, client)

	case domain.DownloadClientTypeReadarr:
		return s.testReadarrConnection(ctx, client)

	case domain.DownloadClientTypeSabnzbd:
		return s.testSabnzbdConnection(ctx, client)

	default:
		return errors.New("unsupported client")
	}
}

func (s *service) testQbittorrentConnection(ctx context.Context, client domain.DownloadClient) error {
	qbtSettings := qbittorrent.Config{
		Host:          client.BuildLegacyHost(),
		Username:      client.Username,
		Password:      client.Password,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.subLogger,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		qbtSettings.BasicUser = client.Settings.Basic.Username
		qbtSettings.BasicPass = client.Settings.Basic.Password
	}

	qbt := qbittorrent.NewClient(qbtSettings)

	if err := qbt.LoginCtx(ctx); err != nil {
		return errors.Wrap(err, "error logging into client: %v", client.Host)
	}

	if _, err := qbt.GetTorrentsCtx(ctx, qbittorrent.TorrentFilterOptions{Filter: qbittorrent.TorrentFilterAll}); err != nil {
		return errors.Wrap(err, "error getting torrents: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for qBittorrent: success")

	return nil
}

func (s *service) testDelugeConnection(client domain.DownloadClient) error {
	var deluge delugeClient.DelugeClient

	settings := delugeClient.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 10,
	}

	switch client.Type {
	case "DELUGE_V1":
		deluge = delugeClient.NewV1(settings)

	case "DELUGE_V2":
		deluge = delugeClient.NewV2(settings)

	default:
		deluge = delugeClient.NewV2(settings)
	}

	// perform connection to Deluge server
	err := deluge.Connect()
	if err != nil {
		return errors.Wrap(err, "error logging into client: %v", client.Host)
	}

	defer deluge.Close()

	// print daemon version
	ver, err := deluge.DaemonVersion()
	if err != nil {
		return errors.Wrap(err, "could not get daemon version: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for Deluge: success - daemon version: %v", ver)

	return nil
}

func (s *service) testRTorrentConnection(client domain.DownloadClient) error {
	// create client
	rt := rtorrent.New(client.Host, true)
	name, err := rt.Name()
	if err != nil {
		return errors.Wrap(err, "error logging into client: %v", client.Host)
	}

	s.log.Trace().Msgf("test client connection for rTorrent: got client: %v", name)

	s.log.Debug().Msgf("test client connection for rTorrent: success")

	return nil
}

func (s *service) testTransmissionConnection(ctx context.Context, client domain.DownloadClient) error {
	tbt, err := transmissionrpc.New(client.Host, client.Username, client.Password, &transmissionrpc.AdvancedConfig{
		HTTPS: client.TLS,
		Port:  uint16(client.Port),
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

	s.log.Debug().Msgf("test client connection for Transmission: got version: %v", version)

	s.log.Debug().Msgf("test client connection for Transmission: success")

	return nil
}

func (s *service) testRadarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := radarr.New(radarr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "radarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for Radarr: success")

	return nil
}

func (s *service) testSonarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := sonarr.New(sonarr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "sonarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for Sonarr: success")

	return nil
}

func (s *service) testLidarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := lidarr.New(lidarr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "lidarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for Lidarr: success")

	return nil
}

func (s *service) testWhisparrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := whisparr.New(whisparr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "whisparr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for whisparr: success")

	return nil
}

func (s *service) testReadarrConnection(ctx context.Context, client domain.DownloadClient) error {
	r := readarr.New(readarr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	if _, err := r.Test(ctx); err != nil {
		return errors.Wrap(err, "readarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for readarr: success")

	return nil
}

func (s *service) testPorlaConnection(client domain.DownloadClient) error {
	p := porla.NewClient(porla.Config{
		Hostname:  client.Host,
		AuthToken: client.Settings.APIKey,
	})

	version, err := p.Version()

	if err != nil {
		return errors.Wrap(err, "porla: failed to get version: %v", client.Host)
	}

	commitish := version.Commitish

	if len(commitish) > 8 {
		commitish = commitish[:8]
	}

	s.log.Debug().Msgf("test client connection for porla: found version %s (commit %s)", version.Version, commitish)

	return nil
}

func (s *service) testSabnzbdConnection(ctx context.Context, client domain.DownloadClient) error {
	opts := sabnzbd.Options{
		Addr:      client.Host,
		ApiKey:    client.Settings.APIKey,
		BasicUser: client.Settings.Basic.Username,
		BasicPass: client.Settings.Basic.Password,
		Log:       nil,
	}

	sab := sabnzbd.New(opts)
	version, err := sab.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting version from sabnzbd")
	}

	s.log.Debug().Msgf("test client connection for sabnzbd: success got version: %s", version.Version)

	return nil
}
