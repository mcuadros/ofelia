# Jobs reference

- [job-exec](#job-exec)
- [job-run](#job-run)
- [job-local](#job-local)
- [job-service-run](#job-service-run)


>[!IMPORTANT]
>Configuration keys are not case sensitive

## Job-exec

This job is executed inside a running container. Similar to `docker exec`

### Parameters

- **Schedule** *
  - *description*: When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
  - *value*: String, see [Scheduling format](https://godoc.org/github.com/robfig/cron) of the Go implementation of `cron`. E.g. `@every 10s` or `0 0 1 * * *` (every night at 1 AM). **Note**: the format starts with seconds, instead of minutes.
  - *default*: Required field, no default.
- **Command** *
  - *description*: Command you want to run inside the container.
  - *value*: String, e.g. `touch /tmp/example`
  - *default*: Required field, no default.
- **Container** *
  - *description*: Name of the container you want to execute the command in.
  - *value*: String, e.g. `nginx-proxy`
  - *default*: Required field, no default.
- **User**
  - *description*: User as which the command should be executed, similar to `docker exec --user <user>`
  - *value*: String, e.g. `www-data`
  - *default*: `root`
- **tty**
  - *description*: Allocate a pseudo-tty, similar to `docker exec -t`. See this [Stack Overflow answer](https://stackoverflow.com/questions/30137135/confused-about-docker-t-option-to-allocate-a-pseudo-tty) for more info.
  - *value*: Boolean, either `false` or `true`
  - *default*: `false`
- **Environment**
  - *description*: Environment variables you want to set in the running container. Note: ***exec configuration Env is only supported in Docker API#1.25 and above***
  - *value*: Same format as used with `-e` flag within `docker run`. For example: `FOO=bar`
    - **INI config**: `Environment` setting can be provided multiple times for multiple environment variables.
    - **Labels config**: multiple environment variables has to be provided as JSON array: `["FOO=bar", "BAZ=qux"]`
  - *default*: Optional field, no default.
  
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

`ofelia` container should be started **after** nginx container, to be able to read its labels, because real time labels reading is not supported yet.

```sh
docker run -it --rm \
    --label ofelia.enabled=true \
    --label ofelia.job-exec.flush-nginx-logs.schedule="@hourly" \
    --label ofelia.job-exec.flush-nginx-logs.command="/bin/bash /flush-logs.sh" \
    --label ofelia.job-exec.flush-nginx-logs.user="www-data" \
    --label ofelia.job-exec.flush-nginx-logs.tty="false" \
        nginx
```

## Job-run

This job can be used in 2 situations:

1. To run a command inside of a new container, using a specific image. Similar to `docker run`
1. To start a stopped container, similar to `docker start`

### Parameters

- **Schedule** * (1,2)
  - *description*: When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
  - *value*: String, see [Scheduling format](https://godoc.org/github.com/robfig/cron) of the Go implementation of `cron`. E.g. `@every 10s` or `0 0 1 * * *` (every night at 1 AM). **Note**: the format starts with seconds, instead of minutes.
  - *default*: Required field, no default.
- **Entrypoint** (1)
  - *description*: Override container default entrypoint.
  - *value*: String, e.g. `/bin/bash -c`
  - *default*: Default container entrypoint
- **Command** (1)
  - *description*: Command you want to run inside the container.
  - *value*: String, e.g. `touch /tmp/example`
  - *default*: Default container command
- **Image** (1)
  - *description*: Image you want to use for the job.
  - *value*: String, e.g. `nginx:latest`
  - *default*: No default. If left blank, Ofelia assumes you will specify a container to start (situation 2).
- **User** (1)
  - *description*: User as which the command should be executed, similar to `docker run --user <user>`
  - *value*: String, e.g. `www-data`
  - *default*: `root`
- **Network** (1)
  - *description*: Connect the container to this network
  - *value*: String, e.g. `backend-proxy`
  - *default*: Optional field, no default.
- **Hostname** (1)
  - *description*: Define the hostname of the instantiated container
  - *value*: String, e.g. `test-server`
  - *default*: Optional field, no default.
- **Delete** (1)
  - *description*: Delete the container after the job is finished. Similar to `docker run --rm`
  - *value*: Boolean, either `true` or `false`
  - *default*: `true`
- **Container** (2)
  - *description*: Name of the container you want to start.
  - *value*: String, e.g. `nginx-proxy`
  - *default*: Required field in case parameter `image` is not specified, no default.
- **tty** (1,2)
  - *description*: Allocate a pseudo-tty, similar to `docker exec -t`. See this [Stack Overflow answer](https://stackoverflow.com/questions/30137135/confused-about-docker-t-option-to-allocate-a-pseudo-tty) for more info.
  - *value*: Boolean, either `true` or `false`
  - *default*: `false`
- **Volume**
  - *description*: Mount host machine directory into container as a [bind mount](https://docs.docker.com/storage/bind-mounts/#start-a-container-with-a-bind-mount)
  - *value*: Same format as used with `-v` flag within `docker run`. For example: `/tmp/test:/tmp/test:ro`
    - **INI config**: `Volume` setting can be provided multiple times for multiple mounts.
    - **Labels config**: multiple mounts has to be provided as JSON array: `["/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"]`
  - *default*: Optional field, no default.
- **Volumes-From**
  - *description*: Use the volumes from another container.
  - *value*: The name of the container, from which the volumes will be used.
    - **INI config**: setting can be provided multiple times for multiple mounts.
    - **Labels config**: multiple mounts has to be provided as JSON array: `["container-foo", "bar-container"]`
  - *default*: Optional field, no default.
- **Environment**
  - *description*: Environment variables you want to set in the running container.
  - *value*: Same format as used with `-e` flag within `docker run`. For example: `FOO=bar`
    - **INI config**: setting can be provided multiple times for multiple environment variables.
    - **Labels config**: multiple environment variables has to be provided as JSON array: `["FOO=bar", "BAZ=qux"]`
  - *default*: Optional field, no default.
  
### INI-file example

```ini
[job-run "print-write-date"]
schedule = @every 5s
image = alpine:latest
command = sh -c 'date | tee -a /tmp/test/date'
volume = /tmp/test:/tmp/test:rw
environment = FOO=bar
environment = BAZ=qux
```

Then you can check output in host machine file `/tmp/test/date`

### Docker labels example

Docker run job has to be configured as labels on the `ofelia` container itself, because it is going to start new container:

```sh
docker run -it --rm \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    --label ofelia.enabled=true \
    --label ofelia.job-run.print-write-date.schedule="@every 5s" \
    --label ofelia.job-run.print-write-date.image="alpine:latest" \
    --label ofelia.job-run.print-write-date.volume="/tmp/test:/tmp/test:rw" \
    --label ofelia.job-run.print-write-date.environment="FOO=bar" \
    --label ofelia.job-run.print-write-date.command="sh -c 'date | tee -a /tmp/test/date'" \
        mcuadros/ofelia:latest daemon --docker
```

## Job-local

Runs the command on the host running Ofelia.

**Note**: In case Ofelia is running inside a container, the command is executed inside the container. Not on the Docker host.

### Parameters

- **Schedule** *
  - *description*: When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
  - *value*: String, see [Scheduling format](https://godoc.org/github.com/robfig/cron) of the Go implementation of `cron`. E.g. `@every 10s` or `0 0 1 * * *` (every night at 1 AM). **Note**: the format starts with seconds, instead of minutes.
  - *default*: Required field, no default.
- **Command** *
  - *description*: Command you want to run on the host.
  - *value*: String, e.g. `touch test.txt`
  - *default*: Required field, no default.
- **Dir**
  - *description*: Base directory to execute the command.
  - *value*: String, e.g. `/tmp/sandbox/`
  - *default*: Current directory
- **Environment**
  - *description*: Environment variables you want to set for the executed command.
  - *value*: Same format as used with `-e` flag within `docker run`. For example: `FOO=bar`
    - **INI config**: `Environment` setting can be provided multiple times for multiple environment variables.
    - **Labels config**: multiple environment variables has to be provided as JSON array: `["FOO=bar", "BAZ=qux"]`
  - *default*: Optional field, no default.

### INI-file example

```ini
[job-local "create-file"]
schedule = @every 15s
command = touch test.txt
dir = /tmp/
```

### Docker labels example

Docker run job has to be configured as labels on the `ofelia` container itself, because it will be executed inside `ofelia` container

```sh
docker run -it --rm \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    --label ofelia.enabled=true \
    --label ofelia.job-local.create-file.schedule="@every 15s" \
    --label ofelia.job-local.create-file.image="alpine:latest" \
    --label ofelia.job-local.create-file.command="touch test.txt" \
    --label ofelia.job-local.create-file.dir="/tmp/" \
        mcuadros/ofelia:latest daemon --docker
```

## Job-service-run

This job can be used to:

- To run a command inside a new "run-once" service, for running inside a swarm.

### Parameters

- **Schedule** * (1,2)
  - *description*: When the job should be executed. E.g. every 10 seconds or every night at 1 AM.
  - *value*: String, see [Scheduling format](https://godoc.org/github.com/robfig/cron) of the Go implementation of `cron`. E.g. `@every 10s` or `0 0 1 * * *` (every night at 1 AM). **Note**: the format starts with seconds, instead of minutes.
  - *default*: Required field, no default.
- **Command** (1, 2)
  - *description*: Command you want to run inside the container.
  - *value*: String, e.g. `touch /tmp/example`
  - *default*: Default container command
- **Image** * (1)
  - *description*: Image you want to use for the job.
  - *value*: String, e.g. `nginx:latest`
  - *default*: No default. If left blank, Ofelia assumes you will specify a container to start (situation 2).
- **Network** (1)
  - *description*: Connect the container to this network
  - *value*: String, e.g. `backend-proxy`
  - *default*: Optional field, no default.
- **delete** (1)
  - *description*: Delete the container after the job is finished.
  - *value*: Boolean, either `true` or `false`
  - *default*: `true`
- **User** (1,2)
  - *description*: User as which the command should be executed.
  - *value*: String, e.g. `www-data`
  - *default*: `root`
- **tty** (1,2)
  - *description*: Allocate a pseudo-tty, similar to `docker exec -t`. See this [Stack Overflow answer](https://stackoverflow.com/questions/30137135/confused-about-docker-t-option-to-allocate-a-pseudo-tty) for more info.
  - *value*: Boolean, either `true` or `false`
  - *default*: `false`
  
### INI-file example

```ini
[job-service-run "service-executed-on-new-container"]
schedule = 0,20,40 * * * *
image = ubuntu
network = swarm_network
command =  touch /tmp/example
```
