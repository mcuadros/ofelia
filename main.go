package main

import (
	"fmt"
	"time"

	"gopkg.in/gcfg.v1"

	"github.com/mcuadros/docker-cron/core"
	"github.com/mcuadros/go-defaults"
	"github.com/tonnerre/golang-pretty"

	"github.com/fsouza/go-dockerclient"
)

type Config struct {
	Jobs map[string]*core.ExecJob `gcfg:"Job"`
}

func (c *Config) LoadFile(filename string) error {
	err := gcfg.ReadFileInto(c, filename)
	if err != nil {
		return err
	}

	c.loadDefaults()
	return nil
}

func (c *Config) loadDefaults() {
	defaults.SetDefaults(c)
	for name, j := range c.Jobs {
		j.Name = name
		defaults.SetDefaults(j)
	}
}

func main() {
	config := &Config{}
	config.LoadFile("config.ini")
	fmt.Printf("%# v\n", pretty.Formatter(config))

	d, err := docker.NewClientFromEnv()
	if err != nil {
		panic(err)
	}

	s := core.NewScheduler()

	for _, j := range config.Jobs {
		j.Client = d
		s.AddJob(j)
	}

	s.Start()

	time.Sleep(time.Hour)
}
