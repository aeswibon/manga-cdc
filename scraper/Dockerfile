FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY scraper/go.mod scraper/go.sum ./
RUN go mod download
COPY scraper/ .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /scraper ./cmd/scraper

RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -S mangacdc && adduser -S mangacdc -G mangacdc

FROM scratch

ARG APP_VERSION=0.4.6
LABEL org.opencontainers.image.title="manga-cdc scraper" \
      org.opencontainers.image.description="Scrapes manga sources and publishes new chapter events to Kafka" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="scratch"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /scraper /scraper
COPY --chown=mangacdc:mangacdc db/migrations /migrations
COPY --chown=mangacdc:mangacdc data/watchlist.yaml /data/watchlist.yaml
ENV MIGRATIONS_DIR=/migrations
ENV WATCHLIST_PATH=/data/watchlist.yaml
EXPOSE 2112
USER mangacdc
CMD ["/scraper"]
