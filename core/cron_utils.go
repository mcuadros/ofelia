package core

import "github.com/robfig/cron/v3"

/*
type Logger interface {
    // Info logs routine messages about cron's operation.
    Info(msg string, keysAndValues ...interface{})
    // Error logs an error condition.
    Error(err error, msg string, keysAndValues ...interface{})
}
	Criticalf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Noticef(format string, args ...interface{})
	Warningf(format string, args ...interface{})

*/

// Implement the cron logger interface
type CronUtils struct {
	Logger Logger
}

func NewCronUtils(l Logger) *CronUtils {
	return &CronUtils{Logger: l}
}

func (c *CronUtils) Info(msg string, keysAndValues ...interface{}) {
	c.Logger.Debugf(msg) // TODO, pass in the keysAndValues
}

func (c *CronUtils) Error(err error, msg string, keysAndValues ...interface{}) {
	c.Logger.Errorf(msg) // TODO, pass in the keysAndValues
}

// TODO: Implement middlewares here
func (c *CronUtils) ApplyMiddleware() cron.JobWrapper {
	return func(j cron.Job) cron.Job {
		return cron.FuncJob(func() {
			j.Run()
		})
	}
}
