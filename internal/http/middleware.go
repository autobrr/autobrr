// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func (s Server) IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			session, err := s.cookieStore.Get(r, "user_session")
			if err != nil {
				s.log.Error().Err(err).Msgf("could not get session from cookieStore")
				session.Values["authenticated"] = false

				// MaxAge<0 means delete cookie immediately
				session.Options.MaxAge = -1
				session.Options.Path = s.config.Config.BaseURL

				if err := session.Save(r, w); err != nil {
					s.log.Error().Err(err).Msgf("could not store session: %s", r.RemoteAddr)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			// Check if user is authenticated
			if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
				s.log.Warn().Msg("session not authenticated")

				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			if created, ok := session.Values["created"].(int64); ok {
				// created is a unix timestamp MaxAge is in seconds
				maxAge := time.Duration(session.Options.MaxAge) * time.Second
				expires := time.Unix(created, 0).Add(maxAge)

				if time.Until(expires) <= 7*24*time.Hour { // 7 days
					s.log.Info().Msgf("Cookie is expiring in less than 7 days on %s - extending session", expires.Format("2006-01-02 15:04:05"))

					session.Values["created"] = time.Now().Unix()

					// Call session.Save as needed - since it writes a header (the Set-Cookie
					// header), making sure you call it before writing out a body is important.
					// https://github.com/gorilla/sessions/issues/178#issuecomment-447674812
					if err := session.Save(r, w); err != nil {
						s.log.Error().Err(err).Msgf("could not store session: %s", r.RemoteAddr)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}

			ctx := context.WithValue(r.Context(), "session", session)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

func LoggerMiddleware(logger *zerolog.Logger) func(next http.Handler) http.Handler {
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

				if !strings.Contains("/api/healthz/liveness|/api/healthz/readiness", r.URL.Path) {
					// log end request
					log.Trace().
						Str("type", "access").
						Timestamp().
						Fields(map[string]interface{}{
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
