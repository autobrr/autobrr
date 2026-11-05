// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"

	"golang.org/x/net/http/httpguts"
)

type Config struct {
	ConfigPath              string
	Host                    string `toml:"host"`
	Port                    int    `toml:"port"`
	CorsAllowedOrigins      string `toml:"corsAllowedOrigins"`
	LogLevel                string `toml:"logLevel"`
	LogPath                 string `toml:"logPath"`
	LogMaxSize              int    `toml:"logMaxSize"`
	LogMaxBackups           int    `toml:"logMaxBackups"`
	BaseURL                 string `toml:"baseUrl"`
	BaseURLModeLegacy       bool   `toml:"baseUrlModeLegacy"`
	CustomDefinitions       string `toml:"customDefinitions"`
	CheckForUpdates         bool   `toml:"checkForUpdates"`
	DatabaseType            string `toml:"databaseType"`
	DatabaseDSN             string `toml:"databaseDSN"`
	DatabaseMaxBackups      int    `toml:"databaseMaxBackups"`
	DatabaseAutoMigrate     bool   `toml:"databaseAutoMigrate"`
	PostgresHost            string `toml:"postgresHost"`
	PostgresPort            int    `toml:"postgresPort"`
	PostgresDatabase        string `toml:"postgresDatabase"`
	PostgresUser            string `toml:"postgresUser"`
	PostgresPass            string `toml:"postgresPass"`
	PostgresSSLMode         string `toml:"postgresSSLMode"`
	PostgresSocket          string `toml:"postgresSocket"`
	PostgresExtraParams     string `toml:"postgresExtraParams"`
	ProfilingEnabled        bool   `toml:"profilingEnabled"`
	ProfilingHost           string `toml:"profilingHost"`
	ProfilingPort           int    `toml:"profilingPort"`
	OIDCEnabled             bool   `toml:"oidcEnabled"`
	OIDCIssuer              string `toml:"oidcIssuer"`
	OIDCClientID            string `toml:"oidcClientId"`
	OIDCClientSecret        string `toml:"oidcClientSecret"`
	OIDCRedirectURL         string `toml:"oidcRedirectUrl"`
	OIDCScopes              string `toml:"oidcScopes"`
	OIDCDisableBuiltInLogin bool   `toml:"oidcDisableBuiltInLogin"`
	OIDCAutoRedirect        bool   `toml:"oidcAutoRedirect"`
	OIDCAllowedSubject      string `toml:"oidcAllowedSubject"`
	OIDCAllowedEmail        string `toml:"oidcAllowedEmail"`
	MetricsEnabled          bool   `toml:"metricsEnabled"`
	MetricsHost             string `toml:"metricsHost"`
	MetricsPort             int    `toml:"metricsPort"`
	MetricsBasicAuthUsers   string `toml:"metricsBasicAuthUsers"`

	// AuthDisabled disables all authentication when AuthDisabled is set and
	// AuthDisabledAcknowledgement is set to AuthDisabledAcknowledgementValue.
	// Intended only for deployments behind a reverse proxy that handles
	// authentication itself. Use IsAuthDisabled() to check whether auth is
	// actually disabled.
	AuthDisabled                bool   `toml:"authDisabled"`
	AuthDisabledAcknowledgement string `toml:"authDisabledAcknowledgement"`

	// AuthAllowedPeerCIDRs is the allowlist consulted in disabled and forward
	// authentication modes. Entries match the immediate TCP peer autobrr sees (normally the
	// reverse proxy or loopback), never a forwarded header or the end user's browser.
	AuthAllowedPeerCIDRs []string `toml:"authAllowedPeerCIDRs"`

	// AuthAllowedHosts lists extra hostnames a request's Host header may name in
	// disabled and forward authentication modes, on top of IP literals, localhost, the machine
	// hostname and the hosts of CorsAllowedOrigins. It blocks DNS rebinding.
	AuthAllowedHosts []string `toml:"authAllowedHosts"`

	// AuthClientIPHeader optionally names a header the reverse proxy overwrites on
	// every request with the end user's IP (e.g. X-Real-IP). In disabled and forward
	// authentication modes it is resolved into the request context for access logs; it never
	// affects the peer allowlist, which always checks the TCP peer.
	AuthClientIPHeader string `toml:"authClientIPHeader"`

	// TrustedProxies lists the reverse proxies allowed to report the client IP in
	// X-Forwarded-For while authentication is enabled. Empty keeps the legacy
	// behavior of trusting forwarded headers from any peer. Ignored in disabled
	// and forward authentication modes, where the TCP peer is always used.
	TrustedProxies []string `toml:"trustedProxies"`

	// AuthTrustedUserHeader enables forward authentication mode when set. It names
	// the header the authenticating reverse proxy overwrites on every request with
	// the username, and is only read from peers in AuthAllowedPeerCIDRs.
	AuthTrustedUserHeader string `toml:"authTrustedUserHeader"`

	// AuthTrustedUser is the only username accepted in forward authentication
	// mode. Required there, since autobrr has a single admin account and a shared
	// identity provider would otherwise make every one of its users an admin.
	AuthTrustedUser string `toml:"authTrustedUser"`

	// AuthLogoutURL is where log out points in forward authentication mode,
	// normally the proxy's logout endpoint. Empty hides log out.
	AuthLogoutURL string `toml:"authLogoutURL"`
}

// AuthDisabledAcknowledgementValue is the exact value AuthDisabledAcknowledgement
// must be set to in order to disable authentication.
const AuthDisabledAcknowledgementValue = "I_ACKNOWLEDGE_THIS_IS_A_BAD_IDEA"

// IsAuthDisabled reports whether authentication is disabled. AuthDisabled must
// be set and AuthDisabledAcknowledgement must equal AuthDisabledAcknowledgementValue,
// so an operator cannot disable auth without explicitly acknowledging the risk.
func (c *Config) IsAuthDisabled() bool {
	return c.AuthDisabled && c.AuthDisabledAcknowledgement == AuthDisabledAcknowledgementValue
}

// ParseAuthAllowedPeerCIDRs parses AuthAllowedPeerCIDRs into netip.Prefix
// values. Bare IPs are treated as single-host prefixes (/32 for IPv4, /128 for IPv6).
// IPv4-mapped IPv6 forms are normalized to IPv4 so they compare consistently with real
// peers. Entries with non-zero host bits (e.g. 192.168.1.5/24) and universal prefixes
// (0.0.0.0/0, ::/0) are rejected to avoid ambiguous or all-address ranges.
func (c *Config) ParseAuthAllowedPeerCIDRs() ([]netip.Prefix, error) {
	return parsePeerPrefixes(c.AuthAllowedPeerCIDRs, "authAllowedPeerCIDRs")
}

// ParseTrustedProxies parses TrustedProxies with the same rules as
// ParseAuthAllowedPeerCIDRs.
func (c *Config) ParseTrustedProxies() ([]netip.Prefix, error) {
	return parsePeerPrefixes(c.TrustedProxies, "trustedProxies")
}

func parsePeerPrefixes(entries []string, field string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(entries))

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		var prefix netip.Prefix

		if strings.Contains(entry, "/") {
			p, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, errors.Wrap(err, "invalid CIDR in %s: %s", field, entry)
			}

			if p.Addr().Is4In6() {
				if p.Bits() < 96 {
					return nil, errors.New("invalid CIDR in %s: %s is an ambiguous IPv4-mapped IPv6 range", field, entry)
				}
				p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
			}

			if p != p.Masked() {
				return nil, errors.New("invalid CIDR in %s: %s has non-zero host bits", field, entry)
			}

			prefix = p
		} else {
			addr, err := netip.ParseAddr(entry)
			if err != nil {
				return nil, errors.Wrap(err, "invalid IP in %s: %s", field, entry)
			}

			addr = addr.Unmap()

			bits := 32
			if addr.Is6() {
				bits = 128
			}

			prefix = netip.PrefixFrom(addr, bits)
		}

		if prefix.Bits() == 0 {
			return nil, errors.New("invalid CIDR in %s: %s matches all addresses", field, entry)
		}

		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}

// ParseAuthAllowedOrigins parses CorsAllowedOrigins into normalized
// scheme://host[:port] origin values, dropping the scheme's default port. While
// authentication is disabled or forwarded the configured origins are enforced server-side
// against the Origin and Host of browser requests, so every entry must be a real
// origin: no wildcard, no bare hostname, no path, and a host that passes
// NormalizeHost so it can feed the Host allowlist.
func (c *Config) ParseAuthAllowedOrigins() ([]string, error) {
	origins := make([]string, 0)

	for entry := range strings.SplitSeq(c.CorsAllowedOrigins, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// rs/cors accepts https://*.example.com, but the same list is matched
		// literally against Origin and Host here, so a wildcard can never match.
		if strings.Contains(entry, "*") {
			return nil, errors.New("wildcard origin in corsAllowedOrigins is not supported: %s", entry)
		}

		u, err := url.Parse(entry)
		if err != nil {
			return nil, errors.Wrap(err, "invalid origin in corsAllowedOrigins: %s", entry)
		}

		if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("invalid origin in corsAllowedOrigins: %s (expected http(s)://host[:port])", entry)
		}

		if _, err := NormalizeHost(u.Host); err != nil {
			return nil, errors.Wrap(err, "invalid host in corsAllowedOrigins origin: %s", entry)
		}

		// Browsers send IPv6 literals in canonical form, so [0:0:0:0:0:0:0:1]
		// must be stored as [::1] to match.
		host := u.Hostname()
		if addr, err := netip.ParseAddr(host); err == nil && addr.Is6() {
			host = "[" + addr.String() + "]"
		}

		// Browsers omit the scheme's default port from Origin, so an explicit one
		// would never match.
		if port := u.Port(); port != "" && !(u.Scheme == "https" && port == "443") && !(u.Scheme == "http" && port == "80") {
			host += ":" + port
		}

		origins = append(origins, strings.ToLower(u.Scheme+"://"+host))
	}

	return origins, nil
}

// ParseAuthAllowedHosts normalizes AuthAllowedHosts with NormalizeHost,
// skipping blank entries.
func (c *Config) ParseAuthAllowedHosts() ([]string, error) {
	hosts := make([]string, 0, len(c.AuthAllowedHosts))

	for _, entry := range c.AuthAllowedHosts {
		if strings.TrimSpace(entry) == "" {
			continue
		}

		host, err := NormalizeHost(entry)
		if err != nil {
			return nil, errors.Wrap(err, "invalid host in authAllowedHosts: %s", entry)
		}

		hosts = append(hosts, host)
	}

	return hosts, nil
}

// NormalizeHost reduces a Host header value or authAllowedHosts entry to a
// lowercase hostname or unmapped IP address, without port, brackets or trailing dot.
func NormalizeHost(host string) (string, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if h, port, err := net.SplitHostPort(host); err == nil {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return "", errors.New("invalid port: %s", port)
		}
		host = h
	}

	if bare, ok := strings.CutPrefix(host, "["); ok {
		host, ok = strings.CutSuffix(bare, "]")
		if !ok {
			return "", errors.New("unterminated IPv6 literal")
		}
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Unmap().WithZone("").String(), nil
	}

	host = strings.TrimSuffix(host, ".")
	if host == "" || len(host) > 253 {
		return "", errors.New("invalid hostname length")
	}

	for label := range strings.SplitSeq(host, ".") {
		if label == "" || len(label) > 63 {
			return "", errors.New("invalid hostname label length")
		}

		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
				return "", errors.New("invalid hostname character: '%c'", r)
			}
		}
	}

	return host, nil
}

// IsOIDCUserAllowed reports whether the OIDC user with the given subject and
// provider-verified email may log in. Each setting is matched only against its
// own claim, so a subject that looks like the allowed email never matches it.
// Every user is allowed when neither setting is set.
func (c *Config) IsOIDCUserAllowed(subject, verifiedEmail string) bool {
	switch {
	case c.OIDCAllowedSubject != "":
		return subject == c.OIDCAllowedSubject
	case c.OIDCAllowedEmail != "":
		return verifiedEmail != "" && strings.EqualFold(verifiedEmail, c.OIDCAllowedEmail)
	default:
		return true
	}
}

// AuthMode is how autobrr authenticates browser users, derived from the config.
type AuthMode string

const (
	AuthModeBuiltin  AuthMode = "builtin"
	AuthModeOIDC     AuthMode = "oidc"
	AuthModeForward  AuthMode = "forward"
	AuthModeDisabled AuthMode = "disabled"
)

func (m AuthMode) String() string {
	return string(m)
}

// AuthMode reports the active authentication mode. ValidateAuthConfig rejects
// configs that enable more than one mode.
func (c *Config) AuthMode() AuthMode {
	switch {
	case strings.TrimSpace(c.AuthTrustedUserHeader) != "":
		return AuthModeForward
	case c.IsAuthDisabled():
		return AuthModeDisabled
	case c.OIDCEnabled:
		return AuthModeOIDC
	default:
		return AuthModeBuiltin
	}
}

// AuthCapabilities lists what the active authentication mode offers. Handlers and
// the frontend check these instead of the mode itself.
type AuthCapabilities struct {
	PasswordLogin     bool `json:"password_login"`
	OIDCLogin         bool `json:"oidc_login"`
	Onboarding        bool `json:"onboarding"`
	AccountManagement bool `json:"account_management"`
	APIKeyManagement  bool `json:"api_key_management"`

	// ProxyGuards enables the peer, Host, browser and framing checks for modes
	// that rely on a reverse proxy to authenticate.
	ProxyGuards bool `json:"proxy_guards"`
}

// AuthCapabilities returns the capabilities of the active authentication mode.
func (c *Config) AuthCapabilities() AuthCapabilities {
	switch c.AuthMode() {
	case AuthModeForward:
		return AuthCapabilities{APIKeyManagement: true, ProxyGuards: true}
	case AuthModeDisabled:
		return AuthCapabilities{ProxyGuards: true}
	case AuthModeOIDC:
		return AuthCapabilities{
			PasswordLogin:     !c.OIDCDisableBuiltInLogin,
			OIDCLogin:         true,
			AccountManagement: true,
			APIKeyManagement:  true,
		}
	default:
		return AuthCapabilities{
			PasswordLogin:     true,
			Onboarding:        true,
			AccountManagement: true,
			APIKeyManagement:  true,
		}
	}
}

// AuthMethod is how a request's caller was authenticated.
type AuthMethod string

const (
	AuthMethodPassword AuthMethod = "password"
	AuthMethodOIDC     AuthMethod = "oidc"
	AuthMethodForward  AuthMethod = "forward"
	AuthMethodDisabled AuthMethod = "disabled"
	AuthMethodAPIKey   AuthMethod = "apikey"
)

func (m AuthMethod) String() string {
	return string(m)
}

// IsTrustedUserAllowed reports whether username, as sent by the proxy in forward
// authentication mode, is exactly AuthTrustedUser. Identity providers can hold
// accounts that differ only by case, so a case-insensitive match could let another
// account in. No user is allowed when AuthTrustedUser is empty.
func (c *Config) IsTrustedUserAllowed(username string) bool {
	user := strings.TrimSpace(c.AuthTrustedUser)

	return user != "" && user == username
}

// ValidateAuthConfig validates the settings of the active authentication mode
// and rejects settings that belong to conflicting modes.
func (c *Config) ValidateAuthConfig() error {
	if c.IsAuthDisabled() && c.OIDCEnabled {
		return errors.New("authDisabled cannot be used together with oidcEnabled")
	}

	switch c.AuthMode() {
	case AuthModeForward:
		return c.validateForwardAuthConfig()
	case AuthModeDisabled:
		return c.validateProxyGuardConfig()
	default:
		if _, err := c.ParseTrustedProxies(); err != nil {
			return errors.Wrap(err, "invalid trustedProxies")
		}
	}

	return nil
}

// validateForwardAuthConfig adds the forward mode rules on top of the proxy guard
// settings it shares with disabled mode.
func (c *Config) validateForwardAuthConfig() error {
	header := strings.TrimSpace(c.AuthTrustedUserHeader)
	if !httpguts.ValidHeaderFieldName(header) {
		return errors.New("authTrustedUserHeader is not a valid header name: %s", header)
	}

	// API keys are read from X-API-Token before the user header, so a username
	// sent there would be looked up as a key and rejected.
	if strings.EqualFold(header, "X-API-Token") {
		return errors.New("authTrustedUserHeader cannot be X-API-Token, which carries API keys")
	}

	conflicts := []struct {
		setting string
		set     bool
	}{
		{setting: "authDisabled", set: c.AuthDisabled},
		{setting: "oidcEnabled", set: c.OIDCEnabled},
		{setting: "oidcDisableBuiltInLogin", set: c.OIDCDisableBuiltInLogin},
		{setting: "oidcAutoRedirect", set: c.OIDCAutoRedirect},
		{setting: "oidcAllowedSubject", set: c.OIDCAllowedSubject != ""},
		{setting: "oidcAllowedEmail", set: c.OIDCAllowedEmail != ""},
	}
	for _, conflict := range conflicts {
		if conflict.set {
			return errors.New("authTrustedUserHeader cannot be used together with %s", conflict.setting)
		}
	}

	if strings.TrimSpace(c.AuthTrustedUser) == "" {
		return errors.New("authTrustedUser is required in forward authentication mode")
	}

	if err := c.validateProxyGuardConfig(); err != nil {
		return err
	}

	// Any allowlisted peer can send the user header itself, so a range would let
	// every address in it log in as any user, past API key scopes.
	prefixes, err := c.ParseAuthAllowedPeerCIDRs()
	if err != nil {
		return err
	}

	for _, prefix := range prefixes {
		if prefix.Bits() != prefix.Addr().BitLen() {
			return errors.New("authAllowedPeerCIDRs entry %s is a range: forward authentication mode only accepts single proxy addresses (/32 or /128)", prefix)
		}
	}

	if c.AuthLogoutURL != "" {
		u, err := url.Parse(c.AuthLogoutURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return errors.New("authLogoutURL must be an absolute http(s) URL: %s", c.AuthLogoutURL)
		}

		if _, err := NormalizeHost(u.Host); err != nil {
			return errors.Wrap(err, "invalid host in authLogoutURL: %s", c.AuthLogoutURL)
		}
	}

	return nil
}

// validateProxyGuardConfig validates the peer, origin and host allowlists that
// disabled and forward modes rely on in place of a login.
func (c *Config) validateProxyGuardConfig() error {
	if len(c.AuthAllowedPeerCIDRs) == 0 {
		return errors.New("authAllowedPeerCIDRs is required in %s authentication mode", c.AuthMode())
	}

	prefixes, err := c.ParseAuthAllowedPeerCIDRs()
	if err != nil {
		return err
	}

	if len(prefixes) == 0 {
		return errors.New("authAllowedPeerCIDRs contains no valid entries")
	}

	// In this mode the CORS origins double as the browser allowlist: the Origin and
	// Host of browser requests are enforced against them server-side, so they must
	// be explicit, valid origins - a wildcard would let any website script the API
	// from an allowlisted browser.
	origins, err := c.ParseAuthAllowedOrigins()
	if err != nil {
		return err
	}

	if len(origins) == 0 {
		return errors.New("corsAllowedOrigins must be set to explicit origins in %s authentication mode", c.AuthMode())
	}

	if _, err := c.ParseAuthAllowedHosts(); err != nil {
		return err
	}

	if header := strings.TrimSpace(c.AuthClientIPHeader); header != "" && !httpguts.ValidHeaderFieldName(header) {
		return errors.New("authClientIPHeader is not a valid header name: %s", header)
	}

	return nil
}

type ConfigUpdate struct {
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty"`
	LogLevel        *string `json:"log_level,omitempty"`
	LogPath         *string `json:"log_path,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	CheckForUpdates *bool   `json:"check_for_updates,omitempty"`
}
