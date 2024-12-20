package list

import (
	"context"
	"sort"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func (s *service) readarr(ctx context.Context, list *domain.List) error {
	l := log.With().Str("type", "readarr").Str("client", list.Name).Logger()

	l.Debug().Msgf("gathering titles...")

	titles, err := s.processReadarr(ctx, list, &l)
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

		f := domain.FilterUpdate{ID: filter.ID, MatchReleases: &joinedTitles}

		if err := s.filterSvc.UpdatePartial(ctx, f); err != nil {
			return errors.Wrap(err, "error updating filter: %v", filter.ID)
		}

		l.Debug().Msgf("successfully updated filter: %v", filter.ID)
	}

	return nil
}

func (s *service) processReadarr(ctx context.Context, list *domain.List, logger *zerolog.Logger) ([]string, error) {
	downloadClient, err := s.downloadClientSvc.GetClient(ctx, int32(list.ClientID))
	if err != nil {
		return nil, errors.Wrap(err, "could not get client with id %d", list.ClientID)
	}

	if !downloadClient.Enabled {
		return nil, errors.New("client %s %s not enabled", downloadClient.Type, downloadClient.Name)
	}

	client := downloadClient.Client.(readarr.Client)

	//var tags []*arr.Tag
	//if len(list.TagsExclude) > 0 || len(list.TagsInclude) > 0 {
	//	t, err := client.GetTags(ctx)
	//	if err != nil {
	//		logger.Debug().Msg("could not get tags")
	//	}
	//	tags = t
	//}

	books, err := client.GetBooks(ctx, "")
	if err != nil {
		return nil, err
	}

	logger.Debug().Msgf("found %d books to process", len(books))

	var titles []string
	var processedTitles int

	for _, book := range books {
		if !list.ShouldProcessItem(book.Monitored) {
			continue
		}

		//if len(list.TagsInclude) > 0 {
		//	if len(book.Tags) == 0 {
		//		continue
		//	}
		//	if !containsTag(tags, book.Tags, list.TagsInclude) {
		//		continue
		//	}
		//}
		//
		//if len(list.TagsExclude) > 0 {
		//	if containsTag(tags, book.Tags, list.TagsExclude) {
		//		continue
		//	}
		//}

		processedTitles++

		titles = append(titles, processTitle(book.Title, list.MatchRelease)...)
	}

	sort.Strings(titles)
	logger.Debug().Msgf("from a total of %d books we found %d titles and created %d release titles", len(books), processedTitles, len(titles))

	return titles, nil
}
