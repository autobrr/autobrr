package download_client

import (
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/lidarr"
	"github.com/autobrr/autobrr/pkg/qbittorrent"
	"github.com/autobrr/autobrr/pkg/radarr"
	"github.com/autobrr/autobrr/pkg/sonarr"
	"github.com/autobrr/autobrr/pkg/whisparr"

	delugeClient "github.com/gdm85/go-libdeluge"
)

func (s *service) testConnection(client domain.DownloadClient) error {
	switch client.Type {
	case domain.DownloadClientTypeQbittorrent:
		return s.testQbittorrentConnection(client)

	case domain.DownloadClientTypeDelugeV1, domain.DownloadClientTypeDelugeV2:
		return s.testDelugeConnection(client)

	case domain.DownloadClientTypeRadarr:
		return s.testRadarrConnection(client)

	case domain.DownloadClientTypeSonarr:
		return s.testSonarrConnection(client)

	case domain.DownloadClientTypeLidarr:
		return s.testLidarrConnection(client)

	case domain.DownloadClientTypeWhisparr:
		return s.testWhisparrConnection(client)
	default:
		return errors.New("unsupported client")
	}
}

func (s *service) testQbittorrentConnection(client domain.DownloadClient) error {
	qbtSettings := qbittorrent.Settings{
		Hostname:      client.Host,
		Port:          uint(client.Port),
		Username:      client.Username,
		Password:      client.Password,
		TLS:           client.TLS,
		TLSSkipVerify: client.TLSSkipVerify,
		Log:           s.subLogger,
	}

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		qbtSettings.BasicAuth = client.Settings.Basic.Auth
		qbtSettings.Basic.Username = client.Settings.Basic.Username
		qbtSettings.Basic.Password = client.Settings.Basic.Password
	}

	qbt := qbittorrent.NewClient(qbtSettings)
	err := qbt.Login()
	if err != nil {
		return errors.Wrap(err, "error logging into client: %v", client.Host)
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

func (s *service) testRadarrConnection(client domain.DownloadClient) error {
	r := radarr.New(radarr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	_, err := r.Test()
	if err != nil {
		return errors.Wrap(err, "radarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for Radarr: success")

	return nil
}

func (s *service) testSonarrConnection(client domain.DownloadClient) error {
	r := sonarr.New(sonarr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	_, err := r.Test()
	if err != nil {
		return errors.Wrap(err, "sonarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for Sonarr: success")

	return nil
}

func (s *service) testLidarrConnection(client domain.DownloadClient) error {
	r := lidarr.New(lidarr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	_, err := r.Test()
	if err != nil {
		return errors.Wrap(err, "lidarr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for Lidarr: success")

	return nil
}

func (s *service) testWhisparrConnection(client domain.DownloadClient) error {
	r := whisparr.New(whisparr.Config{
		Hostname:  client.Host,
		APIKey:    client.Settings.APIKey,
		BasicAuth: client.Settings.Basic.Auth,
		Username:  client.Settings.Basic.Username,
		Password:  client.Settings.Basic.Password,
		Log:       s.subLogger,
	})

	_, err := r.Test()
	if err != nil {
		return errors.Wrap(err, "whisparr: connection test failed: %v", client.Host)
	}

	s.log.Debug().Msgf("test client connection for whisparr: success")

	return nil
}
