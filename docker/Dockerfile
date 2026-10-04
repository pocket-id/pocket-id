# syntax=docker/dockerfile:1

# This file uses multi-stage builds to build the application from source, including the front-end

# Tags passed to "go build"
ARG BUILD_TAGS=""

# Stage 1: Build Frontend
FROM --platform=$BUILDPLATFORM ghcr.io/voidzero-dev/vite-plus:1.0.0 AS frontend-builder
ENV CI=true
WORKDIR /app

COPY --chown=vp:vp package.json pnpm-workspace.yaml pnpm-lock.yaml .node-version ./
COPY --chown=vp:vp --parents frontend/package.json tests/package.json email-templates/package.json ./
RUN --mount=type=cache,id=pocket-id-pnpm,target=/home/vp/.local/share/pnpm/store,sharing=locked,uid=1000,gid=1000 \
  vp -C frontend install --frozen-lockfile --prefer-offline

COPY --chown=vp:vp ./frontend ./frontend/

RUN --mount=type=cache,target=/app/frontend/node_modules/.vite,uid=1000,gid=1000 \
  --mount=type=cache,target=/app/frontend/.svelte-kit,uid=1000,gid=1000 \
  BUILD_OUTPUT_PATH=dist vp -C frontend run build

# Stage 2: Build Backend
FROM golang:1.27.1-alpine AS backend-builder
ARG BUILD_TAGS
WORKDIR /build
COPY ./backend/go.mod ./backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY ./backend ./
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY .version .version

WORKDIR /build/cmd
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  VERSION=$(cat /build/.version) && \
  CGO_ENABLED=0 \
  GOOS=linux \
  go build \
    -tags "${BUILD_TAGS}" \
    -ldflags="-X github.com/pocket-id/pocket-id/backend/internal/common.Version=${VERSION} -buildid=${VERSION}" \
    -trimpath \
    -o /build/pocket-id \
    .

# Stage 3: Production Image
FROM alpine:3.24.2
WORKDIR /app

RUN apk add --no-cache su-exec

COPY --from=backend-builder /build/pocket-id /app/pocket-id
COPY ./scripts/docker /app/docker

RUN chmod +x /app/pocket-id && \
  find /app/docker -name "*.sh" -exec chmod +x {} \;

EXPOSE 1411
ENV APP_ENV=production

HEALTHCHECK --interval=90s --timeout=5s --start-period=10s --retries=3 CMD [ "/app/pocket-id", "healthcheck" ]

ENTRYPOINT ["sh", "/app/docker/entrypoint.sh"]
CMD ["/app/pocket-id"]
