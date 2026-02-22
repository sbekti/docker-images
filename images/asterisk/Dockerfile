FROM    alpine:3.19
LABEL   maintainer=samudra.bekti@gmail.com

RUN     apk --no-cache --update add \
            asterisk \
            asterisk-sounds-en \
            asterisk-sounds-moh

ENTRYPOINT  ["/usr/sbin/asterisk"]
CMD         ["-fp"]
