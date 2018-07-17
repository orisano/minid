# minid
minid is Dockerfile minifier.

## Features
* concatenate RUN command
* concatenate ENV command
* ...

## Installation
```bash
go get -u github.com/orisano/minid
```

## How to use
```bash
$ cat Dockerfile
FROM golang:1.10-alpine3.8 AS build

ENV DEP_VERSION 0.4.1

WORKDIR /go/src/github.com/orisano/gobase

RUN apk add --no-cache git make mailcap tzdata
RUN wget -O /usr/local/bin/dep https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 && chmod +x /usr/local/bin/dep
RUN wget -O /usr/local/bin/depinst https://github.com/orisano/depinst/releases/download/1.0.1/depinst-linux-amd64 && chmod +x /usr/local/bin/depinst

COPY Gopkg.lock Gopkg.toml ./
RUN dep ensure -vendor-only

ENV CGO_ENABLED=0
ENV GO_LDFLAGS="-extldflags='-static'"
RUN go build -i ./vendor/...

COPY . .
RUN make build
```
```bash
$ minid
FROM golang:1.10-alpine3.8 AS build
ENV DEP_VERSION=0.4.1
WORKDIR /go/src/github.com/orisano/gobase
RUN apk add --no-cache git make mailcap tzdata && wget -O /usr/local/bin/dep https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 && chmod +x /usr/local/bin/dep && wget -O /usr/local/bin/depinst https://github.com/orisano/depinst/releases/download/1.0.1/depinst-linux-amd64 && chmod +x /usr/local/bin/depinst
COPY Gopkg.lock Gopkg.toml ./
RUN dep ensure -vendor-only
ENV CGO_ENABLED=0 GO_LDFLAGS="-extldflags='-static'"
RUN go build -i ./vendor/...
COPY . .
RUN make build
```
```bash
$ minid | docker build -f - .
```

## Author
Nao YONASHIRO (@orisano)

## License
MIT
