package core

import (
	"errors"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	ErrEmptyScheduler = errors.New("unable to start a empty scheduler")
	ErrEmptySchedule  = errors.New("unable to add a job with a empty schedule")
)

type Scheduler struct {
	Logger Logger

	middlewareContainer
	cron      *cron.Cron
	wg        sync.WaitGroup
	isRunning bool
}

func NewScheduler(l Logger) *Scheduler {
	cronUtils := NewCronUtils(l)
	return &Scheduler{
		Logger: l,
		cron: cron.New(
			cron.WithLogger(cronUtils),
			cron.WithChain(cron.Recover(cronUtils)),
			cron.WithParser(cron.NewParser(
				// For backward compatibility with cron/v1 configure optional seconds field
				// https://github.com/robfig/cron?tab=readme-ov-file#upgrading-to-v3-june-2019
				cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor),
			),
		),
	}
}

func (s *Scheduler) AddJob(j Job) error {
	if j.GetSchedule() == "" {
		return ErrEmptySchedule
	}

	id, err := s.cron.AddJob(j.GetSchedule(), &jobWrapper{s, j})
	if err != nil {
		s.Logger.Warning("Failed to register job", "job", j.GetName(), "command", j.GetCommand(), "schedule", j.GetSchedule(), "error", err)
		return err
	}

	j.SetCronJobID(int(id))
	j.Use(s.Middlewares()...)
	s.Logger.Info("New job registered", "job", j.GetName(), "command", j.GetCommand(), "schedule", j.GetSchedule(), "cron_id", id)
	return nil
}

func (s *Scheduler) RemoveJob(j Job) error {
	s.Logger.Info("Job deregistered (will not fire again)", "job", j.GetName(), "command", j.GetCommand(), "schedule", j.GetSchedule(), "cron_id", j.GetCronJobID())
	s.cron.Remove(cron.EntryID(j.GetCronJobID()))
	return nil
}

func (s *Scheduler) CronJobs() []cron.Entry {
	return s.cron.Entries()
}

func (s *Scheduler) Start() error {
	s.Logger.Debug("Starting scheduler", "jobs_count", len(s.CronJobs()))

	s.isRunning = true
	s.cron.Start()
	return nil
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
	ctx.Log("Job start", "command", ctx.Job.GetCommand())
}

func (w *jobWrapper) stop(ctx *Context, err error) {
	ctx.Stop(err)

	args := []any{}
	if ctx.Execution.Error != nil {
		args = append(args, "error", ctx.Execution.Error)
	}

	if ctx.Execution.OutputStream.TotalWritten() > 0 {
		args = append(args, "stdout", ctx.Execution.OutputStream)
	}

	if ctx.Execution.ErrorStream.TotalWritten() > 0 {
		args = append(args, "stderr", ctx.Execution.ErrorStream)
	}

	args = append(args, "duration", ctx.Execution.Duration, "failed", ctx.Execution.Failed, "skipped", ctx.Execution.Skipped)

	ctx.Log("Job stop", args...)
}
