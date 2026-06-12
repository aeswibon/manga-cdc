FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY scraper/go.mod scraper/go.sum ./
RUN go mod download
COPY scraper/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /scraper ./cmd/scraper

FROM alpine:3.20

ARG APP_VERSION=0.3.7
LABEL org.opencontainers.image.title="manga-cdc scraper" \
      org.opencontainers.image.description="Scrapes manga sources and publishes new chapter events to Kafka" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="docker.io/library/alpine:3.20"

RUN apk --no-cache add ca-certificates
RUN addgroup -S mangacdc && adduser -S mangacdc -G mangacdc
COPY --from=builder /scraper /scraper
COPY --chown=mangacdc:mangacdc db/migrations /migrations
ENV MIGRATIONS_DIR=/migrations
EXPOSE 2112
USER mangacdc
CMD ["/scraper"]
