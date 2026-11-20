FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go page.html ./
RUN go test ./... && go vet ./... && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /gateway-ui .

FROM scratch
COPY --from=build /gateway-ui /gateway-ui
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/gateway-ui"]
