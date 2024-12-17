# build web
FROM node:20.17.0-alpine3.20 AS web-builder
RUN corepack enable

WORKDIR /web

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web ./
RUN pnpm run build

# build app
FROM golang:1.23-alpine3.20 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME

RUN apk add --no-cache git build-base tzdata

ENV SERVICE=autobrr

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=web-builder /web/dist ./web/dist
COPY --from=web-builder /web/build.go ./web

#ENV GOOS=linux
#ENV CGO_ENABLED=0

RUN go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/autobrr cmd/autobrr/main.go && \
    go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o bin/autobrrctl cmd/autobrrctl/main.go

# build runner
FROM alpine:latest

LABEL org.opencontainers.image.source="https://github.com/autobrr/autobrr"

ENV HOME="/config" \
    XDG_CONFIG_HOME="/config" \
    XDG_DATA_HOME="/config"

RUN apk --no-cache add ca-certificates curl tzdata jq

WORKDIR /app

VOLUME /config

COPY --from=app-builder /src/bin/autobrr /usr/local/bin/
COPY --from=app-builder /src/bin/autobrrctl /usr/local/bin/

EXPOSE 7474

ENTRYPOINT ["/usr/local/bin/autobrr", "--config", "/config"]
