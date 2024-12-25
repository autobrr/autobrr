package list

import (
	"context"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/radarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

func (s *service) radarr(ctx context.Context, list *domain.List) error {
	l := s.log.With().Str("list", list.Name).Str("type", "radarr").Int("client", list.ClientID).Logger()

	l.Debug().Msgf("gathering titles...")

	titles, err := s.processRadarr(ctx, list, &l)
	if err != nil {
		return err
	}

	l.Debug().Msgf("got %d filter titles", len(titles))

	if len(titles) == 0 {
		l.Debug().Msgf("no titles found to update for list: %v", list.Name)
		return nil
	}

	joinedTitles := strings.Join(titles, ",")

	l.Trace().Str("titles", joinedTitles).Msgf("found %d titles", len(joinedTitles))

	filterUpdate := domain.FilterUpdate{Shows: &joinedTitles}

	if list.MatchRelease {
		filterUpdate.Shows = &nullString
		filterUpdate.MatchReleases = &joinedTitles
	}

	for _, filter := range list.Filters {
		l.Debug().Msgf("updating filter: %v", filter.ID)

		filterUpdate.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, filterUpdate); err != nil {
			return errors.Wrap(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}

func (s *service) processRadarr(ctx context.Context, list *domain.List, logger *zerolog.Logger) ([]string, error) {
	downloadClient, err := s.downloadClientSvc.GetClient(ctx, int32(list.ClientID))
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", list.ClientID)
	}

	if !downloadClient.Enabled {
		return nil, errors.New("client %s %s not enabled", downloadClient.Type, downloadClient.Name)
	}

	client := downloadClient.Client.(*radarr.Client)

	var tags []*arr.Tag
	if len(list.TagsExclude) > 0 || len(list.TagsInclude) > 0 {
		t, err := client.GetTags(ctx)
		if err != nil {
			logger.Debug().Msg("could not get tags")
		}
		tags = t
	}

	movies, err := client.GetMovies(ctx, 0)
	if err != nil {
		return nil, err
	}

	logger.Debug().Msgf("found %d movies to process", len(movies))

	titleSet := make(map[string]struct{})
	var processedTitles int

	for _, movie := range movies {
		if !list.ShouldProcessItem(movie.Monitored) {
			continue
		}

		//if !s.shouldProcessItem(movie.Monitored, list) {
		//	continue
		//}

		if len(list.TagsInclude) > 0 {
			if len(movie.Tags) == 0 {
				continue
			}
			if !containsTag(tags, movie.Tags, list.TagsInclude) {
				continue
			}
		}

		if len(list.TagsExclude) > 0 {
			if containsTag(tags, movie.Tags, list.TagsExclude) {
				continue
			}
		}

		processedTitles++

		// Taking the international title and the original title and appending them to the titles array.
		for _, title := range []string{movie.Title, movie.OriginalTitle} {
			if title != "" {
				for _, t := range processTitle(title, list.MatchRelease) {
					titleSet[t] = struct{}{}
				}
			}
		}

		if list.IncludeAlternateTitles {
			for _, title := range movie.AlternateTitles {
				altTitles := processTitle(title.Title, list.MatchRelease)
				for _, altTitle := range altTitles {
					titleSet[altTitle] = struct{}{}
				}
			}
		}
	}

	uniqueTitles := make([]string, 0, len(titleSet))
	for title := range titleSet {
		uniqueTitles = append(uniqueTitles, title)
	}

	sort.Strings(uniqueTitles)
	logger.Debug().Msgf("from a total of %d movies we found %d titles and created %d release titles", len(movies), processedTitles, len(uniqueTitles))

	return uniqueTitles, nil
}

var nullString = ""
