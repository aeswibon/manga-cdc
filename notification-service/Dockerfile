FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline -B
COPY src src
RUN mvn package -DskipTests -B

FROM eclipse-temurin:21 AS jlink
RUN jlink --add-modules java.base,java.desktop,java.logging,java.sql,java.xml,java.naming,java.management,jdk.management,jdk.unsupported,java.security.jgss,java.instrument,jdk.crypto.ec,jdk.crypto.cryptoki,java.security.sasl \
          --strip-debug --no-man-pages --no-header-files --compress=2 \
          --output /jre

FROM eclipse-temurin:21 AS libs
COPY --from=jlink /usr/lib /usr/lib
RUN mkdir -p /out/lib && \
    for lib in libdl.so.2 libz.so.1 librt.so.1 libnsl.so.1 libthread_db.so.1 libstdc++.so.6; do \
        find /usr/lib -name "$lib" \( -type f -o -type l \) -exec cp -L {} /out/lib/ \; ; \
    done

FROM busybox:glibc

ARG APP_VERSION=0.4.0
LABEL org.opencontainers.image.title="manga-cdc notification-service" \
      org.opencontainers.image.description="Consumes chapter CDC events from Kafka and sends Discord, Slack, and Telegram notifications" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="docker.io/library/busybox:glibc"

ENV JAVA_OPTS="-XX:MaxRAMPercentage=75.0 -XX:+UseSerialGC"

COPY --from=jlink /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=libs /out/lib /lib
COPY --from=jlink /jre /jre
COPY --from=builder /app/target/*.jar /app.jar
WORKDIR /app
EXPOSE 8080
CMD ["/bin/sh", "-c", "exec /jre/bin/java $JAVA_OPTS -jar /app.jar"]
