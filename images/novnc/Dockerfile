FROM debian:bookworm-slim

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates \
    novnc \
    && rm -rf /var/lib/apt/lists/*

EXPOSE 6080

ENTRYPOINT ["/usr/bin/websockify"]
CMD ["--help"]

