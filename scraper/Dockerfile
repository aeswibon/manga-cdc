FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /scraper ./cmd/scraper

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /scraper /scraper
EXPOSE 2112
CMD ["/scraper"]
