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
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
