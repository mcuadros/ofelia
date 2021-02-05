# Ofelia - a job scheduler [![GitHub version](https://badge.fury.io/gh/mcuadros%2Fofelia.svg)](https://github.com/mcuadros/ofelia/releases) ![Test](https://github.com/mcuadros/ofelia/workflows/Test/badge.svg)

<img src="https://weirdspace.dk/FranciscoIbanez/Graphics/Ofelia.gif" align="right" width="180px" height="300px" vspace="20" />

**Ofelia** is a modern and low footprint job scheduler for __docker__ environments, built on Go. Ofelia aims to be a replacement for the old fashioned [cron](https://en.wikipedia.org/wiki/Cron).

### Why?

It has been a long time since [`cron`](https://en.wikipedia.org/wiki/Cron) was released, actually more than 28 years. The world has changed a lot and especially since the `Docker` revolution. **Vixie's cron** works great but it's not extensible and it's hard to debug when something goes wrong.

Many solutions are available: ready to go containerized `crons`, wrappers for your commands, etc. but in the end simple tasks become complex.

### How?

The main feature of **Ofelia** is the ability to execute commands directly on Docker containers. Using Docker's API Ofelia emulates the behavior of [`exec`](https://docs.docker.com/reference/commandline/exec/), being able to run a command inside of a running container. Also you can run the command in a new container destroying it at the end of the execution.

## Configuration

### Jobs

[Scheduling format](https://godoc.org/github.com/robfig/cron) is the same as the Go implementation of `cron`. E.g. `@every 10s` or `0 0 1 * * *` (every night at 1 AM).

**Note**: the format starts with seconds, instead of minutes.

you can configure four different kind of jobs:

- `job-exec`: this job is executed inside of a running container.
- `job-run`: runs a command inside of a new container, using a specific image.
- `job-local`: runs the command inside of the host running ofelia.
- `job-service-run`: runs the command inside a new "run-once" service, for running inside a swarm

See [Jobs reference documentation](docs/jobs.md) for all available parameters.

#### INI-style config

Run with `ofelia daemon --config=/path/to/config.ini`

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

#### Docker labels configurations

In order to use this type of configurations, ofelia need access to docker socket.

```sh
docker run -it --rm \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
        mcuadros/ofelia:latest daemon --docker
```

Labels format: `ofelia.<JOB_TYPE>.<JOB_NAME>.<JOB_PARAMETER>=<PARAMETER_VALUE>.
This type of configuration supports all the capabilities provided by INI files.

Also, it is possible to configure `job-exec` by setting labels configurations on the target container. To do that, additional label `ofelia.enabled=true` need to be present on the target container.

For example, we want `ofelia` to execute `uname -a` command in the existing container called `my_nginx`.
To do that, we need to we need to start `my_nginx` container with next configurations:

```sh
docker run -it --rm \
    --label ofelia.enabled=true \
    --label ofelia.job-exec.test-exec-job.schedule="@every 5s" \
    --label ofelia.job-exec.test-exec-job.command="uname -a" \
        nginx
```

Now if we start `ofelia` container with the command provided above, it will execute the task:

- Exec  - `uname -a`

Or with docker-compose:

```yaml
version: "3"
services:
  ofelia:
    image: mcuadros/ofelia:latest
    depends_on:
      - nginx
    command: daemon --docker
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro

  nginx:
    image: nginx
    labels:
      ofelia.enabled: "true"
      ofelia.job-exec.datecron.schedule: "@every 5s"
      ofelia.job-exec.datecron.command: "uname -a"
```

#### Dynamic docker configuration

You can start ofelia in its own container or on the host itself, and it will magically pick up any container that starts, stops or is modified on the fly.
In order to achieve this, you simply have to use docker containers with the labels described above and let ofelia take care of the rest. 

#### Hybrid configuration (INI files + Docker)

You can specify part of the configuration on the INI files, such as globals for the middlewares or even declare tasks in there but also merge them with docker.
The docker labels will be parsed, added and removed on the fly but also, the file config can be used. 

**Use the INI file to:**

- Configure the slack or other middleware integration
- Configure any global setting
- Create a job-run so it executes on a new container each time

```ini
[global]
slack-webhook = https://myhook.com/auth

[job-run "job-executed-on-new-container"]
schedule = @hourly
image = ubuntu:latest
command = touch /tmp/example
```

**Use docker to:**

```sh
docker run -it --rm \
    --label ofelia.enabled=true \
    --label ofelia.job-exec.test-exec-job.schedule="@every 5s" \
    --label ofelia.job-exec.test-exec-job.command="uname -a" \
        nginx
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

The easiest way to deploy **ofelia** is using *Docker*. See examples above.

If don't want to run **ofelia** using our *Docker* image you can download a binary from [releases](https://github.com/mcuadros/ofelia/releases) page.

> Why the project is named Ofelia? Ofelia is the name of the office assistant from the Spanish comic [Mortadelo y Filemón](https://en.wikipedia.org/wiki/Mort_%26_Phil)
