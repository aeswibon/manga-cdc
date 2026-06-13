FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline -B
COPY src src
RUN mvn package -DskipTests -B

FROM eclipse-temurin:21-jdk-alpine AS jlink
RUN jlink --add-modules java.base,java.desktop,java.logging,java.sql,java.xml,java.naming,java.management,jdk.management,jdk.unsupported,java.security.jgss,java.instrument \
          --strip-debug --no-man-pages --no-header-files --compress=2 \
          --output /jre

FROM alpine:3.20

ARG APP_VERSION=0.3.7
LABEL org.opencontainers.image.title="manga-cdc notification-service" \
      org.opencontainers.image.description="Consumes chapter CDC events from Kafka and sends Discord, Slack, and Telegram notifications" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="docker.io/library/alpine:3.20"

RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=jlink /jre /jre
COPY --from=builder /app/target/*.jar app.jar
EXPOSE 8080
CMD ["/jre/bin/java", "-jar", "app.jar"]
