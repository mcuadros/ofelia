# Ofelia - a job scheduler [![GitHub version](https://badge.fury.io/gh/mcuadros%2Fofelia.svg)](https://github.com/mcuadros/ofelia/releases) [![codecov.io](http://codecov.io/github/mcuadros/ofelia/coverage.svg?branch=master)](http://codecov.io/github/mcuadros/ofelia?branch=master) [![Build Status](https://travis-ci.org/mcuadros/ofelia.svg)](https://travis-ci.org/mcuadros/ofelia)

<img src="https://weirdspace.dk/FranciscoIbanez/Graphics/Ofelia.gif" align="right" width="180px" height="300px" vspace="20" />


**Ofelia** is a modern and low footprint job scheduler for __docker__ environments, built on Go. Ofelia aims to be a replacement for the old fashioned [cron](https://en.wikipedia.org/wiki/Cron).

### Why?

It has been a long time since [`cron`](https://en.wikipedia.org/wiki/Cron) was released, actually more than 28 years. The world has changed a lot and especially since the `Docker` revolution. **Vixie's cron** works great but it's not extensible and it's hard to debug when something goes wrong.

Many solutions are available: ready to go containerized `crons`, wrappers for your commands, etc. but in the end simple tasks become complex.

### How?

The main feature of **Ofelia** is the ability to execute commands directly on Docker containers. Using Docker's API Ofelia emulates the behavior of [`exec`](https://docs.docker.com/reference/commandline/exec/), be in able to run a command inside of a running container. Also you can run the command in a new container destroying it at the end of the execution.

## Configuration

### Jobs
It uses a INI-style config file and the [scheduling format](https://godoc.org/github.com/robfig/cron) is exactly the same from the original `cron`, you can configure three different kind of jobs:

- `job-exec`: this job is executed inside of a running container.
- `job-run`: runs a command inside of a new container, using a specific image.
- `job-local`: runs the command inside of the host running ofelia.
- `job-service-run`: runs the command inside a new "run-once" service, for running inside a swarm


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


[job-service-run "service-executed-on-new-container"]
schedule = 0,20,40 * * * *
image = ubuntu
network = swarm_network
command =  touch /tmp/example
```

### Logging
**Ofelia** comes with three different logging drivers that can be configured in the `[global]` section:
- `mail` to send mails
- `save` to save structured execution reports to a directory
- `slack` to send messages via a slack webhook

#### Options
- `smtp-host` - address of the SMTP server.
- `smtp-port` - port number of the SMTP server.
- `smtp-user` - user name used to connect to the SMTP server.
- `smtp-password` - password used to connect to the SMTP server.
- `email-to` - mail address of the receiver of the mail.
- `email-from` - mail address of the sender of the mail.
- `mail-only-on-error` - only send a mail if the execution was not successful.

- `save-folder` - directory in which the reports shall be written.
- `save-only-on-error` - only save a report if the execution was not successful.

- `slack-webhook` - URL of the slack webhook.
- `slack-only-on-error` - only send a slack message if the execution was not successful.

### Overlap
**Ofelia** can prevent that a job is run twice in parallel (e.g. if the first execution didn't complete before a second execution was scheduled. If a job has the option `no-overlap` set, it will not be run concurrently. 

## Installation

The easiest way to deploy **ofelia** is using *Docker*.

```sh
docker run -it -v /etc/ofelia:/etc/ofelia mcuadros/ofelia:latest
```

Don't forget to place your `config.ini` at your host machine.

If don't want to run **ofelia** using our *Docker* image you can download a binary from [releases](https://github.com/mcuadros/ofelia/releases) page.



> Why the project is named Ofelia? Ofelia is the name of the office assistant from the Spanish comic [Mortadelo y Filemón](https://en.wikipedia.org/wiki/Mort_%26_Phil)
