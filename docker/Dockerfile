# This file uses multi-stage builds to build the application from source, including the front-end

# Tags passed to "go build"
ARG BUILD_TAGS=""

# Stage 1: Build Frontend
FROM node:24-alpine AS frontend-builder
RUN corepack enable

WORKDIR /build

COPY pnpm-workspace.yaml pnpm-lock.yaml ./
COPY frontend/package.json ./frontend/
RUN pnpm --filter pocket-id-frontend install --frozen-lockfile

COPY ./frontend ./frontend/

RUN BUILD_OUTPUT_PATH=dist pnpm --filter pocket-id-frontend run build

# Stage 2: Build Backend
FROM golang:1.26-alpine AS backend-builder
ARG BUILD_TAGS
WORKDIR /build
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download

COPY ./backend ./
COPY --from=frontend-builder /build/frontend/dist ./frontend/dist
COPY .version .version

WORKDIR /build/cmd
RUN VERSION=$(cat /build/.version) \ 
  CGO_ENABLED=0 \
  GOOS=linux \
  go build \
    -tags "${BUILD_TAGS}" \
    -ldflags="-X github.com/pocket-id/pocket-id/backend/internal/common.Version=${VERSION} -buildid=${VERSION}" \
    -trimpath \
    -o /build/pocket-id \
    .

# Stage 3: Production Image
FROM alpine
WORKDIR /app

RUN apk add --no-cache curl su-exec

COPY --from=backend-builder /build/pocket-id /app/pocket-id
COPY ./scripts/docker /app/docker

RUN chmod +x /app/pocket-id && \
  find /app/docker -name "*.sh" -exec chmod +x {} \;

EXPOSE 1411
ENV APP_ENV=production

HEALTHCHECK --interval=90s --timeout=5s --start-period=10s --retries=3 CMD [ "/app/pocket-id", "healthcheck" ]

ENTRYPOINT ["sh", "/app/docker/entrypoint.sh"]
CMD ["/app/pocket-id"]
