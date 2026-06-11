FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /scraper ./cmd/scraper

FROM alpine:3.20

ARG IMAGE_REVISION=unknown
ARG IMAGE_VERSION=dev

LABEL org.opencontainers.image.title="manga-cdc scraper" \
      org.opencontainers.image.description="Scrapes manga sources and publishes new chapter events to Kafka" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.base.name="docker.io/library/alpine:3.20" \
      org.opencontainers.image.revision="${IMAGE_REVISION}" \
      org.opencontainers.image.version="${IMAGE_VERSION}"

RUN apk --no-cache add ca-certificates
COPY --from=builder /scraper /scraper
EXPOSE 2112
CMD ["/scraper"]
