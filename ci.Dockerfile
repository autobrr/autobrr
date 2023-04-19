# build app
FROM --platform=$BUILDPLATFORM golang:1.20-alpine3.16 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS TARGETARCH GOMODCACHE GOCACHE

RUN apk add --no-cache git make build-base tzdata

ENV SERVICE=autobrr
ENV GOMODCACHE=$GOMODCACHE
ENV GOCACHE=$GOCACHE

WORKDIR /src
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    --mount=type=bind,rw,source=$GOMODCACHE,target=$GOMODCACHE \
    --mount=type=bind,rw,source=$GOCACHE,target=$GOCACHE \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrr cmd/autobrr/main.go
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    --mount=type=bind,rw,source=$GOMODCACHE,target=$GOMODCACHE \
    --mount=type=bind,rw,source=$GOCACHE,target=$GOCACHE \
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

COPY --from=app-builder /out/bin/autobrr /usr/local/bin/
COPY --from=app-builder /out/bin/autobrrctl /usr/local/bin/

EXPOSE 7474

ENTRYPOINT ["/usr/local/bin/autobrr", "--config", "/config"]
#CMD ["--config", "/config"]
