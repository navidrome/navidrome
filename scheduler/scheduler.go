package scheduler

import (
	"context"

	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/robfig/cron/v3"
)

type Scheduler interface {
	Run(ctx context.Context)
	Add(crontab string, cmd func()) (int, error)
	Remove(id int)
}

func GetInstance() Scheduler {
	return singleton.GetInstance(func() *scheduler {
		c := cron.New(cron.WithLogger(&logger{}))
		return &scheduler{
			c: c,
		}
	})
}

type scheduler struct {
	c *cron.Cron
}

func (s *scheduler) Run(ctx context.Context) {
	s.c.Start()
	<-ctx.Done()
	s.c.Stop()
}

func (s *scheduler) Add(crontab string, cmd func()) (int, error) {
	entryID, err := s.c.AddFunc(crontab, cmd)
	if err != nil {
		return 0, err
	}
	return int(entryID), nil
}

func (s *scheduler) Remove(id int) {
	s.c.Remove(cron.EntryID(id))
}
