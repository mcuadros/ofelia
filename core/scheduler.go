package core

import (
	"errors"
	"fmt"
	"sync"

	"github.com/robfig/cron"
)

var ErrEmptyScheduler error = errors.New("unable to start a empty scheduler.")
var ErrEmptySchedule error = errors.New("unable to add a job with a empty schedule.")

type Scheduler struct {
	Jobs   []Job
	Logger Logger

	middlewareContainer
	cron      *cron.Cron
	wg        sync.WaitGroup
	isRunning bool
}

func NewScheduler(l Logger) *Scheduler {
	return &Scheduler{
		Logger: l,
		cron:   cron.New(),
	}
}

func (s *Scheduler) AddJob(j Job) error {
	s.Logger.Notice("New job registered %q - %q - %q", j.GetName(), j.GetCommand(), j.GetSchedule())

	if j.GetSchedule() == "" {
		return ErrEmptySchedule
	}

	err := s.cron.AddJob(j.GetSchedule(), &jobWrapper{s, j})
	if err != nil {
		return err
	}

	s.Jobs = append(s.Jobs, j)
	return nil
}

func (s *Scheduler) Start() error {
	if len(s.Jobs) == 0 {
		return ErrEmptyScheduler
	}

	s.Logger.Debug("Starting scheduler with %d jobs", len(s.Jobs))

	s.mergeMiddlewares()
	s.isRunning = true
	s.cron.Start()
	return nil
}

func (s *Scheduler) mergeMiddlewares() {
	for _, j := range s.Jobs {
		j.Use(s.Middlewares()...)
	}
}

func (s *Scheduler) Stop() error {
	s.wg.Wait()
	s.cron.Stop()
	s.isRunning = false

	return nil
}

func (s *Scheduler) IsRunning() bool {
	return s.isRunning
}

type jobWrapper struct {
	s *Scheduler
	j Job
}

func (w *jobWrapper) Run() {
	w.s.wg.Add(1)
	defer w.s.wg.Done()

	e := NewExecution()
	ctx := NewContext(w.s, w.j, e)

	w.start(ctx)
	err := ctx.Next()
	w.stop(ctx, err)
}

func (w *jobWrapper) start(ctx *Context) {
	ctx.Start()

	ctx.Logger.Debug(
		"%s - Job started %q - %q",
		ctx.Job.GetName(), ctx.Execution.ID, ctx.Job.GetCommand(),
	)
}

func (w *jobWrapper) stop(ctx *Context, err error) {
	ctx.Stop(err)

	errText := "none"
	if ctx.Execution.Error != nil {
		errText = ctx.Execution.Error.Error()
	}

	msg := fmt.Sprintf(
		"%s - Job finished %q in %q, failed: %t, skipped: %t, error: %s",
		ctx.Job.GetName(), ctx.Execution.ID, ctx.Execution.Duration, ctx.Execution.Failed, ctx.Execution.Skipped, errText,
	)

	if ctx.Execution.Failed {
		ctx.Logger.Error(msg)
	} else if ctx.Execution.Skipped {
		ctx.Logger.Warning(msg)
	} else {
		ctx.Logger.Notice(msg)
	}
}
