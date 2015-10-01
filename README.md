# Ofelia - a job scheduler [![Latest Stable Version](http://img.shields.io/github/release/mcuadros/ofelia.svg?style=flat)](https://github.com/mcuadros/ofelia/releases) [![codecov.io](http://codecov.io/github/mcuadros/ofelia/coverage.svg?branch=master)](http://codecov.io/github/mcuadros/ofelia?branch=master) [![Build Status](http://img.shields.io/travis/mcuadros/ofelia.svg?style=flat)](https://travis-ci.org/mcuadros/ofelia)

<img src="https://weirdspace.dk/FranciscoIbanez/Graphics/Ofelia.gif" align="right" width="180px" height="300px" vspace="20" />


**Ofelia** is a moderm and low footprint job scheduler for __docker__ environments, build on Go. Ofelia aims to be a replacement for the old fashioned [cron](https://en.wikipedia.org/wiki/Cron)

### Why?

It has been a long time since [`cron`](https://en.wikipedia.org/wiki/Cron) was released, actually more than 28 years. The world has changed a lot and especially since the `Docker` revolution. **Vixie's cron** works great but not is extensible and is hard to debug when something is going wrong.

Many solutions are available: ready to go containerized `crons`, wrappers for your commands, etc. but at the end simple tasks become complex.   

### How?

The main feature of **Ofelia** is the ability to execute commands directly on Docker containers. Using the Docker's API Ofelia emulates the behavior of [`exec`](https://docs.docker.com/reference/commandline/exec/) having full control of the process runing inside of the container.


## Configuration

The configuration is done using a INI-style config file and the [sceduling format](https://godoc.org/github.com/robfig/cron) is exactly the same from the original `cron`:

```ini
[job "your-cron-name"]
schedule = @hourly
container = my-container
command = touch /tmp/example
```



> Why the project is named Ofelia? Ofelia is the name of the office assistant from the Spanish comic [Mortadelo y Filemón](https://en.wikipedia.org/wiki/Mort_%26_Phil)
