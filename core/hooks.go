package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

var LogFolder = "/tmp"

type Hook func(*Scheduler, Job, *Execution)

func AfterStartHook(_ *Scheduler, j Job, e *Execution) {
	fmt.Printf(
		"[%s] %s - Job started %q - %s\n",
		e.Date.Format(time.RFC1123), j.GetName(), e.ID, j.GetCommand())
}

func AfterStopHook(_ *Scheduler, j Job, e *Execution) {
	errText := "none"
	if e.Error != nil {
		errText = e.Error.Error()
	}

	fmt.Printf(
		"[%s] %s - Job finished %q in %s, failed: %t, skipped: %t, error: %s\n",
		e.Date.Format(time.RFC1123), j.GetName(), e.ID, e.Duration, e.Failed, e.Skipped, errText,
	)

	js, _ := json.MarshalIndent(NewResult(e), "", "   ")
	log := fmt.Sprintf("%s/%s_%s.json", LogFolder, j.GetName(), e.Date.Format("2006-01-02T15_04_05"))
	ioutil.WriteFile(log, js, 0644)
}

type Result struct {
	ID       string
	Date     time.Time
	Duration string
	Failed   bool
	Skipped  bool
	Error    string `json:",omitempty"`
	Stdout   string `json:",omitempty"`
	Stderr   string `json:",omitempty"`
}

func NewResult(e *Execution) *Result {
	r := &Result{}
	r.ID = e.ID
	r.Date = e.Date
	r.Duration = e.Duration.String()
	r.Failed = e.Failed
	r.Skipped = e.Skipped
	if e.Error != nil {
		r.Error = fmt.Sprintf("%s", e.Error)
	}

	r.Stdout = e.OutputStream.(*bytes.Buffer).String()
	r.Stderr = e.ErrorStream.(*bytes.Buffer).String()
	return r
}
