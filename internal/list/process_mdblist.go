package list

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func (s *service) mdblist(ctx context.Context, list *domain.List) error {
	l := log.With().Str("type", "mdblist").Str("list", list.Name).Logger()

	if list.URL == "" {
		errMsg := "no URL provided for Mdblist"
		l.Error().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	//var titles []string

	//green := color.New(color.FgGreen).SprintFunc()
	l.Debug().Msgf("fetching titles from %s", list.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
	if err != nil {
		l.Error().Err(err).Msg("could not make new request")
		return err
	}

	//for k, v := range list.Headers {
	//	req.Header.Set(k, v)
	//}

	//setUserAgent(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		l.Error().Err(err).Msgf("failed to fetch titles from URL: %s", list.URL)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		l.Error().Msgf("failed to fetch titles from URL: %s", list.URL)
		return fmt.Errorf("failed to fetch titles from URL: %s", list.URL)
	}

	var data []struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		l.Error().Err(err).Msgf("failed to decode JSON data from URL: %s", list.URL)
		return err
	}

	filterTitles := []string{}
	for _, item := range data {
		//titles = append(titles, item.Title)
		filterTitles = append(filterTitles, processTitle(item.Title, list.MatchRelease)...)
	}

	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Msgf("%s", joinedTitles)

	if len(joinedTitles) == 0 {
		//l.Debug().Msgf("no titles found for filter: %v", filterID)
		return nil
	}

	for _, filterID := range list.Filters {
		l.Debug().Msgf("updating filter: %v", filterID)

		//filterTitles := []string{}
		//for _, title := range titles {
		//	filterTitles = append(filterTitles, processTitle(title, list.MatchRelease)...)
		//}

		//joinedTitles := strings.Join(filterTitles, ",")

		//l.Trace().Msgf("%s", joinedTitles)

		//if len(joinedTitles) == 0 {
		//	l.Debug().Msgf("no titles found for filter: %v", filterID)
		//	return nil
		//}

		f := domain.FilterUpdate{Shows: &joinedTitles}

		if list.MatchRelease {
			f = domain.FilterUpdate{MatchReleases: &joinedTitles}
		}

		f.ID = filterID

		if err := s.filterSvc.UpdatePartial(ctx, f); err != nil {
			l.Error().Err(err).Msgf("error updating filter: %v", filterID)
			return errors.Wrapf(err, "error updating filter: %v", filterID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filterID)
	}

	return nil
}
