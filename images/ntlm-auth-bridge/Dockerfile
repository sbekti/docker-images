FROM golang:1.26 AS builder

WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /ntlm-auth-bridge .

FROM ubuntu:26.04

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
 && apt-get install -y --no-install-recommends winbind \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /ntlm-auth-bridge /usr/local/bin/ntlm-auth-bridge

EXPOSE 9555

ENTRYPOINT ["/usr/local/bin/ntlm-auth-bridge"]
