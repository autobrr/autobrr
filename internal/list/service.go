// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	stdErr "errors"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type listRepo interface {
	List(ctx context.Context) ([]*domain.List, error)
	FindByID(ctx context.Context, listID int64) (*domain.List, error)
	Store(ctx context.Context, listID *domain.List) error
	Update(ctx context.Context, listID *domain.List) error
	UpdateLastRefresh(ctx context.Context, list *domain.List) error
	ToggleEnabled(ctx context.Context, listID int64, enabled bool) error
	Delete(ctx context.Context, listID int64) error
	GetListFilters(ctx context.Context, listID int64) ([]domain.ListFilter, error)
	GetAllListFilters(ctx context.Context) (map[int64][]domain.ListFilter, error)
}

type clientService interface {
	GetClient(ctx context.Context, clientId int32) (*domain.DownloadClient, error)
}

type filterService interface {
	UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error
}

type schedulerService interface {
	AddJob(job cron.Job, spec string, identifier string) (int, error)
}

type Service struct {
	log  zerolog.Logger
	repo listRepo

	httpClient        *http.Client
	scheduler         schedulerService
	downloadClientSvc clientService
	filterSvc         filterService
}

func NewService(log zerolog.Logger, repo listRepo, downloadClientSvc clientService, filterSvc filterService, schedulerSvc schedulerService) *Service {
	return &Service{
		log:  log.With().Str("module", "list").Logger(),
		repo: repo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		downloadClientSvc: downloadClientSvc,
		filterSvc:         filterSvc,
		scheduler:         schedulerSvc,
	}
}

func (s *Service) List(ctx context.Context) ([]*domain.List, error) {
	data, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	filters, err := s.repo.GetAllListFilters(ctx)
	if err != nil {
		return nil, err
	}

	for _, list := range data {
		if listFilters, ok := filters[list.ID]; ok {
			list.Filters = listFilters
		}
	}

	return data, nil
}

func (s *Service) FindByID(ctx context.Context, id int64) (*domain.List, error) {
	list, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// attach filters
	filters, err := s.repo.GetListFilters(ctx, list.ID)
	if err != nil {
		return nil, err
	}

	list.Filters = filters

	return list, nil
}

func (s *Service) Store(ctx context.Context, list *domain.List) error {
	if err := list.Validate(); err != nil {
		s.log.Error().Err(err).Str("list", list.Name).Msg("could not validate list")
		return err
	}

	if err := s.repo.Store(ctx, list); err != nil {
		s.log.Error().Err(err).Str("list", list.Name).Msg("could not store list")
		return err
	}

	s.log.Debug().Str("list", list.Name).Msg("successfully created list")

	if list.Enabled {
		if err := s.refreshList(ctx, list); err != nil {
			s.log.Error().Err(err).Str("list", list.Name).Msg("could not refresh list")
			return err
		}
	}

	return nil
}

func (s *Service) Update(ctx context.Context, list *domain.List) error {
	if err := list.Validate(); err != nil {
		s.log.Error().Err(err).Str("list", list.Name).Msg("could not validate list")
		return err
	}

	existingList, err := s.FindByID(ctx, list.ID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			s.log.Error().Err(err).Int64("list_id", list.ID).Msg("could not find list by id")
		} else {
			s.log.Error().Err(err).Int64("list_id", list.ID).Msg("could not get list by id")
		}

		return err
	}

	if domain.IsRedactedString(list.APIKey) {
		list.APIKey = existingList.APIKey
	}

	if err := s.repo.Update(ctx, list); err != nil {
		s.log.Error().Err(err).Str("list", list.Name).Msg("could not update list")
		return err
	}

	s.log.Debug().Str("list", list.Name).Msg("successfully updated list")

	if list.Enabled {
		if err := s.refreshList(ctx, list); err != nil {
			s.log.Error().Err(err).Str("list", list.Name).Msg("could not refresh list")
			return err
		}
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Int64("list_id", id).Msg("could not delete list by id")
		return err
	}

	s.log.Debug().Int64("list_id", id).Msg("successfully deleted list")

	return nil
}

func (s *Service) RefreshAll(ctx context.Context) error {
	lists, err := s.List(ctx)
	if err != nil {
		return err
	}

	s.log.Debug().Int("count", len(lists)).Msg("found lists to refresh")

	if err := s.refreshAll(ctx, lists); err != nil {
		return err
	}

	s.log.Debug().Msg("successfully refreshed all lists")

	return nil
}

func (s *Service) refreshAll(ctx context.Context, lists []*domain.List) error {
	var processingErrors []error

	for _, listItem := range lists {
		if !listItem.Enabled {
			s.log.Debug().Str("list", listItem.Name).Msg("list is disabled, skipping")
			continue
		}

		if err := s.refreshList(ctx, listItem); err != nil {
			if errors.Is(err, domain.ErrRecordNotFound) {
				s.log.Error().Str("type", string(listItem.Type)).Str("list", listItem.Name).Int("client_id", listItem.ClientID).Msg("client not found for list, skipping")
				continue
			}

			s.log.Error().Err(err).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msg("error while refreshing, continuing with other lists")

			processingErrors = append(processingErrors, errors.Wrapf(err, "error while refreshing %s", listItem.Name))
		}
	}

	if len(processingErrors) > 0 {
		err := stdErr.Join(processingErrors...)

		s.log.Error().Err(err).Msg("Errors encountered during processing Arrs:")

		return err
	}

	return nil
}

func (s *Service) refreshList(ctx context.Context, listItem *domain.List) error {
	s.log.Debug().Str("type", string(listItem.Type)).Str("list", listItem.Name).Msg("refresh list")

	var err error

	switch listItem.Type {
	case domain.ListTypeRadarr:
		err = s.radarr(ctx, listItem)

	case domain.ListTypeSonarr:
		err = s.sonarr(ctx, listItem)

	case domain.ListTypeWhisparr, domain.ListTypeWhisparrV3:
		err = s.whisparr(ctx, listItem)

	case domain.ListTypeReadarr:
		err = s.readarr(ctx, listItem)

	case domain.ListTypeLidarr:
		err = s.lidarr(ctx, listItem)

	case domain.ListTypeSportarr:
		err = s.sportarr(ctx, listItem)

	case domain.ListTypeMDBList:
		err = s.mdblist(ctx, listItem)

	case domain.ListTypeMetacritic:
		err = s.metacritic(ctx, listItem)

	case domain.ListTypeSteam:
		err = s.steam(ctx, listItem)

	case domain.ListTypeTrakt:
		err = s.trakt(ctx, listItem)

	case domain.ListTypePlaintext:
		err = s.plaintext(ctx, listItem)

	case domain.ListTypeAniList:
		err = s.anilist(ctx, listItem)

	default:
		err = errors.Errorf("unsupported list type: %s", listItem.Type)
	}

	if err != nil {
		s.log.Error().Err(err).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msg("error refreshing list")

		// update last run for list and set errs and status
		listItem.LastRefreshStatus = domain.ListRefreshStatusError
		listItem.LastRefreshData = err.Error()
		listItem.LastRefreshTime = time.Now()

		if updateErr := s.repo.UpdateLastRefresh(ctx, listItem); updateErr != nil {
			s.log.Error().Err(updateErr).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msg("error updating last refresh for list")
			return updateErr
		}

		return err
	}

	listItem.LastRefreshStatus = domain.ListRefreshStatusSuccess
	//listItem.LastRefreshData = err.Error()
	listItem.LastRefreshTime = time.Now()

	if updateErr := s.repo.UpdateLastRefresh(ctx, listItem); updateErr != nil {
		s.log.Error().Err(updateErr).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msg("error updating last refresh for list")
		return updateErr
	}

	s.log.Debug().Str("list", listItem.Name).Msg("successfully refreshed list")

	return nil
}

func (s *Service) RefreshList(ctx context.Context, listID int64) error {
	list, err := s.FindByID(ctx, listID)
	if err != nil {
		return err
	}

	if err := s.refreshList(ctx, list); err != nil {
		return err
	}

	return nil
}

func (s *Service) RefreshArrLists(ctx context.Context) error {
	lists, err := s.List(ctx)
	if err != nil {
		return err
	}

	var selectedLists []*domain.List
	for _, list := range lists {
		if list.ListTypeArr() && list.Enabled {
			selectedLists = append(selectedLists, list)
		}
	}

	if err := s.refreshAll(ctx, selectedLists); err != nil {
		return err
	}

	return nil
}

func (s *Service) RefreshOtherLists(ctx context.Context) error {
	lists, err := s.List(ctx)
	if err != nil {
		return err
	}

	var selectedLists []*domain.List
	for _, list := range lists {
		if list.ListTypeList() && list.Enabled {
			selectedLists = append(selectedLists, list)
		}
	}

	if err := s.refreshAll(ctx, selectedLists); err != nil {
		return err
	}

	return nil
}

// scheduleJob start list updater in the background
func (s *Service) scheduleJob() error {
	identifierKey := "lists-updater"

	job := NewRefreshListsJob(s.log.With().Str("job", identifierKey).Logger(), s)

	// schedule job to run every 6th hour
	id, err := s.scheduler.AddJob(job, "0 */6 * * *", identifierKey)
	if err != nil {
		return err
	}

	s.log.Debug().Int("job_id", id).Msg("scheduled job")

	return nil
}

func (s *Service) Start() error {
	if err := s.scheduleJob(); err != nil {
		s.log.Error().Err(err).Msg("error while scheduling job")
		return err
	}

	return nil
}
