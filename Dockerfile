FROM golang:1.5.2-alpine
MAINTAINER Máximo Cuadros <mcuadros@gmail.com>

ADD . ${GOPATH}/src/github.com/mcuadros/ofelia
WORKDIR ${GOPATH}/src/github.com/mcuadros/ofelia

RUN apk -U add git \
  && rm -rf /var/cache/apk/* \
  && go get -v ./... \
  && go install -v ./... \
  && rm -rf $GOPATH/src/

VOLUME /etc/ofelia/
CMD ["ofelia", "daemon", "--config", "/etc/ofelia/config.ini"]
