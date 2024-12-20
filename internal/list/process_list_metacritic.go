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

func (s *service) metacritic(ctx context.Context, list *domain.List) error {
	l := log.With().Str("type", "metacritic").Str("list", list.Name).Logger()

	if list.URL == "" {
		errMsg := "no URL provided for metacritic"
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

	if resp.StatusCode == http.StatusNotFound {
		errMsg := fmt.Sprintf("No endpoint found at %v. (404 Not Found)", list.URL)
		l.Error().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if resp.StatusCode != http.StatusOK {
		l.Error().Msgf("failed to fetch titles from URL: %s", list.URL)
		return fmt.Errorf("failed to fetch titles from URL: %s", list.URL)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		errMsg := fmt.Sprintf("invalid content type for URL: %s, content type should be application/json", list.URL)
		return fmt.Errorf(errMsg)
	}

	var data struct {
		Title  string `json:"title"`
		Albums []struct {
			Artist string `json:"artist"`
			Title  string `json:"title"`
		} `json:"albums"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		l.Error().Err(err).Msgf("failed to decode JSON data from URL: %s", list.URL)
		return err
	}

	var titles []string
	var artists []string

	for _, album := range data.Albums {
		titles = append(titles, album.Title)
		artists = append(artists, album.Artist)
	}

	// Deduplicate artists
	uniqueArtists := []string{}
	seenArtists := map[string]struct{}{}
	for _, artist := range artists {
		if _, ok := seenArtists[artist]; !ok {
			uniqueArtists = append(uniqueArtists, artist)
			seenArtists[artist] = struct{}{}
		}
	}

	filterTitles := []string{}
	for _, title := range titles {
		filterTitles = append(filterTitles, processTitle(title, list.MatchRelease)...)
	}

	filterArtists := []string{}
	for _, artist := range uniqueArtists {
		filterArtists = append(filterArtists, processTitle(artist, list.MatchRelease)...)
	}

	joinedArtists := strings.Join(filterArtists, ",")
	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Msgf("%s", joinedTitles)

	if len(joinedTitles) == 0 {
		l.Debug().Msgf("no titles found to update filter: %v", list.Name)
		return nil
	}

	for _, filterID := range list.Filters {
		l.Debug().Msgf("updating filter: %v", filterID)

		f := domain.FilterUpdate{Albums: &joinedTitles, Artists: &joinedArtists}

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
