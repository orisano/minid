# minid

[![Build Status](https://github.com/orisano/minid/workflows/test/badge.svg)](https://github.com/orisano/minid/actions?query=workflow%3Atest)

minid is Dockerfile minifier for reducing the number of layers.

## Features
* concatenate RUN command
* concatenate ENV command
* concatenate LABEL command
* concatenate COPY, ADD command
* ...

## Installation
```bash
go get -u github.com/orisano/minid
```

## How to use
```bash
$ cat Dockerfile # 8 layers
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
$ minid # 6 layers
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
