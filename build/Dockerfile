
FROM golang:latest AS builder
WORKDIR $GOPATH/src/github.com/nm-morais/CyclonTMan
COPY --from=nmmorais/go-babel:latest /src/github.com/nm-morais/go-babel ../go-babel/
COPY . .
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/CyclonTMan *.go


FROM debian:stable-slim as cyclon_tman
RUN apt update 2>/dev/null | grep -P "\d\K upgraded" ; apt install iproute2 -y 2>/dev/null; apt install kmod -y 2>/dev/null
COPY scripts/setupTc.sh /setupTc.sh
COPY build/docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=builder /go/bin/CyclonTMan /go/bin/CyclonTMan
COPY config /config

ARG LATENCY_MAP
ARG IPS_FILE

COPY ${LATENCY_MAP} /latencyMap.txt
COPY ${IPS_FILE} /config.txt
RUN chmod +x /setupTc.sh /docker-entrypoint.sh /go/bin/CyclonTMan

ENTRYPOINT ["/docker-entrypoint.sh", "/latencyMap.txt", "/config.txt"]