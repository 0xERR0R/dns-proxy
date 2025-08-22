# build stage
FROM golang:1.24.6-alpine AS build-env
RUN apk add --no-cache \
    git \
    make \
    gcc \
    libc-dev \
    zip

ENV GO111MODULE=on \
    CGO_ENABLED=0
    
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# add source
ADD . .

RUN make build

# final stage
FROM alpine:3.14

LABEL org.opencontainers.image.source="https://github.com/0xERR0R/dns-proxy" \
      org.opencontainers.image.url="https://github.com/0xERR0R/dns-proxy" \
      org.opencontainers.image.title="DNS/DoT to DoH proxy with load-balancing, fail-over and SSL certificate management"

RUN apk add --no-cache ca-certificates tzdata
COPY --from=build-env /src/bin/dns-proxy /app/dns-proxy

EXPOSE 853

WORKDIR /app

ENTRYPOINT ["/app/dns-proxy"]
