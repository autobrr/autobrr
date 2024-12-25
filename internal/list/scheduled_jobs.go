package list

import (
	"context"
	"math/rand"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Job interface {
	cron.Job
	RunE(ctx context.Context) error
}

type RefreshListSvc interface {
	RefreshAll(ctx context.Context) error
}

type RefreshListsJob struct {
	log     zerolog.Logger
	listSvc RefreshListSvc
}

func NewRefreshListsJob(log zerolog.Logger, listSvc RefreshListSvc) Job {
	return &RefreshListsJob{log: log, listSvc: listSvc}
}

func (job *RefreshListsJob) Run() {
	ctx := context.Background()
	if err := job.RunE(ctx); err != nil {
		job.log.Error().Err(err).Msg("error refreshing lists")
	}
}

func (job *RefreshListsJob) RunE(ctx context.Context) error {
	if err := job.run(ctx); err != nil {
		job.log.Error().Err(err).Msg("error refreshing lists")
		return err
	}

	return nil
}

func (job *RefreshListsJob) run(ctx context.Context) error {
	job.log.Debug().Msg("running refresh lists job")

	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate a random duration between 1 and 35 seconds
	delay := time.Duration(rand.Intn(35-1+1)+1) * time.Second // (35-1+1)+1 => range: 1 to 35

	job.log.Debug().Msgf("delaying for %v...", delay)

	// Sleep for the calculated duration
	time.Sleep(delay)

	if err := job.listSvc.RefreshAll(ctx); err != nil {
		job.log.Error().Err(err).Msg("error refreshing lists")
		return err
	}

	job.log.Debug().Msg("finished refresh lists job")

	return nil
}
