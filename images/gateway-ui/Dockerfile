FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS build
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
