# This file uses multi-stage builds to build the application from source, including the front-end

# Tags passed to "go build"
ARG BUILD_TAGS=""

# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /build
COPY ./frontend/package*.json ./
RUN npm ci
COPY ./frontend ./
RUN BUILD_OUTPUT_PATH=dist npm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend-builder
ARG BUILD_TAGS
WORKDIR /build
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download

COPY ./backend ./
COPY --from=frontend-builder /build/dist ./frontend/dist
COPY .version .version

WORKDIR /build/cmd
RUN VERSION=$(cat /build/.version) \ 
  CGO_ENABLED=0 \
  GOOS=linux \
  go build \
    -tags "${BUILD_TAGS}" \
    -ldflags="-X github.com/pocket-id/pocket-id/backend/internal/common.Version=${VERSION} -buildid=${VERSION}" \
    -trimpath \
    -o /build/pocket-id-backend \
    .

# Stage 3: Production Image
FROM alpine
WORKDIR /app

RUN apk add --no-cache curl su-exec

COPY --from=backend-builder /build/pocket-id-backend /app/pocket-id
COPY ./scripts/docker /app/docker

RUN chmod +x /app/pocket-id && \
  find /app/docker -name "*.sh" -exec chmod +x {} \;

EXPOSE 1411
ENV APP_ENV=production

ENTRYPOINT ["sh", "/app/docker/entrypoint.sh"]
CMD ["/app/pocket-id"]
