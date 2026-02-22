FROM alpine:3.23.3

RUN apk add --no-cache dnsmasq=2.91-r0

USER nobody:nobody
ENTRYPOINT ["dnsmasq"]
CMD ["--keep-in-foreground", "--log-facility=-"]
