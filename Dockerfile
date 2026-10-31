# syntax=docker/dockerfile:1

FROM golang:1.27.3-alpine3.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/paperfit ./cmd/main.go

FROM alpine:3.24

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 paperfit \
    && adduser -S -u 10001 -G paperfit -h /app paperfit \
    && mkdir -p /data/cache \
    && chown -R paperfit:paperfit /data

COPY --from=build /out/paperfit /usr/local/bin/paperfit

USER paperfit
WORKDIR /app

# Defaults only; override them at runtime with -e or compose environment.
ENV SERVER_PORT=4000 \
    PDF_GENERATOR_ENDPOINT=http://gotenberg:3000/forms/chromium/convert/html

EXPOSE 4000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${SERVER_PORT}/application/healthy" || exit 1

ENTRYPOINT ["/usr/local/bin/paperfit"]
