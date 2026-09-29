package list

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr/lidarr"
	"github.com/autobrr/autobrr/pkg/arr/radarr"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"
	"github.com/autobrr/autobrr/pkg/arr/sportarr"
	"github.com/autobrr/autobrr/pkg/arr/whisparr"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Processor interface {
	Process(ctx context.Context) (*domain.FilterUpdate, error)
}

type processorBase struct {
	log  zerolog.Logger
	list *domain.List
}

func (s *Service) getClientInstance[T any](ctx context.Context, clientID int) (*T, error) {
	instance, err := s.downloaderSvc.GetInstance(ctx, int32(clientID))
	if err != nil {
		return nil, err
	}

	cfg := instance.Config()
	if cfg == nil {
		return nil, errors.Errorf("client %d has no config", clientID)
	}

	if !cfg.Enabled {
		return nil, errors.Errorf("client %s %s not enabled", cfg.Type, cfg.Name)
	}

	return instance.ClientAs[*T]()
}

func (s *Service) getProcessor(ctx context.Context, list *domain.List) (Processor, error) {
	var processor Processor
	switch list.Type {
	case domain.ListTypeLidarr:
		client, err := s.getClientInstance[lidarr.Client](ctx, list.ClientID)
		if err != nil {
			return nil, err
		}
		processor = NewLidarrProcessor(s.log, list, client)

	case domain.ListTypeRadarr:
		client, err := s.getClientInstance[radarr.Client](ctx, list.ClientID)
		if err != nil {
			return nil, err
		}
		processor = NewRadarrProcessor(s.log, list, client)

	case domain.ListTypeReadarr:
		client, err := s.getClientInstance[readarr.Client](ctx, list.ClientID)
		if err != nil {
			return nil, err
		}
		processor = NewReadarrProcessor(s.log, list, client)

	case domain.ListTypeSonarr:
		client, err := s.getClientInstance[sonarr.Client](ctx, list.ClientID)
		if err != nil {
			return nil, err
		}
		processor = NewSonarrProcessor(s.log, list, client)

	case domain.ListTypeSportarr:
		client, err := s.getClientInstance[sportarr.Client](ctx, list.ClientID)
		if err != nil {
			return nil, err
		}
		processor = NewSportarrProcessor(s.log, list, client)

	case domain.ListTypeWhisparr, domain.ListTypeWhisparrV3:
		client, err := s.getClientInstance[whisparr.Client](ctx, list.ClientID)
		if err != nil {
			return nil, err
		}
		processor = NewWhisparrProcessor(s.log, list, client)

	case domain.ListTypeAniList:
		processor = NewAnilistProcessor(s.log, list)
	case domain.ListTypeMDBList:
		processor = NewMDBListProcessor(s.log, list)
	case domain.ListTypeMetacritic:
		processor = NewMetacriticProcessor(s.log, list)
	case domain.ListTypePlaintext:
		processor = NewPlaintextProcessor(s.log, list)
	case domain.ListTypeSteam:
		processor = NewSteamProcessor(s.log, list)
	case domain.ListTypeTrakt:
		processor = NewTraktProcessor(s.log, list)
	default:
		return nil, errors.Errorf("unknown list type: %s", list.Type)
	}

	return processor, nil
}

func (s *Service) Process(ctx context.Context, list *domain.List) error {
	processor, err := s.getProcessor(ctx, list)
	if err != nil {
		return err
	}

	if err := s.process(ctx, list, processor); err != nil {
		s.log.Error().Err(err).Str("type", string(list.Type)).Str("list", list.Name).Msg("error refreshing list")

		// update last run for list and set errs and status
		list.LastRefreshStatus = domain.ListRefreshStatusError
		list.LastRefreshData = err.Error()
		list.LastRefreshTime = time.Now()

		if updateErr := s.repo.UpdateLastRefresh(ctx, list); updateErr != nil {
			s.log.Error().Err(updateErr).Str("type", string(list.Type)).Str("list", list.Name).Msg("error updating last refresh for list")
			return updateErr
		}

		return err
	}

	list.LastRefreshStatus = domain.ListRefreshStatusSuccess
	//listItem.LastRefreshData = err.Error()
	list.LastRefreshTime = time.Now()

	if updateErr := s.repo.UpdateLastRefresh(ctx, list); updateErr != nil {
		s.log.Error().Err(updateErr).Str("type", string(list.Type)).Str("list", list.Name).Msg("error updating last refresh for list")
		return updateErr
	}

	s.log.Debug().Str("list", list.Name).Msg("successfully refreshed list")

	return nil
}

func (s *Service) process(ctx context.Context, list *domain.List, processor Processor) error {
	l := s.log.With().Str("type", string(list.Type)).Str("list", list.Name).Logger()

	filterUpdate, err := processor.Process(ctx)
	if err != nil {
		return errors.Wrap(err, "could not make new request for URL")
	}

	if filterUpdate == nil {
		return nil
	}

	for _, filter := range list.Filters {
		l.Debug().Str("filter", filter.Name).Int("filter_id", filter.ID).Msg("updating filter")

		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, *filterUpdate); err != nil {
			return errors.Wrapf(err, "error updating filter: %d", filter.ID)
		}

		l.Debug().Str("filter", filter.Name).Int("filter_id", filter.ID).Msg("successfully updated filter")
	}

	return nil
}
