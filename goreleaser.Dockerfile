FROM alpine:latest

LABEL org.opencontainers.image.source="https://github.com/autobrr/autobrr"

ENV HOME="/config" \
XDG_CONFIG_HOME="/config" \
XDG_DATA_HOME="/config"

#RUN ["apk", "--no-cache", "add", "ca-certificates","curl"]

COPY autobrr /usr/local/bin/autobrr
COPY autobrrctl /usr/local/bin/autobrrctl

WORKDIR /config

VOLUME /config

EXPOSE 7474

ENTRYPOINT ["/usr/local/bin/autobrr", "--config", "/config"]
#CMD ["--config", "/config"]
