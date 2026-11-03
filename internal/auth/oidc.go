package auth

import (
	"context"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

const (
	oidcDiscoveryTimeout  = 15 * time.Second
	oidcDiscoveryMinDelay = 5 * time.Second
	oidcDiscoveryMaxDelay = time.Minute
)

// ErrProviderUnavailable is returned until the OIDC provider has been discovered.
var ErrProviderUnavailable = errors.New("OIDC provider is not available")

type OIDCService struct {
	log zerolog.Logger
	cfg *domain.Config

	// discovered is nil until discovery succeeds. It is written once by the
	// discovery goroutine and read by request handlers, so it is swapped in whole.
	discovered atomic.Pointer[discoveredProvider]
}

type discoveredProvider struct {
	issuer       string
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauthConfig  *oauth2.Config
	supportsPKCE bool
}

func NewOIDCService(log zerolog.Logger, cfg *domain.Config) *OIDCService {
	return &OIDCService{
		log: log.With().Str("module", "oidc").Logger(),
		cfg: cfg,
	}
}

func (s *OIDCService) IsEnabled() bool {
	return s.cfg.OIDCEnabled
}

// IsReady reports whether the provider has been discovered and logins can start.
func (s *OIDCService) IsReady() bool {
	return s.discovered.Load() != nil
}

func (s *OIDCService) ValidateConfig() error {
	s.log.Debug().
		Bool("oidc_enabled", s.cfg.OIDCEnabled).
		Str("oidc_issuer", s.cfg.OIDCIssuer).
		Str("oidc_client_id", s.cfg.OIDCClientID).
		Str("oidc_redirect_url", s.cfg.OIDCRedirectURL).
		Str("oidc_scopes", s.cfg.OIDCScopes).
		Bool("oidc_allowed_subject_set", s.cfg.OIDCAllowedSubject != "").
		Bool("oidc_allowed_email_set", s.cfg.OIDCAllowedEmail != "").
		Msg("initializing OIDC handler with oidcConfig")

	var validationErrors []string

	if s.cfg.OIDCIssuer == "" {
		validationErrors = append(validationErrors, "issuer is required")
	}

	if s.cfg.OIDCClientID == "" {
		validationErrors = append(validationErrors, "client ID is required")
	}

	if s.cfg.OIDCClientSecret == "" {
		validationErrors = append(validationErrors, "client secret is required")
	}

	if s.cfg.OIDCRedirectURL == "" {
		validationErrors = append(validationErrors, "redirect URL is required")
	}

	if s.cfg.OIDCAllowedSubject != "" && s.cfg.OIDCAllowedEmail != "" {
		validationErrors = append(validationErrors, "set only one of oidcAllowedSubject and oidcAllowedEmail")
	}

	if len(validationErrors) > 0 {
		return errors.New("OIDC config validation errors: %s", strings.Join(validationErrors, ", "))
	}

	return nil
}

// Discover validates the config and discovers the provider in the background.
// A provider that is down at startup must not stop autobrr from starting, so
// discovery keeps retrying until it succeeds or ctx is cancelled.
func (s *OIDCService) Discover(ctx context.Context) error {
	if !s.IsEnabled() {
		s.log.Debug().Msg("OIDC disabled")
		return nil
	}

	if err := s.ValidateConfig(); err != nil {
		s.log.Error().Err(err).Msg("failed to validate OIDC config")
		return err
	}

	if s.cfg.OIDCAllowedSubject == "" && s.cfg.OIDCAllowedEmail == "" {
		s.log.Warn().Msg("neither oidcAllowedSubject nor oidcAllowedEmail is set: any user your OIDC provider authenticates for this client can log in")
	}

	go s.discover(ctx)

	return nil
}

func (s *OIDCService) discover(ctx context.Context) {
	delay := oidcDiscoveryMinDelay

	for attempt := 1; ; attempt++ {
		discovered, err := s.discoverProvider(ctx)
		if err == nil {
			s.discovered.Store(discovered)
			s.log.Info().Str("issuer", discovered.issuer).Bool("pkce", discovered.supportsPKCE).Msg("OIDC provider initialized successfully")
			return
		}

		s.log.Warn().Err(err).Int("attempt", attempt).Dur("retry_in", delay).Msg("could not discover OIDC provider, logins are unavailable until it responds")

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		delay = min(delay*2, oidcDiscoveryMaxDelay)
	}
}

// discoverProvider tries the issuer with and without a trailing slash, since
// providers disagree on which form their discovery document reports.
func (s *OIDCService) discoverProvider(ctx context.Context) (*discoveredProvider, error) {
	issuer := s.cfg.OIDCIssuer
	candidates := []string{issuer}
	if strings.HasSuffix(issuer, "/") {
		candidates = append(candidates, strings.TrimRight(issuer, "/"))
	} else {
		candidates = append(candidates, issuer+"/")
	}

	var lastErr error
	for _, candidate := range candidates {
		s.log.Trace().Str("issuer", candidate).Msg("attempting OIDC provider initialization")

		attemptCtx, cancel := context.WithTimeout(ctx, oidcDiscoveryTimeout)
		provider, err := oidc.NewProvider(attemptCtx, candidate)
		cancel()
		if err != nil {
			lastErr = err
			continue
		}

		var claims Claims
		if err := provider.Claims(&claims); err != nil {
			s.log.Warn().Err(err).Msg("failed to parse provider claims for endpoints")
		} else {
			s.log.Debug().Str("authorization_endpoint", claims.AuthURL).Str("token_endpoint", claims.TokenURL).Str("jwks_uri", claims.JWKSURL).Str("userinfo_endpoint", claims.UserURL).Msg("discovered OIDC provider endpoints")
		}

		return &discoveredProvider{
			issuer:   candidate,
			provider: provider,
			verifier: provider.Verifier(&oidc.Config{ClientID: s.cfg.OIDCClientID}),
			oauthConfig: &oauth2.Config{
				ClientID:     s.cfg.OIDCClientID,
				ClientSecret: s.cfg.OIDCClientSecret,
				RedirectURL:  s.cfg.OIDCRedirectURL,
				Endpoint:     provider.Endpoint(),
				Scopes:       []string{"openid", "profile", "email"},
			},
			supportsPKCE: slices.Contains(claims.CodeChallenges, "S256"),
		}, nil
	}

	return nil, lastErr
}

func (s *OIDCService) GetProvider() *oidc.Provider {
	if d := s.discovered.Load(); d != nil {
		return d.provider
	}

	return nil
}

func (s *OIDCService) GetIssuer() string {
	if d := s.discovered.Load(); d != nil {
		return d.issuer
	}

	return ""
}

func (s *OIDCService) GetVerifier() *oidc.IDTokenVerifier {
	if d := s.discovered.Load(); d != nil {
		return d.verifier
	}

	return nil
}

func (s *OIDCService) GetEndpoint() oauth2.Endpoint {
	if d := s.discovered.Load(); d != nil {
		return d.provider.Endpoint()
	}

	return oauth2.Endpoint{}
}

func (s *OIDCService) GetAuthorizationURL() string {
	return s.GetEndpoint().AuthURL
}

func (s *OIDCService) GetTokenURL() string {
	return s.GetEndpoint().TokenURL
}

func (s *OIDCService) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*oidc.UserInfo, error) {
	d := s.discovered.Load()
	if d == nil {
		return nil, ErrProviderUnavailable
	}

	userInfo, err := d.provider.UserInfo(ctx, tokenSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get userinfo from provider")
	}

	return userInfo, nil
}

func (s *OIDCService) VerifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	d := s.discovered.Load()
	if d == nil {
		return nil, ErrProviderUnavailable
	}

	return d.verifier.Verify(ctx, idToken)
}

func (s *OIDCService) GetOAuthConfig() *oauth2.Config {
	if d := s.discovered.Load(); d != nil {
		return d.oauthConfig
	}

	return nil
}

func (s *OIDCService) OAuthExchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	d := s.discovered.Load()
	if d == nil {
		return nil, ErrProviderUnavailable
	}

	return d.oauthConfig.Exchange(ctx, code, opts...)
}

// OauthAuthCodeURL returns "" until the provider has been discovered.
func (s *OIDCService) OauthAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	d := s.discovered.Load()
	if d == nil {
		return ""
	}

	return d.oauthConfig.AuthCodeURL(state, opts...)
}

type Claims struct {
	AuthURL        string   `json:"authorization_endpoint"`
	TokenURL       string   `json:"token_endpoint"`
	JWKSURL        string   `json:"jwks_uri"`
	UserURL        string   `json:"userinfo_endpoint"`
	CodeChallenges []string `json:"code_challenge_methods_supported"`
}

func (s *OIDCService) SupportsPKCE() bool {
	if d := s.discovered.Load(); d != nil {
		return d.supportsPKCE
	}

	return false
}
