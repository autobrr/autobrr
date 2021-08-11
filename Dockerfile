# build web
FROM node:16-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY web .
RUN yarn build

# build app
FROM golang:1.16 AS app-builder

ENV SERVICE=autobrr

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

COPY --from=web-builder /web/build ./web/build
COPY --from=web-builder /web/build.go ./web

ENV CGO_ENABLED=0
ENV GOOS=linux

#RUN make -f Makefile build/app
RUN go build -o bin/${SERVICE} ./cmd/${SERVICE}/main.go

# build runner
FROM alpine:latest

ENV HOME="/config" \
XDG_CONFIG_HOME="/config" \
XDG_DATA_HOME="/config"

RUN apk --no-cache add ca-certificates

WORKDIR /app

VOLUME /config

COPY --from=app-builder /src/bin/autobrr /usr/local/bin/

ENTRYPOINT ["autobrr", "--config", "/config"]
#CMD ["--config", "/config"]