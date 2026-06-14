# Production image: GraalVM native binary on distroless glibc (linux/amd64 in CI).
FROM ghcr.io/graalvm/native-image-community:21 AS builder
WORKDIR /app
COPY mvnw .
COPY .mvn .mvn
COPY pom.xml .
RUN ./mvnw dependency:go-offline -B
COPY src src
RUN ./mvnw -Pnative native:compile -DskipTests -B

# Native binary is dynamically linked; copy runtime libs from Debian bookworm (distroless base).
FROM debian:bookworm-slim AS libs
RUN apt-get update \
    && apt-get install -y --no-install-recommends zlib1g libstdc++6 \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /out/lib \
    && cp -L /lib/x86_64-linux-gnu/libz.so.1 /lib/x86_64-linux-gnu/libdl.so.2 /lib/x86_64-linux-gnu/libpthread.so.0 /lib/x86_64-linux-gnu/libm.so.6 /lib/x86_64-linux-gnu/librt.so.1 /out/lib/ \
    && cp -L /usr/lib/x86_64-linux-gnu/libgcc_s.so.1 /usr/lib/x86_64-linux-gnu/libstdc++.so.6 /out/lib/

FROM gcr.io/distroless/base-debian12

ARG APP_VERSION=0.4.5
LABEL org.opencontainers.image.title="manga-cdc notification-service" \
      org.opencontainers.image.description="Consumes chapter CDC events from Kafka and sends Discord, Slack, and Telegram notifications" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="gcr.io/distroless/base-debian12"

COPY --from=libs /out/lib/* /lib/x86_64-linux-gnu/
COPY --from=busybox:1.36-glibc /bin/wget /bin/wget

COPY --from=builder /app/target/notification-service /app/notification-service
WORKDIR /app
EXPOSE 8080
CMD ["/app/notification-service"]
