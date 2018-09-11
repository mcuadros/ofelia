FROM golang:1.10.0 AS builder

WORKDIR ${GOPATH}/src/github.com/mcuadros/ofelia
COPY . ${GOPATH}/src/github.com/mcuadros/ofelia

ENV CGO_ENABLED 0
ENV GOOS linux

RUN go get -v ./...
RUN go build -a -installsuffix cgo -ldflags '-w  -extldflags "-static"' -o /go/bin/ofelia .

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /go/bin/ofelia /usr/bin/ofelia

VOLUME /etc/ofelia/
ENTRYPOINT ["/usr/bin/ofelia"]

CMD ["daemon", "--config", "/etc/ofelia/config.ini"]