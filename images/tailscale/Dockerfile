ARG TAILSCALE_VERSION=v1.102.3

FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine3.23 AS build

ARG TARGETARCH
ARG TARGETOS
ARG TAILSCALE_VERSION

WORKDIR /src

RUN apk add --no-cache git \
 && git clone --branch "${TAILSCALE_VERSION}" --depth 1 \
      https://github.com/tailscale/tailscale.git . \
 && mkdir /out \
 && GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
      TAGS="ts_kube,ts_package_container,ts_include_cli" \
      ./build_dist.sh --strip -o /out/tailscaled ./cmd/tailscaled

FROM tailscale/tailscale:${TAILSCALE_VERSION} AS upstream

FROM tailscale/alpine-base:3.22

COPY --from=build /out/tailscaled /usr/local/bin/tailscaled
COPY --from=upstream /usr/local/bin/containerboot /usr/local/bin/containerboot

RUN ln -s tailscaled /usr/local/bin/tailscale

CMD ["/usr/local/bin/containerboot"]
