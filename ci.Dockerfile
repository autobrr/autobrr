# build app
FROM --platform=$BUILDPLATFORM golang:1.25-alpine3.22 AS app-builder
RUN apk add --no-cache git tzdata

ENV SERVICE=autobrr

WORKDIR /src

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

RUN --network=none --mount=target=. \
export GOOS=$TARGETOS; \
export GOARCH=$TARGETARCH; \
[ "$GOARCH" = "amd64" ] && export GOAMD64="$TARGETVARIANT"; \
[ "$GOARCH" = "arm64" ] && case "$TARGETVARIANT" in *.*) export GOARM64="$TARGETVARIANT";; esac; \
[ "$GOARCH" = "arm64" ] && case "$TARGETVARIANT" in *.*) : ;; v*) export GOARM64="$TARGETVARIANT.0";; esac; \
[ "$GOARCH" = "arm" ] && [ "$TARGETVARIANT" = "v6" ] && export GOARM=6; \
[ "$GOARCH" = "arm" ] && [ "$TARGETVARIANT" = "v7" ] && export GOARM=7; \
echo $GOARCH $GOOS $GOARM$GOAMD64$GOARM64; \
go build -pgo=cpu.pprof -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrr cmd/autobrr/main.go && \
go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrrctl cmd/autobrrctl/main.go

# build runner
FROM alpine:latest AS runner

LABEL org.opencontainers.image.source="https://github.com/autobrr/autobrr"
LABEL org.opencontainers.image.licenses="GPL-2.0-or-later"
LABEL org.opencontainers.image.base.name="alpine:latest"

ENV HOME="/config" \
    XDG_CONFIG_HOME="/config" \
    XDG_DATA_HOME="/config"

RUN apk --no-cache add ca-certificates curl tzdata jq

WORKDIR /app
VOLUME /config
EXPOSE 7474

COPY --link --from=app-builder /out/bin/autobrr* /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/autobrr", "--config", "/config"]
