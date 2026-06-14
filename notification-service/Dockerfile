# Production image: GraalVM native binary on distroless glibc (linux/amd64 in CI).
FROM ghcr.io/graalvm/native-image-community:21 AS builder
WORKDIR /app
COPY mvnw .
COPY .mvn .mvn
COPY pom.xml .
RUN ./mvnw dependency:go-offline -B
COPY src src
RUN ./mvnw -Pnative native:compile -DskipTests -B

FROM gcr.io/distroless/base-debian12

ARG APP_VERSION=0.4.5
LABEL org.opencontainers.image.title="manga-cdc notification-service" \
      org.opencontainers.image.description="Consumes chapter CDC events from Kafka and sends Discord, Slack, and Telegram notifications" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="gcr.io/distroless/base-debian12"

# Static wget for docker-compose healthchecks (distroless has no shell or package manager).
COPY --from=busybox:1.36-glibc /bin/wget /bin/wget

COPY --from=builder /app/target/notification-service /app/notification-service
WORKDIR /app
EXPOSE 8080
CMD ["/app/notification-service"]
