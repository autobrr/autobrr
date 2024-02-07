# build app
FROM --platform=$BUILDPLATFORM golang:1.20-alpine3.19 AS app-base
RUN apk add --no-cache git tzdata
ENV SERVICE=autobrr
WORKDIR /src

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS TARGETARCH TARGETVARIANT

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

FROM --platform=$BUILDPLATFORM app-base AS autobrr
RUN --network=none --mount=target=. \
export GOOS=$TARGETOS; \
export GOARCH=$TARGETARCH; \
[[ "$GOARCH" == "amd64" ]] && export GOAMD64=$TARGETVARIANT; \
[[ "$GOARCH" == "arm" ]] && [[ "$TARGETVARIANT" == "v6" ]] && export GOARM=6; \
[[ "$GOARCH" == "arm" ]] && [[ "$TARGETVARIANT" == "v7" ]] && export GOARM=7; \
echo $GOARCH $GOOS $GOARM$GOAMD64; \
go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrr cmd/autobrr/main.go && \
go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrrctl cmd/autobrrctl/main.go

# build runner
FROM alpine:latest AS runner

LABEL   org.opencontainers.image.source = "https://github.com/autobrr/autobrr" \
        org.opencontainers.image.licenses = "GPL-2.0-or-later" \
        org.opencontainers.image.base.name = "alpine:latest"

ENV HOME="/config" \
    XDG_CONFIG_HOME="/config" \
    XDG_DATA_HOME="/config"

WORKDIR /app
VOLUME /config
EXPOSE 7474
ENTRYPOINT ["/usr/local/bin/autobrr", "--config", "/config"]

RUN apk --no-cache add ca-certificates curl tzdata jq

COPY --link --from=autobrr /out/bin/autobrr /usr/local/bin/
COPY --link --from=autobrr /out/bin/autobrrctl /usr/local/bin/
