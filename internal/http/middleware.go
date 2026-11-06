// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"cmp"
	"context"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/matrix"
	"github.com/autobrr/autobrr/internal/notification"

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

// Caller is who a request was authenticated as. IsAuthenticated sets it on every
// request it lets through.
type Caller struct {
	Method         domain.AuthMethod `json:"auth_method"`
	Username       string            `json:"username"`
	ProfilePicture string            `json:"profile_picture,omitempty"`

	// APIKey is set only when Method is domain.AuthMethodAPIKey.
	APIKey *domain.APIKey `json:"-"`
}

// IsUser reports whether the caller is a person rather than an API key.
func (c *Caller) IsUser() bool {
	return c.Method != domain.AuthMethodAPIKey
}

// HasAccess reports whether the caller may use resource with access. Users have
// full access; API keys are limited to their scopes.
func (c *Caller) HasAccess(resource domain.APIResource, access domain.APIAccess) bool {
	if c.IsUser() {
		return true
	}

	return c.APIKey != nil && c.APIKey.HasAccess(resource, access)
}

type callerContextKey struct{}

// callerSlot is shared by pointer so middlewares that run before IsAuthenticated,
// such as LoggerMiddleware, can read the caller it resolves further down the chain.
type callerSlot struct {
	caller *Caller
}

// withCallerSlot gives the request an empty caller slot unless it already has one.
func withCallerSlot(r *http.Request) *http.Request {
	if _, ok := r.Context().Value(callerContextKey{}).(*callerSlot); ok {
		return r
	}

	return r.WithContext(context.WithValue(r.Context(), callerContextKey{}, &callerSlot{}))
}

func withCaller(r *http.Request, caller *Caller) *http.Request {
	r = withCallerSlot(r)
	r.Context().Value(callerContextKey{}).(*callerSlot).caller = caller

	return r
}

func callerFromContext(ctx context.Context) (*Caller, bool) {
	slot, ok := ctx.Value(callerContextKey{}).(*callerSlot)
	if !ok || slot.caller == nil {
		return nil, false
	}

	return slot.caller, true
}

// IsAuthenticated resolves the caller of the request and rejects requests it cannot
// resolve: an API key if one is sent, otherwise the login method of the auth mode,
// a session or the forward auth header.
func (s *Server) IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, ok := s.resolveCaller(w, r)
		if !ok {
			return
		}

		next.ServeHTTP(w, withCaller(r, caller))
	})
}

// resolveCaller writes the error response itself when it returns false.
func (s *Server) resolveCaller(w http.ResponseWriter, r *http.Request) (*Caller, bool) {
	if s.config.Config.AuthMode() == domain.AuthModeDisabled {
		return &Caller{Method: domain.AuthMethodDisabled, Username: "admin"}, true
	}

	if token := cmp.Or(r.Header.Get("X-API-Token"), r.URL.Query().Get("apikey")); token != "" {
		apiKey, ok := s.apiService.ValidateAPIKey(r.Context(), token)
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return nil, false
		}

		return &Caller{Method: domain.AuthMethodAPIKey, APIKey: apiKey}, true
	}

	if s.config.Config.AuthMode() == domain.AuthModeForward {
		return s.forwardCaller(w, r)
	}

	return s.sessionCaller(w, r)
}

// forwardCaller resolves the user the reverse proxy vouches for in the trusted user
// header. RequireAuthDisabledIPAllowlist has already limited the request to the
// proxy's own address, which is what makes the header trustworthy. A missing header
// means the proxy stopped authenticating, so it fails closed.
func (s *Server) forwardCaller(w http.ResponseWriter, r *http.Request) (*Caller, bool) {
	header := strings.TrimSpace(s.config.Config.AuthTrustedUserHeader)

	// The last value is the one the proxy closest to us set; earlier ones may have
	// come from the client if the proxy appends instead of overwriting.
	var username string
	if values := r.Header.Values(header); len(values) > 0 {
		username = strings.TrimSpace(values[len(values)-1])
	}

	if username == "" {
		s.log.Warn().Str("header", header).Str("url", r.URL.Path).Str("remote_addr", r.RemoteAddr).Msg("forward auth: rejected request without trusted user header")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil, false
	}

	if !s.config.Config.IsTrustedUserAllowed(username) {
		s.log.Warn().Str("username", username).Str("url", r.URL.Path).Msg("forward auth: rejected user other than authTrustedUser")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil, false
	}

	return &Caller{Method: domain.AuthMethodForward, Username: username}, true
}

// sessionCaller resolves a password or OIDC login session. Failures answer 403 on
// purpose, never 401: a 401 makes browsers holding cached reverse-proxy Basic Auth
// credentials treat them as invalid and re-prompt, breaking those setups.
func (s *Server) sessionCaller(w http.ResponseWriter, r *http.Request) (*Caller, bool) {
	ctx := r.Context()

	if !s.sessionManager.GetBool(ctx, "authenticated") {
		s.log.Debug().Msg("session not authenticated")
		if err := s.sessionManager.Destroy(ctx); err != nil {
			s.log.Error().Err(err).Msg("failed to destroy session")
		}
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil, false
	}

	caller := &Caller{
		Method:         domain.AuthMethod(s.sessionManager.GetString(ctx, "auth_method")),
		Username:       s.sessionManager.GetString(ctx, "username"),
		ProfilePicture: s.sessionManager.GetString(ctx, "profile_picture"),
	}

	// sessions outlive restarts, so one created before the allowed OIDC user
	// was set or changed must not keep a now-disallowed user logged in
	if caller.Method == domain.AuthMethodOIDC {
		subject := s.sessionManager.GetString(ctx, "oidc_subject")
		if !s.config.Config.IsOIDCUserAllowed(subject, s.sessionManager.GetString(ctx, "oidc_verified_email")) {
			s.log.Warn().Str("sub", subject).Str("remote_addr", r.RemoteAddr).Msg("OIDC session user is no longer allowed, destroying session")
			if err := s.sessionManager.Destroy(ctx); err != nil {
				s.log.Error().Err(err).Msg("failed to destroy session")
			}
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return nil, false
		}
	}

	deadline := s.sessionManager.Deadline(ctx)
	if time.Until(deadline) <= 7*24*time.Hour {
		s.log.Trace().Time("deadline", deadline).Msg("session expiring in less than 7 days, extending")

		if err := s.sessionManager.RenewToken(ctx); err != nil {
			s.log.Error().Err(err).Str("username", caller.Username).Str("remote_addr", r.RemoteAddr).Msg("failed to renew session token")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return nil, false
		}
	}

	return caller, true
}

// requireScope limits the route to callers with access to resource. Requests without
// a caller are rejected, so a route mounted outside IsAuthenticated stays closed.
func requireScope(resource domain.APIResource, access domain.APIAccess) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			caller, ok := callerFromContext(r.Context())
			if !ok || !caller.HasAccess(resource, access) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// requireUser keeps API keys away from routes that could widen a key's own access, such as managing keys,
// users or the config file.
func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, ok := callerFromContext(r.Context())
		if !ok || !caller.IsUser() {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// eventStreamResources maps each server-sent event stream to the resource an API key needs read access to.
var eventStreamResources = map[string]domain.APIResource{
	logger.StreamLogs:           domain.APIResourceLogs,
	irc.EventStreamKey:          domain.APIResourceIRC,
	matrix.EventStreamKey:       domain.APIResourceMatrix,
	notification.InboxStreamKey: domain.APIResourceNotifications,
}

// requireStreamAccess scopes the stream picked by the stream query param. Every stream is served by the same
// SSE server, so the route alone does not say what is being read.
func requireStreamAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resource, ok := eventStreamResources[r.URL.Query().Get("stream")]
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		caller, ok := callerFromContext(r.Context())
		if !ok || !caller.HasAccess(resource, domain.APIAccessRead) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
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

type peerAddrKey struct{}

// StorePeerAddr saves the TCP peer that opened the connection in the request
// context. It must run before RealIP, which rewrites r.RemoteAddr from headers
// any peer may send, so that limiters can still key on who really connected.
func StorePeerAddr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if addr, ok := parsePeerAddr(r.RemoteAddr); ok {
			r = r.WithContext(context.WithValue(r.Context(), peerAddrKey{}, addr))
		}

		next.ServeHTTP(w, r)
	})
}

func peerAddrFromContext(ctx context.Context) (netip.Addr, bool) {
	addr, ok := ctx.Value(peerAddrKey{}).(netip.Addr)

	return addr, ok
}

// parsePeerAddr parses a host:port or bare host remote address, with
// IPv4-mapped IPv6 unmapped and any IPv6 zone stripped so the address compares
// consistently against configured prefixes.
func parsePeerAddr(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}

	return addr.Unmap().WithZone(""), true
}

// RequireAuthDisabledIPAllowlist restricts access to authAllowedPeerCIDRs
// in disabled and forward authentication modes. It runs before middleware.RealIP so that
// X-Forwarded-For/X-Real-IP headers cannot be used to spoof past the restriction -
// it always checks the real TCP peer that opened the connection.
func (s *Server) RequireAuthDisabledIPAllowlist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.AuthCapabilities().ProxyGuards {
			next.ServeHTTP(w, r)
			return
		}

		addr, ok := parsePeerAddr(r.RemoteAddr)
		if !ok {
			s.log.Error().Str("remote_addr", r.RemoteAddr).Msg("proxy guard: could not parse remote address")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

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

		s.log.Warn().Str("peer_ip", addr.String()).Msg("proxy guard: rejected request, peer not in authAllowedPeerCIDRs")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	})
}

// RejectUntrustedBrowserRequests blocks cross-site and same-site browser requests
// to the API in disabled and forward authentication modes, where the reverse
// proxy's session authenticates whatever the browser sends. CORS cannot do that: it only
// controls whether the browser may read the response, and rs/cors invokes the
// handler regardless of origin, so a cross-site request still executes
// server-side. Same-site is rejected too because a sibling subdomain under the
// same registrable domain shares the reverse proxy's SameSite=Lax cookie, so an
// <img> there reaches state-changing GET routes; the UI's own requests are
// same-origin. Requests without browser markers (curl, API clients) pass
// untouched. Fetch metadata is absent over plain HTTP, so DNS rebinding is
// handled by RequireAuthDisabledAllowedHost.
func (s *Server) RejectUntrustedBrowserRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.AuthCapabilities().ProxyGuards {
			next.ServeHTTP(w, r)
			return
		}

		originAllowed := false
		if origin := r.Header.Get("Origin"); origin != "" {
			_, originAllowed = s.authAllowedOrigins[strings.ToLower(origin)]
			if !originAllowed && !isSameOrigin(origin, r) {
				s.log.Warn().Str("origin", origin).Str("url", r.URL.Path).Msg("proxy guard: rejected request, origin not in corsAllowedOrigins")
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		// Sec-Fetch-Site covers what Origin cannot: GET navigations and subresource
		// loads (links, iframes, <img>) carry no Origin header. same-site is as
		// hostile as cross-site here: an attacker on a sibling subdomain rides the
		// proxy's SameSite=Lax cookie. An allowlisted Origin overrides both so
		// operators can deliberately grant cross-origin API access via
		// corsAllowedOrigins.
		secFetchSite := r.Header.Get("Sec-Fetch-Site")
		if (secFetchSite == "cross-site" || secFetchSite == "same-site") && !originAllowed {
			s.log.Warn().Str("sec_fetch_site", secFetchSite).Str("url", r.URL.Path).Msg("proxy guard: rejected browser request from another site")
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
					s.log.Warn().Str("content_type", r.Header.Get("Content-Type")).Str("url", r.URL.Path).Msg("proxy guard: rejected mutation, content type must be application/json")
					http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// isSameOrigin reports whether origin names the host the request was sent to.
// Browsers send Origin on same-origin writes too, so without this the UI reached
// on an IP or localhost, which needs no allowlist entry to load, could not save.
// The scheme is checked only when a proxy reports it in X-Forwarded-Proto: a
// TLS-terminating proxy forwards plain HTTP, so without the header the backend
// cannot tell which scheme the browser used. The Host itself has already passed
// RequireAuthDisabledAllowedHost. A matching Origin does not override a
// cross-site Sec-Fetch-Site the way an allowlisted one does.
func isSameOrigin(origin string, r *http.Request) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		proto, _, _ = strings.Cut(proto, ",")
		if !strings.EqualFold(u.Scheme, strings.TrimSpace(proto)) {
			return false
		}
	}

	return strings.EqualFold(u.Host, r.Host)
}

// RequireAuthDisabledAllowedHost rejects requests whose Host is not allowed
// in disabled and forward authentication modes. A DNS-rebound browser is an allowlisted peer
// sending a same-origin request; only the Host header gives it away, and over
// plain HTTP it carries neither Origin nor Sec-Fetch-Site, so every request is
// checked regardless of browser markers.
func (s *Server) RequireAuthDisabledAllowedHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.AuthCapabilities().ProxyGuards {
			next.ServeHTTP(w, r)
			return
		}

		if !s.isAllowedHost(r.Host) {
			s.log.Error().Str("host", r.Host).Str("url", r.URL.Path).Msg("proxy guard: rejected request, host not in authAllowedHosts or corsAllowedOrigins")
			// The peer already passed the IP allowlist, so this is almost always a
			// trusted client on an unlisted hostname; naming the fix beats a bare
			// Forbidden page, and a rebinding attacker learns nothing new from it.
			http.Error(w, "Forbidden: this hostname is not allowed, add it to authAllowedHosts in config.toml", http.StatusForbidden)
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

// RestrictAuthDisabledFraming limits which sites may frame autobrr in disabled and
// forward authentication modes. With a login, the SameSite=Lax session cookie
// keeps a cross-site frame logged out; when the proxy authenticates, any framing
// site would get a fully working UI to clickjack.
func (s *Server) RestrictAuthDisabledFraming(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.Config.AuthCapabilities().ProxyGuards {
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

// RealIPFromTrustedProxies rewrites r.RemoteAddr to the client address from
// X-Forwarded-For like middleware.RealIP, but only when the TCP peer is a
// trusted proxy. The chain is read right to left past trusted hops, so a
// client cannot pick its own address by sending the header itself.
func RealIPFromTrustedProxies(trusted []netip.Prefix) func(http.Handler) http.Handler {
	prefixes := make([]string, 0, len(trusted))
	for _, p := range trusted {
		prefixes = append(prefixes, p.String())
	}

	fromXFF := middleware.ClientIPFromXFF(prefixes...)

	return func(next http.Handler) http.Handler {
		rewrite := fromXFF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ip := middleware.GetClientIP(r.Context()); ip != "" {
				r.RemoteAddr = ip
			}

			next.ServeHTTP(w, r)
		}))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer, ok := parsePeerAddr(r.RemoteAddr)
			if !ok || !isTrustedProxy(trusted, peer) {
				next.ServeHTTP(w, r)
				return
			}

			rewrite.ServeHTTP(w, r)
		})
	}
}

func isTrustedProxy(trusted []netip.Prefix, peer netip.Addr) bool {
	return slices.ContainsFunc(trusted, func(p netip.Prefix) bool { return p.Contains(peer) })
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

			r = withCallerSlot(r)
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

					var username string
					if caller, ok := callerFromContext(r.Context()); ok {
						username = caller.Username
					}

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
							"username":   username,
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
