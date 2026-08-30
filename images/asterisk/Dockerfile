FROM alpine:3.24

RUN apk add --no-cache \
      asterisk \
      asterisk-sounds-en \
      asterisk-sounds-moh

ENTRYPOINT ["/usr/sbin/asterisk"]
CMD ["-fp"]
