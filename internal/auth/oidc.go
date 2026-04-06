package auth

import (
	"context"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/avast/retry-go"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

const (
	oidcInitMaxAttempts = 50
)

type OIDCService struct {
	log zerolog.Logger
	cfg *domain.Config

	issuer   string
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier

	oauthConfig *oauth2.Config
}

func NewOIDCService(log logger.Logger, cfg *domain.Config) *OIDCService {
	return &OIDCService{
		log: log.With().Str("module", "oidc").Logger(),
		cfg: cfg,
	}
}

func (s *OIDCService) IsEnabled() bool {
	return s.cfg.OIDCEnabled
}

func (s *OIDCService) ValidateConfig() error {
	s.log.Debug().
		Bool("oidc_enabled", s.cfg.OIDCEnabled).
		Str("oidc_issuer", s.cfg.OIDCIssuer).
		Str("oidc_client_id", s.cfg.OIDCClientID).
		Str("oidc_redirect_url", s.cfg.OIDCRedirectURL).
		Str("oidc_scopes", s.cfg.OIDCScopes).
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

	if len(validationErrors) > 0 {
		return errors.New("OIDC config validation errors: %s", strings.Join(validationErrors, ", "))
	}

	return nil
}

func (s *OIDCService) Discover(ctx context.Context) error {
	if !s.IsEnabled() {
		s.log.Debug().Msg("OIDC disabled")
		return nil
	}

	if err := s.ValidateConfig(); err != nil {
		s.log.Error().Err(err).Msg("failed to validate OIDC config")
		return err
	}

	go func() {
		if err := s.discover(ctx); err != nil {
			s.log.Error().Err(err).Msg("failed to discover OIDC provider")
		}
	}()

	return nil
}

func (s *OIDCService) discover(ctx context.Context) error {
	issuer := s.cfg.OIDCIssuer
	if err := s.DiscoverProvider(ctx, issuer); err != nil {
		return errors.Wrap(err, "failed to discover OIDC provider")
	}

	s.initVerifier()

	var claims *Claims
	if err := s.provider.Claims(&claims); err != nil {
		s.log.Warn().Err(err).Msg("failed to parse provider claims for endpoints")
	} else {
		s.log.Debug().Str("authorization_endpoint", claims.AuthURL).Str("token_endpoint", claims.TokenURL).Str("jwks_uri", claims.JWKSURL).Str("userinfo_endpoint", claims.UserURL).Msg("discovered OIDC provider endpoints")
	}

	scopes := []string{"openid", "profile", "email"}

	s.oauthConfig = &oauth2.Config{
		ClientID:     s.cfg.OIDCClientID,
		ClientSecret: s.cfg.OIDCClientSecret,
		RedirectURL:  s.cfg.OIDCRedirectURL,
		Endpoint:     s.provider.Endpoint(),
		Scopes:       scopes,
	}

	return nil
}

func (s *OIDCService) DiscoverProvider(ctx context.Context, issuer string) error {
	candidates := []string{issuer}
	if strings.HasSuffix(issuer, "/") {
		candidates = append(candidates, strings.TrimRight(issuer, "/"))
	} else {
		candidates = append(candidates, issuer+"/")
	}

	retryFunc := func() error {
		var lastErr error
		for _, candidate := range candidates {
			s.log.Trace().Str("issuer", candidate).Msg("attempting OIDC provider initialization")

			provider, err := oidc.NewProvider(ctx, candidate)
			if err == nil {
				s.log.Info().Str("issuer", candidate).Msg("OIDC provider initialized successfully")
				s.issuer = candidate
				s.provider = provider
				return nil
			}

			lastErr = err

			s.log.Warn().Err(err).Str("issuer", candidate).Msg("failed to initialize OIDC provider candidate, retrying..")

			time.Sleep(500 * time.Millisecond)
		}
		return lastErr
	}

	return retry.Do(
		retryFunc,
		retry.OnRetry(func(n uint, err error) {
			if n > 0 {
				s.log.Debug().Int("attempt", int(n)).Msg("OIDC provider initialization attempt")
			}
		}),
		retry.Attempts(oidcInitMaxAttempts),
		retry.Delay(time.Second*5),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.MaxJitter(time.Second*1),
	)
}

func (s *OIDCService) initVerifier() {
	oidcConfig := &oidc.Config{
		ClientID: s.cfg.OIDCClientID,
	}
	s.verifier = s.provider.Verifier(oidcConfig)
}

func (s *OIDCService) GetProvider() *oidc.Provider {
	return s.provider
}

func (s *OIDCService) GetIssuer() string {
	return s.issuer
}

func (s *OIDCService) GetVerifier() *oidc.IDTokenVerifier {
	return s.verifier
}

func (s *OIDCService) GetEndpoint() oauth2.Endpoint {
	return s.provider.Endpoint()
}

func (s *OIDCService) GetAuthorizationURL() string {
	return s.provider.Endpoint().AuthURL
}

func (s *OIDCService) GetTokenURL() string {
	return s.provider.Endpoint().TokenURL
}

func (s *OIDCService) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*oidc.UserInfo, error) {
	userInfo, err := s.provider.UserInfo(ctx, tokenSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get userinfo from provider")
	}
	return userInfo, err
}

func (s *OIDCService) VerifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	return s.verifier.Verify(ctx, idToken)
}

func (s *OIDCService) GetOAuthConfig() *oauth2.Config {
	return s.oauthConfig
}

func (s *OIDCService) OAuthExchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return s.oauthConfig.Exchange(ctx, code, opts...)
}

func (s *OIDCService) OauthAuthCodeURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state)
}

type Claims struct {
	AuthURL  string `json:"authorization_endpoint"`
	TokenURL string `json:"token_endpoint"`
	JWKSURL  string `json:"jwks_uri"`
	UserURL  string `json:"userinfo_endpoint"`
}
