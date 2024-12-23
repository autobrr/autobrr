package list

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
)

func (s *service) steam(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "steam").Str("list", list.Name).Logger()

	if list.URL == "" {
		return errors.New("no URL provided for steam")
	}

	l.Debug().Msgf("fetching titles from %s", list.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if err != nil {
		return errors.Wrapf(err, "could not make new request for URL: %s", list.URL)
	}

	list.SetRequestHeaders(req)

	//setUserAgent(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch titles from URL: %s", list.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to fetch titles, non-OK status recieved: %d", resp.StatusCode)
	}

	var data map[string]struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return errors.Wrapf(err, "failed to decode JSON data from URL: %s", list.URL)
	}

	var titles []string
	for _, item := range data {
		titles = append(titles, item.Name)
	}

	filterTitles := []string{}
	for _, title := range titles {
		filterTitles = append(filterTitles, processTitle(title, list.MatchRelease)...)
	}

	if len(filterTitles) == 0 {
		l.Debug().Msgf("no titles found for list to update: %v", list.Name)
		return nil
	}

	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Str("titles", joinedTitles).Msgf("found %d titles", len(joinedTitles))

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
