package action

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/stringutils"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
	"github.com/rs/zerolog"
)

func (s *Service) runWebhook(ctx context.Context, action *domain.Action) error {
	l := zerolog.Ctx(ctx)

	l.Debug().Msg("running Webhook action")

	l.Trace().Str("host", action.WebhookHost).Str("payload", stringutils.TruncateStr(action.WebhookData, 1024)).Msg("running Webhook action")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, action.WebhookHost, bytes.NewBufferString(action.WebhookData))
	if err != nil {
		return errors.Wrap(err, "could not build request for webhook")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	start := time.Now()
	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request for webhook")
	}

	defer sharedhttp.DrainAndClose(res)

	l.Info().Str("host", action.WebhookHost).Str("payload", stringutils.TruncateStr(action.WebhookData, 256)).Dur("duration", time.Since(start)).Msg("webhook action executed")

	return nil
}
