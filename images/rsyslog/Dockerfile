ARG ALPINE_VERSION=3.24.4
FROM alpine:${ALPINE_VERSION}

RUN apk upgrade --no-cache \
 && apk add --no-cache rsyslog \
 && addgroup -S -g 10001 collector \
 && adduser -S -D -H -u 10001 -G collector collector

USER 10001:10001
ENTRYPOINT ["rsyslogd", "-n", "-iNONE", "-f", "/etc/rsyslog.conf"]
