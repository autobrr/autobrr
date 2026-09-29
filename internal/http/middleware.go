// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) sessionMiddleware(next http.Handler) http.Handler {
	handler := s.sessionManager.LoadAndSave(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") != "https" {
			handler.ServeHTTP(w, r)
			return
		}

		// SCS must write its cookie before we add Secure, including when the handler writes no response.
		sw := &secureSessionResponseWriter{ResponseWriter: w, cookieName: s.sessionManager.Cookie.Name}
		handler.ServeHTTP(sw, r)
		sw.secureCookie()
	})
}

type secureSessionResponseWriter struct {
	http.ResponseWriter
	cookieName string
	written    bool
}

func (s *secureSessionResponseWriter) secureCookie() {
	if s.written {
		return
	}

	for i, cookie := range s.Header()["Set-Cookie"] {
		if strings.HasPrefix(cookie, s.cookieName+"=") {
			s.Header()["Set-Cookie"][i] = cookie + "; Secure"
		}
	}
	s.written = true
}

func (s *secureSessionResponseWriter) WriteHeader(status int) {
	if status >= http.StatusOK {
		s.secureCookie()
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *secureSessionResponseWriter) Write(data []byte) (int, error) {
	s.secureCookie()

	return s.ResponseWriter.Write(data)
}

func (s *secureSessionResponseWriter) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func (s *Server) IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.Config.IsAuthDisabled() {
			next.ServeHTTP(w, r)
			return
		}

		if token := r.Header.Get("X-API-Token"); token != "" {
			// check header
			if !s.apiService.ValidateAPIKey(r.Context(), token) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

		} else if key := r.URL.Query().Get("apikey"); key != "" {
			// check query param like ?apikey=TOKEN
			if !s.apiService.ValidateAPIKey(r.Context(), key) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
		} else {
			// check session
			authenticated := s.sessionManager.GetBool(r.Context(), "authenticated")
			if !authenticated {
				s.log.Debug().Msg("session not authenticated")
				if err := s.sessionManager.Destroy(r.Context()); err != nil {
					s.log.Error().Err(err).Msg("failed to destroy session")
				}
				// 403 on purpose, never 401: a 401 makes browsers holding cached
				// reverse-proxy Basic Auth credentials treat them as invalid and
				// re-prompt, breaking those setups
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			deadline := s.sessionManager.Deadline(r.Context())
			if time.Until(deadline) <= 7*24*time.Hour {
				s.log.Trace().Time("deadline", deadline).Msg("session expiring in less than 7 days, extending")

				if err := s.sessionManager.RenewToken(r.Context()); err != nil {
					s.log.Error().Err(err).Str("username", s.sessionManager.GetString(r.Context(), "username")).Str("remote_addr", r.RemoteAddr).Msg("failed to renew session token")
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// isHealthCheckRequest reports whether r is a GET/HEAD to one of the built-in
// health endpoints (matched by exact resolved path, base URL included). Those are
// allowed from loopback even when the peer is not in authAllowedPeerCIDRs,
// so container/orchestrator health checks on the same host keep working.
func (s *Server) isHealthCheckRequest(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	return slices.Contains(s.healthCheckPaths, r.URL.Path)
}

// RequireAuthDisabledIPAllowlist restricts access to authAllowedPeerCIDRs
// when authentication is disabled. It runs before middleware.RealIP so that
// X-Forwarded-For/X-Real-IP headers cannot be used to spoof past the restriction -
// it always checks the real TCP peer that opened the connection.
func (s *Server) RequireAuthDisabledIPAllowlist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.IsAuthDisabled() {
			next.ServeHTTP(w, r)
			return
		}

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		addr, err := netip.ParseAddr(host)
		if err != nil {
			s.log.Error().Str("remote_addr", r.RemoteAddr).Msg("auth disabled: could not parse remote address")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// Normalize IPv4-mapped IPv6 and strip any IPv6 zone so the peer compares
		// consistently against the configured prefixes.
		addr = addr.Unmap().WithZone("")

		if addr.IsLoopback() && s.isHealthCheckRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		for _, prefix := range s.authAllowedPeerPrefixes {
			if prefix.Contains(addr) {
				next.ServeHTTP(w, r)
				return
			}
		}

		s.log.Warn().Str("peer_ip", addr.String()).Msg("auth disabled: rejected request, peer not in authAllowedPeerCIDRs")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	})
}

// RejectUntrustedBrowserRequests blocks cross-site browser requests to the API
// while authentication is disabled. CORS cannot do that: it only controls whether
// the browser may read the response, and rs/cors invokes the handler regardless of
// origin, so a cross-site request still executes server-side. Requests without
// browser markers (curl, API clients) pass untouched. Fetch metadata is absent
// over plain HTTP, so DNS rebinding is handled by RequireAuthDisabledAllowedHost.
func (s *Server) RejectUntrustedBrowserRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.IsAuthDisabled() {
			next.ServeHTTP(w, r)
			return
		}

		originAllowed := false
		if origin := r.Header.Get("Origin"); origin != "" {
			if _, ok := s.authAllowedOrigins[strings.ToLower(origin)]; !ok {
				s.log.Warn().Str("origin", origin).Str("url", r.URL.Path).Msg("auth disabled: rejected request, origin not in corsAllowedOrigins")
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			originAllowed = true
		}

		// Sec-Fetch-Site covers what Origin cannot: cross-site GET navigations
		// (links, iframes) carry no Origin header. An allowlisted Origin overrides
		// it so operators can deliberately grant cross-origin API access via
		// corsAllowedOrigins.
		secFetchSite := r.Header.Get("Sec-Fetch-Site")
		if secFetchSite == "cross-site" && !originAllowed {
			s.log.Warn().Str("url", r.URL.Path).Msg("auth disabled: rejected cross-site browser request")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			// A cross-site HTML form can submit JSON inside a text/plain body with
			// no preflight; requiring the JSON media type on mutations closes that
			// for browsers that predate Origin and Sec-Fetch.
			if r.ContentLength != 0 {
				if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
					s.log.Warn().Str("content_type", r.Header.Get("Content-Type")).Str("url", r.URL.Path).Msg("auth disabled: rejected mutation, content type must be application/json")
					http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuthDisabledAllowedHost rejects requests whose Host is not allowed
// while authentication is disabled. A DNS-rebound browser is an allowlisted peer
// sending a same-origin request; only the Host header gives it away, and over
// plain HTTP it carries neither Origin nor Sec-Fetch-Site, so every request is
// checked regardless of browser markers.
func (s *Server) RequireAuthDisabledAllowedHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.IsAuthDisabled() {
			next.ServeHTTP(w, r)
			return
		}

		if !s.isAllowedHost(r.Host) {
			s.log.Error().Str("host", r.Host).Str("url", r.URL.Path).Msg("auth disabled: rejected request, host not in authAllowedHosts or corsAllowedOrigins")
			// The peer already passed the IP allowlist, so this is almost always a
			// trusted client on an unlisted hostname; naming the fix beats a bare
			// Forbidden page, and a rebinding attacker learns nothing new from it.
			http.Error(w, "Forbidden: this hostname is not allowed while authentication is disabled, add it to authAllowedHosts in config.toml", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAllowedHost reports whether host is an IP literal, localhost, or a host from
// authAllowedHosts, corsAllowedOrigins or the machine hostname. Rebinding needs a
// DNS name the attacker controls, which rules out the first two.
func (s *Server) isAllowedHost(host string) bool {
	host, err := domain.NormalizeHost(host)
	if err != nil {
		return false
	}

	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}

	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}

	_, ok := s.authAllowedHosts[host]

	return ok
}

// RestrictAuthDisabledFraming limits which sites may frame autobrr while
// authentication is disabled. With auth on, the SameSite=Lax session cookie
// keeps a cross-site frame logged out; with auth off, any framing site would get
// a fully working UI to clickjack.
func (s *Server) RestrictAuthDisabledFraming(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.IsAuthDisabled() {
			next.ServeHTTP(w, r)
			return
		}

		policy := s.authFrameAncestors
		if policy == "" {
			policy = "frame-ancestors 'self'"
		}
		w.Header().Set("Content-Security-Policy", policy)

		next.ServeHTTP(w, r)
	})
}

// LoggerMiddleware logs each request at trace level. Requests whose path exactly
// matches one of skipPaths (e.g. health-check endpoints) are not logged to keep
// polling noise out of the logs.
func LoggerMiddleware(logger *zerolog.Logger, skipPaths []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// the hlog ctx logger carries the request_id field; fall back if
			// this middleware is mounted without hlog.NewHandler
			log := *hlog.FromRequest(r)
			if log.GetLevel() == zerolog.Disabled {
				log = logger.With().Logger()
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				// Recover and record stack traces in case of a panic
				if rec := recover(); rec != nil {
					log.Error().
						Str("type", "error").
						Timestamp().
						Interface("recover_info", rec).
						Bytes("debug_stack", debug.Stack()).
						Msg("log system error")
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}

				if !slices.Contains(skipPaths, r.URL.Path) {
					// prefer the client IP a ClientIPFrom* middleware resolved from a
					// trusted header; RemoteAddr otherwise (RealIP-rewritten when active).
					remoteIP := r.RemoteAddr
					if ip := middleware.GetClientIP(r.Context()); ip != "" {
						remoteIP = ip
					}

					// log end request
					log.Trace().
						Str("type", "access").
						Timestamp().
						Fields(map[string]any{
							"remote_ip":  remoteIP,
							"url":        r.URL.Path,
							"proto":      r.Proto,
							"method":     r.Method,
							"user_agent": r.Header.Get("User-Agent"),
							"status":     ww.Status(),
							"latency_ms": float64(t2.Sub(t1).Nanoseconds()) / 1000000.0,
							"bytes_in":   r.Header.Get("Content-Length"),
							"bytes_out":  ww.BytesWritten(),
						}).
						Msg("incoming_request")
				}
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

// BasicAuth implements a simple middleware handler for adding basic http auth to a route.
func BasicAuth(realm string, users string) func(next http.Handler) http.Handler {
	creds := map[string]string{}

	for cred := range strings.SplitSeq(users, ",") {
		credParts := strings.Split(cred, ":")
		if len(credParts) != 2 {
			//s.log.Warn().Msgf("Invalid metrics basic auth credentials: %s", cred)
			continue
		}

		creds[credParts[0]] = credParts[1]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				basicAuthFailed(w, realm)
				return
			}

			// Validate username and password using htpasswd data
			if hashedPassword, exists := creds[username]; exists {
				// Use bcrypt to validate the password
				if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err == nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			basicAuthFailed(w, realm)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, realm string) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
}
