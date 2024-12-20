package list

import (
	"context"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func (s *service) sonarr(ctx context.Context, list *domain.List) error {
	var arrType string
	if list.Type == domain.ListTypeWhisparr {
		arrType = "whisparr"
	} else {
		arrType = "sonarr"
	}

	l := log.With().Str("type", arrType).Str("client", list.Name).Logger()

	l.Debug().Msgf("gathering titles...")

	titles, err := s.processSonarr(ctx, list, &l)
	if err != nil {
		return err
	}

	l.Debug().Msgf("got %d filter titles", len(titles))

	joinedTitles := strings.Join(titles, ",")

	l.Trace().Msgf("%v", joinedTitles)

	if len(joinedTitles) == 0 {
		return nil
	}

	for _, filter := range list.Filters {
		l.Debug().Msgf("updating filter: %v", filter.ID)

		f := domain.FilterUpdate{Shows: &joinedTitles}

		if list.MatchRelease {
			f = domain.FilterUpdate{MatchReleases: &joinedTitles}
		}

		f.ID = filter.ID

		if err := s.filterSvc.UpdatePartial(ctx, f); err != nil {
			return errors.Wrap(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}

func (s *service) processSonarr(ctx context.Context, list *domain.List, logger *zerolog.Logger) ([]string, error) {
	downloadClient, err := s.downloadClientSvc.GetClient(ctx, int32(list.ClientID))
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", list.ClientID)
	}

	if !downloadClient.Enabled {
		return nil, errors.New("client %s %s not enabled", downloadClient.Type, downloadClient.Name)
	}

	client := downloadClient.Client.(sonarr.Client)

	var tags []*arr.Tag
	if len(list.TagsExclude) > 0 || len(list.TagsInclude) > 0 {
		t, err := client.GetTags(ctx)
		if err != nil {
			logger.Debug().Msg("could not get tags")
		}
		tags = t
	}

	shows, err := client.GetAllSeries(ctx)
	if err != nil {
		return nil, err
	}

	logger.Debug().Msgf("found %d shows to process", len(shows))

	titleSet := make(map[string]struct{})
	var processedTitles int

	for _, show := range shows {
		if !list.ShouldProcessItem(show.Monitored) {
			continue
		}

		//if !s.shouldProcessItem(show.Monitored, list) {
		//	continue
		//}

		if len(list.TagsInclude) > 0 {
			if len(show.Tags) == 0 {
				continue
			}
			if !containsTag(tags, show.Tags, list.TagsInclude) {
				continue
			}
		}

		if len(list.TagsExclude) > 0 {
			if containsTag(tags, show.Tags, list.TagsExclude) {
				continue
			}
		}

		processedTitles++

		titles := processTitle(show.Title, list.MatchRelease)
		for _, title := range titles {
			titleSet[title] = struct{}{}
		}

		if !list.ExcludeAlternateTitles {
			for _, title := range show.AlternateTitles {
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
	logger.Debug().Msgf("from a total of %d shows we found %d titles and created %d release titles", len(shows), processedTitles, len(uniqueTitles))

	return uniqueTitles, nil
}
