# Jobs reference

- [`exec`](#exec)
- [`run`](#run)
- [`local`](#local)
- [`service-run`](#service-run)

## `exec`

This job is executed inside a running container, similar to `docker exec`.

### Parameters

- **`schedule`: string**
  - When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
- **`command`: string**
  - Command you want to run inside the container.
- **`container`: string**
  - Name of the container you want to execute the command in.
- `user`: string = `root`
  - User as which the command should be executed, similar to `docker exec --user <user>`
- `tty`: boolean = `false`
  - Allocate a pseudo-tty, similar to `docker exec -t`. See this [Stack Overflow answer](https://stackoverflow.com/questions/30137135/confused-about-docker-t-option-to-allocate-a-pseudo-tty) for more info.
- `environment`
  - Environment variables you want to set in the running container. **Note:** only supported in Docker API v1.25 and above
  - Same format as used with `-e` flag within `docker run`. For example: `FOO=bar`
    - **INI config**: `Environment` setting can be provided multiple times for multiple environment variables.
    - **Labels config**: multiple environment variables has to be provided as JSON array: `["FOO=bar", "BAZ=qux"]`
- `no-overlap`: boolean = `false`
  - Prevent that the job runs concurrently

### INI-file example

```ini
[job-exec "flush-nginx-logs"]
schedule = @hourly
container = nginx-proxy
command = /bin/bash /flush-logs.sh
user = www-data
tty = false
```

### Docker labels example

```sh
docker run -it --rm \
    --label ofelia.enabled=true \
    --label ofelia.job-exec.flush-nginx-logs.schedule="@hourly" \
    --label ofelia.job-exec.flush-nginx-logs.command="/bin/bash /flush-logs.sh" \
    --label ofelia.job-exec.flush-nginx-logs.user="www-data" \
    --label ofelia.job-exec.flush-nginx-logs.tty="false" \
        nginx
```

## `run`

This job can be used in 2 situations:

1. To run a command inside of a new container, using a specific image, similar to `docker run`
1. To start a stopped container, similar to `docker start`

### Parameters

- **`schedule`: string** (1, 2)
  - When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
- `command`: string = default container command (1)
  - Command you want to run inside the container.
- **`image`: string** (1)
  - Image you want to use for the job.
  - If left blank, Ofelia assumes you will specify a container to start (situation 2).
- `user`: string = `root` (1)
  - User as which the command should be executed, similar to `docker run --user <user>`
- `network`: string (1)
  - Connect the container to this network
- `delete`: boolean = `true` (1)
  - Delete the container after the job is finished. Similar to `docker run --rm`
- **`container`: string** (2)
  - Name of the container you want to start.
  - Required field in case parameter `image` is not specified, no default.
- `tty`: boolean = `false` (1, 2)
  - Allocate a pseudo-tty, similar to `docker exec -t`. See this [Stack Overflow answer](https://stackoverflow.com/questions/30137135/confused-about-docker-t-option-to-allocate-a-pseudo-tty) for more info.
- `volume`:
  - Mount host machine directory into container as a [bind mount](https://docs.docker.com/storage/bind-mounts/#start-a-container-with-a-bind-mount)
  - Same format as used with `-v` flag within `docker run`. For example: `/tmp/test:/tmp/test:ro`
    - **INI config**: `Volume` setting can be provided multiple times for multiple mounts.
    - **Labels config**: multiple mounts has to be provided as JSON array: `["/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"]`
- `environment`
  - Environment variables you want to set in the running container.
  - Same format as used with `-e` flag within `docker run`. For example: `FOO=bar`
    - **INI config**: `Environment` setting can be provided multiple times for multiple environment variables.
    - **Labels config**: multiple environment variables has to be provided as JSON array: `["FOO=bar", "BAZ=qux"]`
- `no-overlap`: boolean = `false`
  - Prevent that the job runs concurrently

### INI-file example

```ini
[job-run "print-write-date"]
schedule = @every 5s
image = alpine:latest
command = sh -c 'date | tee -a /tmp/test/date'
volume = /tmp/test:/tmp/test:rw
environment = FOO=bar
```

Then you can check output in host machine file `/tmp/test/date`

### Running Ofelia in Docker

```sh
docker run -it --rm \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    --label ofelia.enabled=true \
    --label ofelia.job-run.print-write-date.schedule="@every 5s" \
    --label ofelia.job-run.print-write-date.image="alpine:latest" \
    --label ofelia.job-run.print-write-date.volume="/tmp/test:/tmp/test:rw" \
    --label ofelia.job-run.print-write-date.environment="FOO=bar" \
    --label ofelia.job-run.print-write-date.command="sh -c 'date | tee -a /tmp/test/date'" \
        mcuadros/ofelia:latest daemon
```

## `local`

Runs the command on the host running Ofelia.

**Note**: In case Ofelia is running inside a container, the command is executed inside the container. Not on the Docker host.

### Parameters

- **`schedule`: string**
  - When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
- **`command`: string**
  - Command you want to run on the host.
- `dir`: string = `$(pwd)`
  - Base directory to execute the command.
- `environment`
  - Environment variables you want to set for the executed command.
  - Same format as used with `-e` flag within `docker run`. For example: `FOO=bar`
    - **INI config**: `Environment` setting can be provided multiple times for multiple environment variables.
    - **Labels config**: multiple environment variables has to be provided as JSON array: `["FOO=bar", "BAZ=qux"]`
- `no-overlap`: boolean = `false`
  - Prevent that the job runs concurrently

### INI-file example

```ini
[job-local "create-file"]
schedule = @every 15s
command = touch test.txt
dir = /tmp/
```

## `service-run`

This job can be used to:

- To run a command inside a new `run-once` service, for running inside a swarm.

### Parameters

- **`schedule`: string** (1, 2)
  - When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
- `command`: string = default container command (1, 2)
  - Command you want to run inside the container.
- **`image`: string** (1)
  - Image you want to use for the job.
  - If left blank, Ofelia assumes you will specify a container to start (situation 2).
- `network`: string (1)
  - Connect the container to this network
- `delete`: boolean = `true` (1)
  - Delete the container after the job is finished.
- `user`: string = `root` (1, 2)
  - User as which the command should be executed.
- `tty`: boolean = `false` (1, 2)
  - Allocate a pseudo-tty, similar to `docker exec -t`. See this [Stack Overflow answer](https://stackoverflow.com/questions/30137135/confused-about-docker-t-option-to-allocate-a-pseudo-tty) for more info.
- `no-overlap`: boolean = `false`
  - Prevent that the job runs concurrently

### INI-file example

```ini
[job-service-run "service-executed-on-new-container"]
schedule = 0,20,40 * * * *
image = ubuntu
network = swarm_network
command =  touch /tmp/example
```
