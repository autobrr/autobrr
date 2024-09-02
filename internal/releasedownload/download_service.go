package releasedownload

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog"
	"golang.org/x/net/publicsuffix"
)

type DownloadService struct {
	log  zerolog.Logger
	repo domain.ReleaseRepo

	indexerRepo domain.IndexerRepo

	proxySvc proxy.Service
}

func NewDownloadService(log logger.Logger, repo domain.ReleaseRepo, indexerRepo domain.IndexerRepo, proxySvc proxy.Service) *DownloadService {
	return &DownloadService{
		log:         log.With().Str("module", "release-download").Logger(),
		repo:        repo,
		indexerRepo: indexerRepo,
		proxySvc:    proxySvc,
	}
}

func (s *DownloadService) DownloadRelease(ctx context.Context, rls *domain.Release) error {
	if rls.HasMagnetUri() {
		return errors.New("downloading magnet links is not supported: %s", rls.MagnetURI)
	} else if rls.Protocol != domain.ReleaseProtocolTorrent {
		return errors.New("could not download file: protocol %s is not supported", rls.Protocol)
	}

	if rls.DownloadURL == "" {
		return errors.New("download_file: url can't be empty")
	} else if rls.TorrentTmpFile != "" {
		// already downloaded
		return nil
	}

	// get indexer
	indexer, err := s.indexerRepo.FindByID(ctx, rls.Indexer.ID)
	if err != nil {
		return err
	}

	// get proxy
	if indexer.UseProxy {
		proxyConf, err := s.proxySvc.FindByID(ctx, indexer.ProxyID)
		if err != nil {
			return err
		}

		if proxyConf.Enabled {
			s.log.Debug().Msgf("using proxy: %s", proxyConf.Name)

			indexer.Proxy = proxyConf
		} else {
			s.log.Debug().Msgf("proxy disabled, skip: %s", proxyConf.Name)
		}
	}

	// download release
	err = s.downloadTorrentFile(ctx, indexer, rls)
	if err != nil {
		return err
	}

	return nil
}

func (s *DownloadService) downloadTorrentFile(ctx context.Context, indexer *domain.Indexer, r *domain.Release) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.DownloadURL, nil)
	if err != nil {
		return errors.Wrap(err, "error downloading file")
	}

	req.Header.Set("User-Agent", "autobrr")

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: sharedhttp.TransportTLSInsecure,
	}

	// handle proxy
	if indexer.Proxy != nil {
		s.log.Debug().Msgf("using proxy: %s", indexer.Proxy.Name)

		proxiedClient, err := proxy.GetProxiedHTTPClient(indexer.Proxy)
		if err != nil {
			return errors.Wrap(err, "could not get proxied http client")
		}

		httpClient = proxiedClient
	}

	if r.RawCookie != "" {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return errors.Wrap(err, "could not create cookiejar")
		}
		httpClient.Jar = jar

		// set the cookie on the header instead of req.AddCookie
		// since we have a raw cookie like "uid=10; pass=000"
		req.Header.Set("Cookie", r.RawCookie)
	}

	tmpFilePattern := "autobrr-"
	tmpDir := os.TempDir()

	// Create tmp file
	// TODO check if tmp file is wanted
	tmpFile, err := os.CreateTemp(tmpDir, tmpFilePattern)
	if err != nil {
		if os.IsNotExist(err) {
			if mkdirErr := os.MkdirAll(tmpDir, os.ModePerm); mkdirErr != nil {
				return errors.Wrap(mkdirErr, "could not create TMP dir: %s", tmpDir)
			}

			tmpFile, err = os.CreateTemp(tmpDir, tmpFilePattern)
			if err != nil {
				return errors.Wrap(err, "error creating tmp file in: %s", tmpDir)
			}
		} else {
			return errors.Wrap(err, "error creating tmp file")
		}
	}
	defer tmpFile.Close()

	errFunc := retry.Do(retryableRequest(httpClient, req, r, tmpFile), retry.Delay(time.Second*3), retry.Attempts(3), retry.MaxJitter(time.Second*1))

	return errFunc
}

func retryableRequest(httpClient *http.Client, req *http.Request, r *domain.Release, tmpFile *os.File) func() error {
	return func() error {
		// Get the data
		resp, err := httpClient.Do(req)
		if err != nil {
			if errors.As(err, net.OpError{}) {
				return retry.Unrecoverable(errors.Wrap(err, "issue from proxy"))
			}
			return errors.Wrap(err, "error downloading file")
		}
		defer resp.Body.Close()

		// Check server response
		switch resp.StatusCode {
		case http.StatusOK:
			// Continue processing the response
			break

		//case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		//	// Handle redirect
		//	return retry.Unrecoverable(errors.New("redirect encountered for torrent (%s) file (%s) - status code: %d - check indexer keys for %s", r.TorrentName, r.DownloadURL, resp.StatusCode, r.Indexer.Name))

		case http.StatusUnauthorized, http.StatusForbidden:
			return retry.Unrecoverable(errors.New("unrecoverable error downloading torrent (%s) file (%s) - status code: %d - check indexer keys for %s", r.TorrentName, r.DownloadURL, resp.StatusCode, r.Indexer.Name))

		case http.StatusMethodNotAllowed:
			return retry.Unrecoverable(errors.New("unrecoverable error downloading torrent (%s) file (%s) from '%s' - status code: %d. Check if the request method is correct", r.TorrentName, r.DownloadURL, r.Indexer.Name, resp.StatusCode))
		case http.StatusNotFound:
			return errors.New("torrent %s not found on %s (%d) - retrying", r.TorrentName, r.Indexer.Name, resp.StatusCode)

		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return errors.New("server error (%d) encountered while downloading torrent (%s) file (%s) from '%s' - retrying", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name)

		case http.StatusInternalServerError:
			return errors.New("server error (%d) encountered while downloading torrent (%s) file (%s) - check indexer keys for %s", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name)

		default:
			return retry.Unrecoverable(errors.New("unexpected status code %d: check indexer keys for %s", resp.StatusCode, r.Indexer.Name))
		}

		resetTmpFile := func() {
			tmpFile.Seek(0, io.SeekStart)
			tmpFile.Truncate(0)
		}

		// Read the body into bytes
		bodyBytes, err := io.ReadAll(bufio.NewReader(resp.Body))
		if err != nil {
			return errors.Wrap(err, "error reading response body")
		}

		// Create a new reader for bodyBytes
		bodyReader := bytes.NewReader(bodyBytes)

		// Try to decode as torrent file
		meta, err := metainfo.Load(bodyReader)
		if err != nil {
			resetTmpFile()

			// explicitly check for unexpected content type that match html
			var bse *bencode.SyntaxError
			if errors.As(err, &bse) {
				// regular error so we can retry if we receive html first run
				return errors.Wrap(err, "metainfo unexpected content type, got HTML expected a bencoded torrent. check indexer keys for %s - %s", r.Indexer.Name, r.TorrentName)
			}

			return retry.Unrecoverable(errors.Wrap(err, "metainfo unexpected content type. check indexer keys for %s - %s", r.Indexer.Name, r.TorrentName))
		}

		torrentMetaInfo, err := meta.UnmarshalInfo()
		if err != nil {
			resetTmpFile()
			return retry.Unrecoverable(errors.Wrap(err, "metainfo could not unmarshal info from torrent: %s", tmpFile.Name()))
		}

		hashInfoBytes := meta.HashInfoBytes().Bytes()
		if len(hashInfoBytes) < 1 {
			resetTmpFile()
			return retry.Unrecoverable(errors.New("could not read infohash"))
		}

		// Write the body to file
		// TODO move to io.Reader and pass around in the future
		if _, err := tmpFile.Write(bodyBytes); err != nil {
			resetTmpFile()
			return errors.Wrap(err, "error writing downloaded file: %s", tmpFile.Name())
		}

		r.TorrentTmpFile = tmpFile.Name()
		r.TorrentHash = meta.HashInfoBytes().String()
		r.Size = uint64(torrentMetaInfo.TotalLength())

		return nil
	}
}

func (s *DownloadService) ResolveMagnetURI(ctx context.Context, r *domain.Release) error {
	if r.MagnetURI == "" {
		return nil
	} else if strings.HasPrefix(r.MagnetURI, domain.MagnetURIPrefix) {
		return nil
	}

	// get indexer
	indexer, err := s.indexerRepo.FindByID(ctx, r.Indexer.ID)
	if err != nil {
		return err
	}

	httpClient := &http.Client{
		Timeout:   time.Second * 45,
		Transport: sharedhttp.MagnetTransport,
	}

	// get proxy
	if indexer.UseProxy {
		proxyConf, err := s.proxySvc.FindByID(ctx, indexer.ProxyID)
		if err != nil {
			return err
		}

		s.log.Debug().Msgf("using proxy: %s", proxyConf.Name)

		proxiedClient, err := proxy.GetProxiedHTTPClient(proxyConf)
		if err != nil {
			return errors.Wrap(err, "could not get proxied http client")
		}

		httpClient = proxiedClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.MagnetURI, nil)
	if err != nil {
		return errors.Wrap(err, "could not build request to resolve magnet uri")
	}

	//req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	res, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request to resolve magnet uri")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "could not read response body")
	}

	magnet := string(body)
	if magnet != "" {
		r.MagnetURI = magnet
	}

	return nil
}
