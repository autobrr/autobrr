// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	stdErr "errors"
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
	List(ctx context.Context) ([]*domain.List, error)
	FindByID(ctx context.Context, id int64) (*domain.List, error)
	Store(ctx context.Context, list *domain.List) error
	Update(ctx context.Context, list *domain.List) error
	Delete(ctx context.Context, id int64) error
	RefreshAll(ctx context.Context) error
	RefreshList(ctx context.Context, listID int64) error
	RefreshArrLists(ctx context.Context) error
	RefreshOtherLists(ctx context.Context) error
	Start()
}

type service struct {
	log  zerolog.Logger
	repo domain.ListRepo

	httpClient        *http.Client
	scheduler         scheduler.Service
	downloadClientSvc download_client.Service
	filterSvc         filter.Service
}

func NewService(log logger.Logger, repo domain.ListRepo, downloadClientSvc download_client.Service, filterSvc filter.Service, schedulerSvc scheduler.Service) Service {
	return &service{
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

func (s *service) List(ctx context.Context) ([]*domain.List, error) {
	data, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// attach filters
	for _, list := range data {
		filters, err := s.repo.GetListFilters(ctx, list.ID)
		if err != nil {
			return nil, err
		}

		list.Filters = filters
	}

	return data, nil
}

func (s *service) FindByID(ctx context.Context, id int64) (*domain.List, error) {
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

func (s *service) Store(ctx context.Context, list *domain.List) error {
	if err := list.Validate(); err != nil {
		s.log.Error().Err(err).Msgf("could not validate list %s", list.Name)
		return err
	}

	if err := s.repo.Store(ctx, list); err != nil {
		s.log.Error().Err(err).Msgf("could not store list %s", list.Name)
		return err
	}

	s.log.Debug().Msgf("successfully created list %s", list.Name)

	if list.Enabled {
		if err := s.refreshList(ctx, list); err != nil {
			s.log.Error().Err(err).Msgf("could not refresh list %s", list.Name)
			return err
		}
	}

	return nil
}

func (s *service) Update(ctx context.Context, list *domain.List) error {
	if err := list.Validate(); err != nil {
		s.log.Error().Err(err).Msgf("could not validate list %s", list.Name)
		return err
	}

	if err := s.repo.Update(ctx, list); err != nil {
		s.log.Error().Err(err).Msgf("could not update list %s", list.Name)
		return err
	}

	s.log.Debug().Msgf("successfully updated list %s", list.Name)

	if list.Enabled {
		if err := s.refreshList(ctx, list); err != nil {
			s.log.Error().Err(err).Msgf("could not refresh list %s", list.Name)
			return err
		}
	}

	return nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not delete list by id %d", id)
		return err
	}

	s.log.Debug().Msgf("successfully deleted list %d", id)

	return nil
}

func (s *service) RefreshAll(ctx context.Context) error {
	lists, err := s.List(ctx)
	if err != nil {
		return err
	}

	s.log.Debug().Msgf("found %d lists to refresh", len(lists))

	if err := s.refreshAll(ctx, lists); err != nil {
		return err
	}

	s.log.Debug().Msgf("successfully refreshed all lists")

	return nil
}

func (s *service) refreshAll(ctx context.Context, lists []*domain.List) error {
	var processingErrors []error

	for _, listItem := range lists {
		if !listItem.Enabled {
			s.log.Debug().Msgf("list %s is disabled, skipping...", listItem.Name)
			continue
		}

		if err := s.refreshList(ctx, listItem); err != nil {
			s.log.Error().Err(err).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msgf("error while refreshing %s, continuing with other lists", listItem.Type)

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

func (s *service) refreshList(ctx context.Context, listItem *domain.List) error {
	s.log.Debug().Msgf("refresh list %s - %s", listItem.Type, listItem.Name)

	var err error

	switch listItem.Type {
	case domain.ListTypeRadarr:
		err = s.radarr(ctx, listItem)

	case domain.ListTypeSonarr:
		err = s.sonarr(ctx, listItem)

	case domain.ListTypeWhisparr:
		err = s.sonarr(ctx, listItem)

	case domain.ListTypeReadarr:
		err = s.readarr(ctx, listItem)

	case domain.ListTypeLidarr:
		err = s.lidarr(ctx, listItem)

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

	default:
		err = errors.Errorf("unsupported list type: %s", listItem.Type)
	}

	if err != nil {
		s.log.Error().Err(err).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msgf("error refreshing %s list", listItem.Name)

		// update last run for list and set errs and status
		listItem.LastRefreshStatus = domain.ListRefreshStatusError
		listItem.LastRefreshData = err.Error()
		listItem.LastRefreshTime = time.Now()

		if updateErr := s.repo.UpdateLastRefresh(ctx, listItem); updateErr != nil {
			s.log.Error().Err(updateErr).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msgf("error updating last refresh for %s list", listItem.Name)
			return updateErr
		}

		return err
	}

	listItem.LastRefreshStatus = domain.ListRefreshStatusSuccess
	//listItem.LastRefreshData = err.Error()
	listItem.LastRefreshTime = time.Now()

	if updateErr := s.repo.UpdateLastRefresh(ctx, listItem); updateErr != nil {
		s.log.Error().Err(updateErr).Str("type", string(listItem.Type)).Str("list", listItem.Name).Msgf("error updating last refresh for %s list", listItem.Name)
		return updateErr
	}

	s.log.Debug().Msgf("successfully refreshed list %s", listItem.Name)

	return nil
}

func (s *service) RefreshList(ctx context.Context, listID int64) error {
	list, err := s.FindByID(ctx, listID)
	if err != nil {
		return err
	}

	if err := s.refreshList(ctx, list); err != nil {
		return err
	}

	return nil
}

func (s *service) RefreshArrLists(ctx context.Context) error {
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

func (s *service) RefreshOtherLists(ctx context.Context) error {
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
func (s *service) scheduleJob() error {
	identifierKey := "lists-updater"

	job := NewRefreshListsJob(s.log.With().Str("job", identifierKey).Logger(), s)

	// schedule job to run every 6th hour
	id, err := s.scheduler.AddJob(job, "0 */6 * * *", identifierKey)
	if err != nil {
		return err
	}

	s.log.Debug().Msgf("scheduled job with id %d", id)

	return nil
}

func (s *service) Start() {
	if err := s.scheduleJob(); err != nil {
		s.log.Error().Err(err).Msg("error while scheduling job")
	}
}
