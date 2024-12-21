package list

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/pkg/errors"
)

func (s *service) plaintext(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("type", "plaintext").Str("list", list.Name).Logger()

	if list.URL == "" {
		errMsg := "no URL provided for plaintext"
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
		l.Error().Msgf("failed to fetch titles from URL: %s", list.URL)
		return fmt.Errorf("failed to fetch titles from URL: %s", list.URL)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		l.Error().Msgf("failed to fetch plaintext from URL: %s", list.URL)
		return fmt.Errorf("failed to fetch plaintext from URL: %s", list.URL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Error().Err(err).Msgf("failed to read response body from URL: %s", list.URL)
		return err
	}

	var titles []string
	titleLines := strings.Split(string(body), "\n")
	for _, titleLine := range titleLines {
		title := strings.TrimSpace(titleLine)
		if title == "" {
			continue
		}
		titles = append(titles, title)
	}

	filterTitles := []string{}
	for _, title := range titles {
		filterTitles = append(filterTitles, processTitle(title, list.MatchRelease)...)
	}

	joinedTitles := strings.Join(filterTitles, ",")

	l.Trace().Msgf("%s", joinedTitles)

	if len(joinedTitles) == 0 {
		l.Debug().Msgf("no titles found to update for list: %v", list.Name)
		return nil
	}

	filterUpdate := domain.FilterUpdate{Shows: &joinedTitles}

	if list.MatchRelease {
		filterUpdate.Shows = nil
		filterUpdate.MatchReleases = &joinedTitles
	}

	for _, filter := range list.Filters {
		l.Debug().Msgf("updating filter: %v", filter.ID)

		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, filterUpdate); err != nil {
			return errors.Wrapf(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}
