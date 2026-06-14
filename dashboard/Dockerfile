# Build stage
FROM oven/bun:1-alpine AS build
WORKDIR /app
ARG VITE_STATUS_PAGE_URL=http://localhost:3001
ENV VITE_STATUS_PAGE_URL=$VITE_STATUS_PAGE_URL
COPY package.json bun.lock* ./
RUN bun install
COPY . .
RUN bun run build

# Serve stage
FROM nginx:alpine

ARG APP_VERSION=0.4.6
LABEL org.opencontainers.image.title="manga-cdc dashboard" \
      org.opencontainers.image.description="Read-only manga watchlist dashboard for manga-cdc" \
      org.opencontainers.image.source="https://github.com/aeswibon/manga-cdc" \
      org.opencontainers.image.vendor="aeswibon" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.base.name="nginx:alpine"

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
