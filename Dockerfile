FROM golang:1.13.3-alpine AS builder

WORKDIR ${GOPATH}/src/github.com/mcuadros/ofelia
COPY . ${GOPATH}/src/github.com/mcuadros/ofelia

RUN apk --no-cache add gcc musl-dev
RUN go build -o /go/bin/ofelia .

FROM alpine:latest

# this label is required to identify container with ofelia running
LABEL ofelia.service=true
LABEL ofelia.enabled=true

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /go/bin/ofelia /usr/bin/ofelia

ENTRYPOINT ["/usr/bin/ofelia"]

CMD ["daemon", "--config", "/etc/ofelia/config.ini"]
