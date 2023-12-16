FROM golang:1.20-alpine3.18 AS app-pull
WORKDIR /src
RUN --mount=target=. \
go mod download -x

FROM scratch AS app-cache
COPY --link --from=app-pull /go/pkg /go/pkg

# build app
FROM --platform=$BUILDPLATFORM golang:1.20-alpine3.18 AS app-builder
WORKDIR /src
RUN apk add --no-cache git tzdata

ENV SERVICE=autobrr

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS TARGETARCH

RUN --mount=target=. --mount=from=app-cache,target=/go/pkg/mod,source=/go/pkg/mod \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrr cmd/autobrr/main.go && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrrctl cmd/autobrrctl/main.go

# build runner
FROM alpine:latest

LABEL org.opencontainers.image.source = "https://github.com/autobrr/autobrr"

ENV HOME="/config" \
    XDG_CONFIG_HOME="/config" \
    XDG_DATA_HOME="/config"

RUN apk --no-cache add ca-certificates curl tzdata jq

WORKDIR /app
VOLUME /config
EXPOSE 7474

COPY --from=app-builder /out/bin/autobrr /usr/local/bin/
COPY --from=app-builder /out/bin/autobrrctl /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/autobrr", "--config", "/config"]

