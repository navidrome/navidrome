package scheduler

import (
	"context"

	"github.com/robfig/cron/v3"
)

type Scheduler interface {
	Run(ctx context.Context)
	Add(crontab string, cmd func()) error
}

func New() Scheduler {
	c := cron.New(cron.WithLogger(&logger{}))
	return &scheduler{
		c: c,
	}
}

type scheduler struct {
	c *cron.Cron
}

func (s *scheduler) Run(ctx context.Context) {
	s.c.Start()
	<-ctx.Done()
	s.c.Stop()
}

func (s *scheduler) Add(crontab string, cmd func()) error {
	_, err := s.c.AddFunc(crontab, cmd)
	return err
}
