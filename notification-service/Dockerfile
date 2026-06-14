FROM eclipse-temurin:21-jdk-jammy AS builder
WORKDIR /app
COPY mvnw .
COPY .mvn .mvn
COPY pom.xml .
RUN ./mvnw dependency:go-offline -B
COPY src src
RUN ./mvnw package -DskipTests -B

FROM eclipse-temurin:21-jre-jammy

ARG APP_VERSION=0.4.5
LABEL org.opencontainers.image.title="manga-cdc notification-service" \
      org.opencontainers.image.description="Consumes chapter CDC events from Kafka and sends Discord, Slack, and Telegram notifications" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="eclipse-temurin:21-jre-jammy"

RUN apt-get update \
    && apt-get install -y --no-install-recommends wget \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/target/notification-service-*.jar /app/app.jar
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["java", "-XX:+UseContainerSupport", "-XX:MaxRAMPercentage=75.0", "-jar", "/app/app.jar"]
