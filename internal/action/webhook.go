// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

func (s *service) webhook(ctx context.Context, action *domain.Action, release domain.Release) ([]string,error) {
	s.log.Debug().Msgf("action WEBHOOK: '%s' file: %s", action.Name, release.TorrentName)

	if len(action.WebhookData) > 1024 {
		s.log.Trace().Msgf("webhook action '%s' - host: %s data: %s", action.Name, action.WebhookHost, action.WebhookData[:1024])
	} else {
		s.log.Trace().Msgf("webhook action '%s' - host: %s data: %s", action.Name, action.WebhookHost, action.WebhookData)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, action.WebhookHost, bytes.NewBufferString(action.WebhookData))
	if err != nil {
		return nil,errors.Wrap(err, "could not build request for webhook")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	start := time.Now()
	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil,errors.Wrap(err, "could not make request for webhook")
	}

	defer res.Body.Close()

	if len(action.WebhookData) > 256 {
		s.log.Info().Msgf("successfully ran webhook action: '%s' to: %s payload: %s finished in %s with: %d", action.Name, action.WebhookHost, action.WebhookData[:256], time.Since(start), res.StatusCode)
	} else {
		s.log.Info().Msgf("successfully ran webhook action: '%s' to: %s payload: %s finished in %s with: %d", action.Name, action.WebhookHost, action.WebhookData, time.Since(start), res.StatusCode)
	}

	s.log.Info().Msgf("webhook ran and gave %i", res.StatusCode)
	if res.StatusCode == 200 {
		return nil, nil
	} else if res.StatusCode == 201{
		return []string{"Rejected release"}, nil
	}

	return nil, nil
}
