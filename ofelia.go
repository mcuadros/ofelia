package main

import (
	"fmt"

	"github.com/mcuadros/ofelia/core"

	"github.com/fsouza/go-dockerclient"
	"github.com/mcuadros/go-defaults"
	"gopkg.in/gcfg.v1"
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
	if err := config.LoadFile("config.ini"); err != nil {
		panic(err)
	}

	fmt.Printf("Loaded %d job(s)\n", len(config.Jobs))

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

	select {}
}
