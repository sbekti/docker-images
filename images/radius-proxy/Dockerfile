FROM ghcr.io/sbekti/freeradius:v3.2.8

ENV RADIUS_LISTEN_ADDRESS="*" \
    RADIUS_LISTEN_PORT="1812" \
    RADIUS_ACCOUNTING_LISTEN_PORT="1813" \
    RADIUS_CLIENT_REQUIRE_MESSAGE_AUTHENTICATOR="yes" \
    RADIUS_CLEANUP_DELAY="0" \
    RADIUS_PRIMARY_PORT="1812" \
    RADIUS_PRIMARY_ACCOUNTING_PORT="1813" \
    RADIUS_SECONDARY_PORT="1812" \
    RADIUS_RESPONSE_WINDOW="2" \
    RADIUS_RESPONSE_TIMEOUTS="1" \
    RADIUS_ZOMBIE_PERIOD="20" \
    RADIUS_PRIMARY_STATUS_CHECK="status-server" \
    RADIUS_PRIMARY_CHECK_INTERVAL="10" \
    RADIUS_PRIMARY_CHECK_TIMEOUT="3" \
    RADIUS_PRIMARY_NUM_ANSWERS_TO_ALIVE="3" \
    RADIUS_SECONDARY_STATUS_CHECK="none" \
    RADIUS_SECONDARY_REVIVE_INTERVAL="60"

COPY clients.conf proxy.conf /etc/freeradius/3.0/
COPY default /etc/freeradius/3.0/sites-available/default

RUN sed -i \
      's/^cleanup_delay = 5$/cleanup_delay = $ENV{RADIUS_CLEANUP_DELAY}/' \
      /etc/freeradius/3.0/radiusd.conf \
 && rm -f \
      /etc/freeradius/3.0/mods-enabled/eap \
      /etc/freeradius/3.0/sites-enabled/inner-tunnel \
 && env \
      RADIUS_CLIENT_ADDRESS=192.0.2.0/24 \
      RADIUS_CLIENT_SECRET=build-only-client \
      RADIUS_PRIMARY_ADDRESS=192.0.2.10 \
      RADIUS_PRIMARY_SECRET=build-only-primary \
      RADIUS_SECONDARY_ADDRESS=192.0.2.11 \
      RADIUS_SECONDARY_SECRET=build-only-secondary \
      RADIUS_DEFAULT_VLAN=10 \
      freeradius -XC
