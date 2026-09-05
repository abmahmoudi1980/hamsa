# Hamsa backend — multi-stage build.
# Runtime config comes from HAMSA_CONFIG (baked prod config, no secrets) plus
# environment overrides (HAMSA_DB_DSN, HAMSA_JWT_SECRET, ...).
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
# ca-certificates: outbound HTTPS (SMS provider, payment gateway).
RUN addgroup -S hamsa && adduser -S hamsa -G hamsa && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server /app/server
COPY migrations /app/migrations
COPY config.prod.yaml /app/config.prod.yaml
RUN mkdir -p /data/files && chown -R hamsa:hamsa /data /app
USER hamsa
ENV HAMSA_CONFIG=/app/config.prod.yaml
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/app/server"]
