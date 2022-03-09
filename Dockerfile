# build web
FROM node:16-alpine AS web-builder
WORKDIR /web
COPY web/package.json ./
RUN yarn install
COPY web .
RUN yarn build

# build app
FROM golang:1.17.6-alpine AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME

RUN apk add --no-cache git make build-base

ENV SERVICE=autobrr

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

COPY --from=web-builder /web/build ./web/build
COPY --from=web-builder /web/build.go ./web

ENV GOOS=linux
ENV CGO_ENABLED=1

RUN go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/autobrr cmd/autobrr/main.go
RUN go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/autobrrctl cmd/autobrrctl/main.go

# build runner
FROM alpine:latest

LABEL org.opencontainers.image.source = "https://github.com/autobrr/autobrr"

ENV HOME="/config" \
XDG_CONFIG_HOME="/config" \
XDG_DATA_HOME="/config"

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

VOLUME /config

COPY --from=app-builder /src/bin/autobrr /usr/local/bin/
COPY --from=app-builder /src/bin/autobrrctl /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/autobrr", "--config", "/config"]
#CMD ["--config", "/config"]
