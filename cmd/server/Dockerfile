FROM golang:1.26.2 AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 go build -o mayfly-server ./cmd/server/

FROM alpine:latest

RUN apk add --no-cache \
    unbound iptables ip6tables ca-certificates dns-root-hints

RUN mkdir -p /var/lib/unbound && unbound-anchor -a /var/lib/unbound/root.key || true

COPY --from=builder /src/mayfly-server /usr/local/bin/mayfly-server
COPY cmd/server/unbound.conf /etc/unbound/custom.conf

EXPOSE 51820/udp
EXPOSE 8080/tcp
ENTRYPOINT [ "/usr/local/bin/mayfly-server" ]