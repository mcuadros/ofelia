# Ofelia - a job scheduler [![GitHub version](https://badge.fury.io/gh/netresearch%2Fofelia.svg)](https://github.com/netresearch/ofelia/releases) [![go test](https://github.com/netresearch/ofelia/actions/workflows/test.yml/badge.svg)](https://github.com/netresearch/ofelia/actions/workflows/test.yml)

<img src="https://weirdspace.dk/FranciscoIbanez/Graphics/Ofelia.gif" align="right" width="180px" height="300px" vspace="20" />

**Ofelia** is a modern and low footprint job scheduler for **Docker** environments, built on Go. Ofelia aims to be a replacement for the old fashioned [cron](https://en.wikipedia.org/wiki/Cron).

This fork is based off of [mcuadros/ofelia](https://github.com/mcuadros/ofelia).

## Using it

### Docker

The easiest way to deploy **Ofelia** is using **Docker**.

    docker pull ghcr.io/netresearch/ofelia

### Standalone

If don't want to run **Ofelia** using our **Docker** image, you can download a binary from [our releases page](https://github.com/netresearch/ofelia/releases).

    wget https://github.com/netresearch/ofelia/releases/latest

## Configuration

### Jobs

#### Scheduling format

This application uses the [Go implementation of `cron`](https://pkg.go.dev/github.com/robfig/cron) and uses a parser for supporting optional seconds.

Supported formats:

- `@every 10s`
- `20 0 1 * * *` (every night, 20 seconds after 1 AM - [Quartz format](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/tutorial-lesson-06.html)
- `0 1 * * *` (every night at 1 AM - standard [cron format](https://en.wikipedia.org/wiki/Cron)).

You can configure four different kind of jobs:

- `job-exec`: this job is executed inside of a running container.
- `job-run`: runs a command inside of a new container, using a specific image.
- `job-local`: runs the command inside of the host running ofelia.
- `job-service-run`: runs the command inside a new "run-once" service, for running inside a swarm

See [Jobs reference documentation](docs/jobs.md) for all available parameters.

### Logging

**Ofelia** comes with three different logging drivers that can be configured in the `[global]` section or as top-level Docker labels:

- `mail` to send mails
- `save` to save structured execution reports to a directory
- `slack` to send messages via a slack webhook

### Global Options

- `smtp-host` - address of the SMTP server.
- `smtp-port` - port number of the SMTP server.
- `smtp-user` - user name used to connect to the SMTP server.
- `smtp-password` - password used to connect to the SMTP server.
- `smtp-tls-skip-verify` - when `true` ignores certificate signed by unknown authority error.
- `email-to` - mail address of the receiver of the mail.
- `email-from` - mail address of the sender of the mail.
- `mail-only-on-error` - only send a mail if the execution was not successful.

- `save-folder` - directory in which the reports shall be written.
- `save-only-on-error` - only save a report if the execution was not successful.

- `slack-webhook` - URL of the slack webhook.
- `slack-only-on-error` - only send a slack message if the execution was not successful.

### INI-style configuration

Run with `ofelia daemon --config=/path/to/config.ini`

```ini
[global]
save-folder = /var/log/ofelia_reports
save-only-on-error = true

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

### Docker label configurations

In order to use this type of configuration, Ofelia needs access to Docker socket.

> ⚠ **Warning**: This command changed! Please remove the `--docker` flag from your command.

```sh
docker run -it --rm \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    --label ofelia.save-folder="/var/log/ofelia_reports" \
    --label ofelia.save-only-on-error="true" \
        ghcr.io/netresearch/ofelia:latest daemon
```

Labels format: `ofelia.<JOB_TYPE>.<JOB_NAME>.<JOB_PARAMETER>=<PARAMETER_VALUE>`.
This type of configuration supports all the capabilities provided by INI files, including the global logging options.

Also, it is possible to configure `job-exec` by setting labels configurations on the target container. To do that, additional label `ofelia.enabled=true` need to be present on the target container.

For example, we want `ofelia` to execute `uname -a` command in the existing container called `nginx`.
To do that, we need to we need to start the `nginx` container with next configurations:

```sh
docker run -it --rm \
    --label ofelia.enabled=true \
    --label ofelia.job-exec.test-exec-job.schedule="@every 5s" \
    --label ofelia.job-exec.test-exec-job.command="uname -a" \
        nginx
```

**Ofelia** reads labels of all Docker containers for configuration by default. To apply on a subset of containers only, use the flag `--docker-filter` (or `-f`) similar to the [filtering for `docker ps`](https://docs.docker.com/engine/reference/commandline/ps/#filter). E.g. to apply to current docker compose project only using `label` filter:

```yaml
version: "3"
services:
  ofelia:
    image: ghcr.io/netresearch/ofelia:latest
    depends_on:
      - nginx
    command: daemon --docker -f label=com.docker.compose.project=${COMPOSE_PROJECT_NAME}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    labels:
      ofelia.job-local.my-test-job.schedule: "@every 5s"
      ofelia.job-local.my-test-job.command: "date"
  nginx:
    image: nginx
    labels:
      ofelia.enabled: "true"
      ofelia.job-exec.datecron.schedule: "@every 5s"
      ofelia.job-exec.datecron.command: "uname -a"
```

### Dynamic Docker configuration

You can start Ofelia in its own container or on the host itself, and it will magically pick up any container that starts, stops or is modified on the fly.
In order to achieve this, you simply have to use Docker containers with the labels described above and let Ofelia take care of the rest.

### Hybrid configuration (INI files + Docker)

You can specify part of the configuration on the INI files, such as globals for the middlewares or even declare tasks in there but also merge them with Docker.
The Docker labels will be parsed, added and removed on the fly but also, the file config can be used.

Use the INI file to:

- Configure any middleware
- Configure any global setting
- Create a `run` jobs, so they executes in a new container each time

```ini
[global]
slack-webhook = https://myhook.com/auth

[job-run "job-executed-on-new-container"]
schedule = @hourly
image = ubuntu:latest
command = touch /tmp/example
```

Use docker to:

- Create `exec` jobs

```sh
docker run -it --rm \
    --label ofelia.enabled=true \
    --label ofelia.job-exec.test-exec-job.schedule="@every 5s" \
    --label ofelia.job-exec.test-exec-job.command="uname -a" \
        nginx
```
