// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
	"encoding/json"

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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read the response body of the webhook")
	}
	s.log.Info().Msgf("webhook action received %s", body)

	type PushResponse struct {
		Approved     bool     `json:"approved"`
		Rejected     bool     `json:"rejected"`
		TempRejected bool     `json:"temporarilyRejected"`
		Rejections   []string `json:"rejections"`
	}

	var pushResponse []PushResponse                                                                                                                                          
	if err = json.Unmarshal(body, &pushResponse); err != nil {                                                                                                                        
		return nil, errors.Wrap(err, "could not unmarshal data")                                                                                                                 
	}    	
	s.log.Info().Msgf("webhook received approved: %s, rejected %s, rejections %s",pushResponse[0].Approved, pushResponse[0].Rejected, pushResponse[0].Rejections)


	if len(action.WebhookData) > 256 {
		s.log.Info().Msgf("successfully ran webhook action: '%s' to: %s payload: %s finished in %s with: %s", action.Name, action.WebhookHost, action.WebhookData[:256], time.Since(start), pushResponse)
	} else {
		s.log.Info().Msgf("successfully ran webhook action: '%s' to: %s payload: %s finished in %s with: %s", action.Name, action.WebhookHost, action.WebhookData, time.Since(start), pushResponse)
	}

	if pushResponse[0].Rejected {
		s.log.Info().Msgf("Rejected release because of %s",pushResponse[0].Rejections)


		return pushResponse[0].Rejections, nil
	}
	return nil, nil
}
