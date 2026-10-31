FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build
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
