// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package releasedownload

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
	"github.com/autobrr/go-torrent/bencode"
	"github.com/autobrr/go-torrent/metainfo"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog"
	"golang.org/x/net/publicsuffix"
)

// RetriableError is a custom error that contains a positive duration for the next retry
type RetriableError struct {
	Err        error
	RetryAfter time.Duration
}

// Error returns error message and a Retry-After duration
func (e *RetriableError) Error() string {
	return fmt.Sprintf("%s (retry after %v)", e.Err.Error(), e.RetryAfter)
}

var _ error = (*RetriableError)(nil)

type proxyService interface {
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
}

type indexerRepo interface {
	FindByID(ctx context.Context, id int) (*domain.Indexer, error)
}

type DownloadService struct {
	log zerolog.Logger

	indexerRepo indexerRepo
	proxySvc    proxyService
}

func NewDownloadService(log zerolog.Logger, indexerRepo indexerRepo, proxySvc proxyService) *DownloadService {
	return &DownloadService{
		log:         log.With().Str("module", "release-download").Logger(),
		indexerRepo: indexerRepo,
		proxySvc:    proxySvc,
	}
}

// releaseLogger derives from s.log rather than the ctx logger because callers
// live in both the action and filter packages, and inheriting their logger
// would report the wrong module.
func (s *DownloadService) releaseLogger(r *domain.Release) *zerolog.Logger {
	l := s.log.With().
		Str("trace_id", r.TraceID).
		Str("release", r.TorrentName).
		Str("indexer", r.Indexer.Identifier).
		Logger()

	return &l
}

func (s *DownloadService) DownloadRelease(ctx context.Context, rls *domain.Release) error {
	if rls.HasMagnetUri() {
		return errors.New("downloading magnet links is not supported: %s", rls.MagnetURI)
	} else if rls.Protocol != domain.ReleaseProtocolTorrent {
		return errors.New("could not download file: protocol %s is not supported", rls.Protocol)
	}

	if rls.DownloadURL == "" {
		return errors.New("download_file: url can't be empty")
	} else if len(rls.TorrentDataRawBytes) != 0 {
		// already downloaded
		return nil
	}

	l := s.releaseLogger(rls)

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
			l.Debug().Str("proxy", proxyConf.Name).Msg("using proxy")

			indexer.Proxy = proxyConf
		} else {
			l.Debug().Str("proxy", proxyConf.Name).Msg("proxy disabled, skipping")
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
	l := s.releaseLogger(r)

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
		l.Debug().Str("proxy", indexer.Proxy.Name).Msg("using proxy")

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

	errFunc := retry.Do(
		retryableRequest(httpClient, req, r),
		retry.Attempts(3),
		retry.MaxJitter(time.Second*1),
		//retry.Delay(time.Second*3),
		retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
			l.Error().Err(err).Uint("attempt", n).Msg("http call encountered error")

			var retriable *RetriableError
			if errors.As(err, &retriable) {
				l.Debug().Uint("attempt", n).Dur("retry_after", retriable.RetryAfter).Msg("http call rate-limited")
				return retriable.RetryAfter
			}
			return time.Second * 3
			// apply a default exponential back off strategy
			//return retry.BackOffDelay(n, err, config)
		}),
	)

	return errFunc
}

func retryableRequest(httpClient *http.Client, req *http.Request, r *domain.Release) func() error {
	return func() error {
		// Get the data
		resp, err := httpClient.Do(req)
		if err != nil {
			var opErr *net.OpError
			if errors.As(err, &opErr) {
				return retry.Unrecoverable(errors.Wrap(err, "issue from proxy"))
			}

			var urlErr *url.Error
			if errors.As(err, &urlErr) {
				return retry.Unrecoverable(errors.Wrap(err, "url parse error"))
			}

			return errors.Wrap(err, "error downloading file")
		}
		defer sharedhttp.DrainAndClose(resp)

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

		case http.StatusTooManyRequests:
			// check Retry-After header if it contains seconds to wait for the next retry
			after := resp.Header.Get("Retry-After")
			if after == "" {
				delay := 3
				return &RetriableError{
					Err:        errors.New("rate-limit reached (%d) while downloading torrent (%s) file (%s) indexer (%s), retrying in %d seconds...", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name, delay),
					RetryAfter: time.Duration(delay) * time.Second,
				}
			}

			if retryAfter, e := strconv.ParseInt(after, 10, 32); e == nil {
				// the server returns 0 to inform that the operation cannot be retried
				if retryAfter <= 0 {
					return retry.Unrecoverable(errors.New("rate-limit reached (%d) while downloading torrent (%s) file (%s) indexer (%s)", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name))
				}
				if retryAfter > 7200 {
					return retry.Unrecoverable(errors.New("rate-limit reached (%d) while downloading torrent (%s) file (%s) indexer (%s) retry-after %d seconds is higher than allowed limit of 2h, aborting", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name, retryAfter))
				}

				rateLimitErr := errors.New("rate-limit reached (%d) while downloading torrent (%s) file (%s) indexer (%s), retrying in %d seconds", resp.StatusCode, r.TorrentName, r.DownloadURL, r.Indexer.Name, retryAfter)

				return &RetriableError{
					Err:        rateLimitErr,
					RetryAfter: time.Duration(retryAfter) * time.Second,
				}
			}

		default:
			return retry.Unrecoverable(errors.New("unexpected status code %d: check indexer keys for %s", resp.StatusCode, r.Indexer.Name))
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
			return retry.Unrecoverable(errors.Wrap(err, "metainfo could not unmarshal info from torrent: %s", r.TorrentName))
		}

		hashInfoBytes := meta.HashInfoBytes().Bytes()
		if len(hashInfoBytes) < 1 {
			return retry.Unrecoverable(errors.New("could not read infohash"))
		}

		// keep the torrent in memory, a tmp file is only written on demand
		// for the path macros via Release.WriteTemporaryFile
		r.TorrentDataRawBytes = bodyBytes
		r.TorrentHash = meta.HashInfoBytes().String()
		// A malformed torrent can carry negative file lengths; keep the
		// announce-derived size rather than storing a wrapped uint64.
		if size := torrentMetaInfo.TotalLength(); size > 0 {
			r.Size = uint64(size)
		}

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

		s.releaseLogger(r).Debug().Str("proxy", proxyConf.Name).Msg("using proxy")

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

	defer sharedhttp.DrainAndClose(res)

	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "could not read response body")
	}

	magnet := strings.TrimSpace(string(body))
	if !strings.HasPrefix(magnet, domain.MagnetURIPrefix) {
		// the url did not lead to a magnet after all, drop it so HasMagnetUri stays
		// false and the release falls back to downloading from DownloadURL
		r.MagnetURI = ""
		return nil
	}

	r.MagnetURI = magnet

	return nil
}
