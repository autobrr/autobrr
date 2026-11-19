// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"context"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/matrix"
)

// deviceLogin is an OAuth 2.0 device authorization the user approves in a browser.
type deviceLogin struct {
	cancel context.CancelFunc

	m     sync.Mutex
	state domain.MatrixLogin
}

func (l *deviceLogin) snapshot() domain.MatrixLogin {
	l.m.Lock()
	defer l.m.Unlock()

	return l.state
}

func (l *deviceLogin) finish(state domain.MatrixLoginState, userID string, err error) {
	l.m.Lock()
	defer l.m.Unlock()

	l.state.State = state
	l.state.UserID = userID

	if err != nil {
		l.state.Error = err.Error()
	}
}

// cancelLogin stops a pending login. The caller holds s.m.
func (s *Service) cancelLogin(id int64) {
	login, ok := s.logins[id]
	if !ok {
		return
	}

	login.cancel()
	delete(s.logins, id)
}

// StartLogin begins a device login for a network and returns the code the user
// enters at the verification URL. The login completes in the background; its
// progress is reported in the network health.
func (s *Service) StartLogin(ctx context.Context, id int64) (*domain.MatrixLogin, error) {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		return nil, err
	}

	l := s.log.With().Str("network", network.Name).Int64("network_id", network.ID).Logger()

	client := matrix.NewClient(network.Homeserver, s.httpClient)
	oauth := matrix.NewOAuthClient(s.httpClient)

	metadata, err := client.AuthMetadata(ctx)
	if err != nil {
		l.Error().Err(err).Msg("could not discover matrix authorization server")
		return nil, errors.Wrap(err, "homeserver does not support OAuth login, use an access token instead")
	}

	clientID := network.OAuthClientID
	deviceID := matrix.NewDeviceID()

	var authorization *matrix.DeviceAuthorization

	if clientID != "" {
		authorization, err = oauth.RequestDeviceAuthorization(ctx, metadata, clientID, deviceID)
		if err != nil {
			// the authorization server may have dropped the client registered at the last login
			l.Debug().Err(err).Msg("could not reuse matrix oauth client, registering a new one")
			clientID = ""
		}
	}

	if clientID == "" {
		clientID, err = oauth.RegisterClient(ctx, metadata, matrix.DefaultClientMetadata())
		if err != nil {
			l.Error().Err(err).Msg("could not register matrix oauth client")
			return nil, errors.Wrap(err, "could not register oauth client")
		}

		authorization, err = oauth.RequestDeviceAuthorization(ctx, metadata, clientID, deviceID)
		if err != nil {
			l.Error().Err(err).Msg("could not start matrix device login")
			return nil, errors.Wrap(err, "could not start device login")
		}
	}

	expiresAt := time.Now().Add(time.Duration(authorization.ExpiresIn) * time.Second)

	loginCtx, cancel := context.WithDeadline(context.Background(), expiresAt)

	login := &deviceLogin{
		cancel: cancel,
		state: domain.MatrixLogin{
			State:                   domain.MatrixLoginStatePending,
			UserCode:                authorization.UserCode,
			VerificationURI:         authorization.VerificationURI,
			VerificationURIComplete: authorization.VerificationURIComplete,
			ExpiresAt:               expiresAt,
		},
	}

	s.m.Lock()
	defer s.m.Unlock()

	if s.stopped {
		cancel()
		return nil, errors.New("matrix service is stopped")
	}

	s.cancelLogin(id)
	s.logins[id] = login

	s.wg.Go(func() {
		defer cancel()

		userID, err := s.pollLogin(loginCtx, network.ID, network.Homeserver, metadata, clientID, authorization)
		if err != nil {
			l.Error().Err(err).Msg("matrix device login failed")
			login.finish(domain.MatrixLoginStateFailed, "", err)
			return
		}

		l.Info().Str("user_id", userID).Msg("matrix device login succeeded")
		login.finish(domain.MatrixLoginStateSucceeded, userID, nil)
	})

	state := login.snapshot()

	return &state, nil
}

// pollLogin polls the token endpoint until the user approves, then stores the
// tokens and restarts the network.
func (s *Service) pollLogin(ctx context.Context, networkID int64, homeserver string, metadata *matrix.AuthMetadata, clientID string, authorization *matrix.DeviceAuthorization) (string, error) {
	oauth := matrix.NewOAuthClient(s.httpClient)
	interval := time.Duration(authorization.Interval) * time.Second

	var token *matrix.Token

	for token == nil {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return "", errors.New("login code expired")
			}

			return "", errors.New("login cancelled")
		case <-time.After(interval):
		}

		t, err := oauth.PollDeviceToken(ctx, metadata, clientID, authorization.DeviceCode)
		if err != nil {
			oerr, ok := errors.AsType[*matrix.OAuthError](err)
			if !ok {
				// transient network failures do not end the login while the code is valid
				s.log.Debug().Err(err).Int64("network_id", networkID).Msg("could not poll matrix device token")
				continue
			}

			switch oerr.Code {
			case matrix.OAuthErrAuthorizationPending:
				continue
			case matrix.OAuthErrSlowDown:
				interval += 5 * time.Second
				continue
			case matrix.OAuthErrAccessDenied:
				return "", errors.New("login was denied")
			case matrix.OAuthErrExpiredToken:
				return "", errors.New("login code expired")
			default:
				return "", err
			}
		}

		token = t
	}

	client := matrix.NewClient(homeserver, s.httpClient)
	client.SetAccessToken(token.AccessToken)

	whoami, err := client.Whoami(ctx)
	if err != nil {
		return "", errors.Wrap(err, "could not verify new access token")
	}

	tokens := domain.MatrixTokens{
		AuthMethod:         domain.MatrixAuthMethodOAuth,
		UserID:             whoami.UserID,
		DeviceID:           whoami.DeviceID,
		AccessToken:        token.AccessToken,
		RefreshToken:       token.RefreshToken,
		OAuthClientID:      clientID,
		OAuthTokenEndpoint: metadata.TokenEndpoint,
	}

	if expiresAt := token.ExpiresAt(time.Now()); !expiresAt.IsZero() {
		tokens.TokenExpiresAt = &expiresAt
	}

	s.m.Lock()
	defer s.m.Unlock()

	// a login replaced or cancelled while this one waited for the lock must not store its tokens
	if ctx.Err() != nil {
		return "", errors.New("login cancelled")
	}

	if err := s.replaceTokens(ctx, networkID, homeserver, tokens); err != nil {
		return "", err
	}

	return whoami.UserID, nil
}
