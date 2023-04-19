FROM scratch
ARG SRCAPP=autobrr
ARG SRCCTL=autobrrctl
ARG DST=/app/

LABEL org.opencontainers.image.source = "https://github.com/autobrr/autobrr"
ENV HOME="/config" \
XDG_CONFIG_HOME="/config" \
XDG_DATA_HOME="/config"

VOLUME /config
WORKDIR /app
ADD $SRCAPP $DST
ADD $SRCCTL $DST

EXPOSE 7474
ENTRYPOINT ["/app/autobrr", "--config", "/config"]
