# build app
FROM --platform=$BUILDPLATFORM golang:1.20-alpine3.18 AS app-builder
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
ARG TARGETOS TARGETARCH TARGETVARIANT

RUN --mount=target=. \
export GOOS=$TARGETOS; \
export GOARCH=$TARGETARCH; \
[[ $GOARCH -eq "amd64"]] && export GOAMD64=$TARGETVARIANT; \
[[ $GOARCH -eq "arm"]] && [[ $TARGETVARIANT -eq "v6" ]] && export GOARM=6; \
[[ $GOARCH -eq "arm"]] && [[ $TARGETVARIANT -eq "v7" ]] && export GOARM=7; \
go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrr cmd/autobrr/main.go && \
go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" -o /out/bin/autobrrctl cmd/autobrrctl/main.go

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
