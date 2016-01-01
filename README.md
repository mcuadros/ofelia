# Ofelia - a job scheduler [![GitHub version](https://badge.fury.io/gh/mcuadros%2Fofelia.svg)](https://github.com/mcuadros/ofelia/releases) [![codecov.io](http://codecov.io/github/mcuadros/ofelia/coverage.svg?branch=master)](http://codecov.io/github/mcuadros/ofelia?branch=master) [![Build Status](https://travis-ci.org/mcuadros/ofelia.svg)](https://travis-ci.org/mcuadros/ofelia)

<img src="https://weirdspace.dk/FranciscoIbanez/Graphics/Ofelia.gif" align="right" width="180px" height="300px" vspace="20" />


**Ofelia** is a moderm and low footprint job scheduler for __docker__ environments, built on Go. Ofelia aims to be a replacement for the old fashioned [cron](https://en.wikipedia.org/wiki/Cron).

### Why?

It has been a long time since [`cron`](https://en.wikipedia.org/wiki/Cron) was released, actually more than 28 years. The world has changed a lot and especially since the `Docker` revolution. **Vixie's cron** works great but it's not extensible and it's hard to debug when something goes wrong.

Many solutions are available: ready to go containerized `crons`, wrappers for your commands, etc. but in the end simple tasks become complex.   

### How?

The main feature of **Ofelia** is the ability to execute commands directly on Docker containers. Using Docker's API Ofelia emulates the behavior of [`exec`](https://docs.docker.com/reference/commandline/exec/), be in able to run a command inside of a running container. Also you can run the command in a new container destroying it at the end of the execution.

## Configuration

It uses a INI-style config file and the [scheduling format](https://godoc.org/github.com/robfig/cron) is exactly the same from the original `cron`, you can configure three different kind of jobs:

- `job-exec`: this job is executed inside of a running container.
- `job-run`: runs a command inside of a new container, using a specific image.
- `job-local`: runs the command inside of the host running ofelia.


```ini
[job-exec "job-executed-on-running-container"]
schedule = @hourly
container = my-container
command = touch /tmp/example

[job-run "job-executed-on-new-container"]
schedule = @hourly
image = ubuntu:latest
command = touch /tmp/example

[job-local "job-executed-on-current-host"]
schedule = @hourly
command = touch /tmp/example
```

## Installation

The easiest way to deploy **ofelia** is using *Docker*.

```sh
docker run -it -v /etc/ofelia:/etc/ofelia mcuadros/ofelia:latest
```

Don't forget to place your `config.ini` at your host machine.

If don't want to run **ofelia** using our *Docker* image you can download a binary from [releases](https://github.com/mcuadros/ofelia/releases) page.



> Why the project is named Ofelia? Ofelia is the name of the office assistant from the Spanish comic [Mortadelo y Filemón](https://en.wikipedia.org/wiki/Mort_%26_Phil)
