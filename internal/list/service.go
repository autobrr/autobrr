package list

import (
	"context"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/scheduler"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Service interface {
}

type service struct {
	log zerolog.Logger

	httpClient        *http.Client
	scheduler         scheduler.Service
	downloadClientSvc download_client.Service
	filterSvc         filter.Service
}

func NewService(log logger.Logger, downloadClientSvc download_client.Service, filterSvc filter.Service, schedulerSvc scheduler.Service) Service {
	return &service{
		log: log.With().Str("module", "list").Logger(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		downloadClientSvc: downloadClientSvc,
		filterSvc:         filterSvc,
		scheduler:         schedulerSvc,
	}
}

func (s *service) List(ctx context.Context) ([]domain.List, error) {
	//data := make([]domain.List, 0)
	data := []domain.List{
		{
			ID:      1,
			Name:    "test",
			Type:    "RADARR",
			Filters: []int{1},
		},
	}

	return data, nil
}

func (s *service) Get(ctx context.Context, id int) (*domain.List, error) {
	return nil, nil
}

func (s *service) Store(ctx context.Context, list domain.List) (*domain.List, error) {
	return nil, nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	return nil
}

func (s *service) Update(ctx context.Context, list domain.List) (*domain.List, error) {
	return nil, nil
}

func (s *service) RefreshAll(ctx context.Context) error {
	lists, err := s.List(ctx)
	if err != nil {
		return err
	}

	if err := s.refreshAll(ctx, lists); err != nil {
		return err
	}

	return nil
}

func (s *service) refreshAll(ctx context.Context, lists []domain.List) error {
	var processingErrors []error

	for _, listItem := range lists {
		if !listItem.Enabled {
			continue
		}

		s.log.Debug().Msgf("run processing for %s - %s", listItem.Type, listItem.Name)

		var err error

		switch listItem.Type {
		case domain.ListTypeRadarr:
			//err = s.radarr(ctx, arrClient, dryRun, s.autobrrClient)

		case domain.ListTypeSonarr:
			err = s.sonarr(ctx, &listItem)

		case domain.ListTypeWhisparr:
			//err = s.sonarr(ctx, arrClient, dryRun, s.autobrrClient)

		case domain.ListTypeReadarr:
			//err = s.readarr(ctx, arrClient, dryRun, s.autobrrClient)

		case domain.ListTypeLidarr:
			//err = s.lidarr(ctx, arrClient, dryRun, s.autobrrClient)

		case domain.ListTypeMDBList:

		case domain.ListTypeTrakt:

		case domain.ListTypeMetacritic:

		case domain.ListTypeSteam:

		default:
			err = errors.Errorf("unsupported list type: %s", listItem.Type)
		}

		if err != nil {
			s.log.Error().Err(err).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msgf("error while processing %s, continuing with other lists", listItem.Type)

			processingErrors = append(processingErrors, errors.Wrapf(err, "%s - %s", listItem.Type, listItem.Name))
		}
	}

	if len(processingErrors) > 0 {
		s.log.Error().Msg("Errors encountered during processing Arrs:")
		for _, errMsg := range processingErrors {
			s.log.Error().Err(errMsg).Msg("error message:")
		}
	}

	return nil
}

func (s *service) TriggerRefresh(ctx context.Context, listID int) error {
	list, err := s.Get(ctx, listID)
	if err != nil {
		return err
	}

	// TODO get single one
	if err := s.refreshAll(ctx, []domain.List{*list}); err != nil {
		return err
	}

	return nil
}

// shouldProcessItem determines if an item should be processed based on its monitored status and configuration
func (s *service) shouldProcessItem(monitored bool, list *domain.List) bool {
	if list.IncludeUnmonitored {
		return true
	}
	return monitored
}
