package core

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
	c.Logger.Errorf("msg: %v, error: %v", msg, err) // TODO, pass in the keysAndValues
}
