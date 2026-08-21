// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

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
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			deadline := s.sessionManager.Deadline(r.Context())
			if time.Until(deadline) <= 7*24*time.Hour {
				s.log.Trace().Msgf("session is expiring in less than 7 days on %s - extending session", deadline.Format("2006-01-02 15:04:05"))

				if err := s.sessionManager.RenewToken(r.Context()); err != nil {
					s.log.Error().Err(err).Msgf("Auth: Failed to renew session token for username: [%s] ip: %s", s.sessionManager.GetString(r.Context(), "username"), r.RemoteAddr)
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

// LoggerMiddleware logs each request at trace level. Requests whose path exactly
// matches one of skipPaths (e.g. health-check endpoints) are not logged to keep
// polling noise out of the logs.
func LoggerMiddleware(logger *zerolog.Logger, skipPaths []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			log := logger.With().Logger()

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
					// log end request
					log.Trace().
						Str("type", "access").
						Timestamp().
						Fields(map[string]any{
							"remote_ip":  r.RemoteAddr,
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

	userCreds := strings.Split(users, ",")
	for _, cred := range userCreds {
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
