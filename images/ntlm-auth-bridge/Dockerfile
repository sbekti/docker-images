FROM golang:1.26.8-alpine3.23 AS builder

WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /ntlm-auth-bridge .

FROM alpine:3.24

RUN apk add --no-cache samba-winbind-clients

COPY --from=builder /ntlm-auth-bridge /usr/local/bin/ntlm-auth-bridge

EXPOSE 9555

ENTRYPOINT ["/usr/local/bin/ntlm-auth-bridge"]
