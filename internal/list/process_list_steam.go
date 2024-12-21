package list

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
)

func (s *service) steam(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "steam").Str("list", list.Name).Logger()

	if list.URL == "" {
		errMsg := "no URL provided for steam"
		l.Error().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	l.Debug().Msgf("fetching titles from %s", list.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if err != nil {
		l.Error().Err(err).Msg("could not make new request")
		return err
	}

	list.SetRequestHeaders(req)

	//setUserAgent(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		l.Error().Err(err).Msgf("failed to fetch titles from URL: %s", list.URL)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		l.Error().Msg("failed to fetch titles, non-OK HTTP status received")
		return fmt.Errorf("failed to fetch titles, non-OK HTTP status received")
	}

	var data map[string]struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		l.Error().Err(err).Msgf("failed to decode JSON data from URL: %s", list.URL)
		return err
	}

	var titles []string
	for _, item := range data {
		titles = append(titles, item.Name)
	}

	filterTitles := []string{}
	for _, title := range titles {
		filterTitles = append(filterTitles, processTitle(title, list.MatchRelease)...)
	}

	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Msgf("%s", joinedTitles)

	if len(joinedTitles) == 0 {
		l.Debug().Msgf("no titles found for list to update: %v", list.Name)
		return nil
	}

	filterUpdate := domain.FilterUpdate{MatchReleases: &joinedTitles}

	for _, filter := range list.Filters {
		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, filterUpdate); err != nil {
			return errors.Wrapf(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}
