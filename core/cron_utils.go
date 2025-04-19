package core

import (
	"log/slog"
)

// Implement the cron logger interface
type CronUtils struct {
	Logger Logger
}

func NewCronUtils(l Logger) *CronUtils {
	return &CronUtils{Logger: l}
}

func formatKeysAndValues(keysAndValues ...interface{}) string {
	r := slog.Record{}
	r.Add(keysAndValues...)

	attrs := []slog.Attr{}
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	return slog.GroupValue(attrs...).String()
}

func (c *CronUtils) Info(msg string, keysAndValues ...interface{}) {
	c.Logger.Debugf("%v", formatKeysAndValues(append([]interface{}{"cron", msg}, keysAndValues...)...))
}

func (c *CronUtils) Error(err error, msg string, keysAndValues ...interface{}) {
	c.Logger.Errorf("%v", formatKeysAndValues(append([]interface{}{"cron", msg, "error", err}, keysAndValues...)...))
}
