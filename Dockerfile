ARG ARCH=amd64

FROM golang:1.10.0 AS builder-amd64

FROM arm32v6/golang:1.10.0 AS builder-arm32v6

FROM builder-${ARCH} AS builder

WORKDIR ${GOPATH}/src/github.com/mcuadros/ofelia
COPY . ${GOPATH}/src/github.com/mcuadros/ofelia

ENV CGO_ENABLED 0
ENV GOOS linux

RUN go get -v ./...
RUN go build -a -installsuffix cgo -ldflags '-w  -extldflags "-static"' -o /go/bin/ofelia .

FROM scratch

COPY --from=builder /go/bin/ofelia /usr/bin/ofelia

VOLUME /etc/ofelia/
ENTRYPOINT ["/usr/bin/ofelia"]

CMD ["daemon", "--config", "/etc/ofelia/config.ini"]
