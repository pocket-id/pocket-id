# Tags passed to "go build"
ARG BUILD_TAGS=""

# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY ./frontend/package*.json ./
RUN npm ci
COPY ./frontend ./
RUN npm run build
RUN npm prune --production

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend-builder
ARG BUILD_TAGS
WORKDIR /app/backend
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download

RUN apk add --no-cache gcc musl-dev

COPY ./backend ./
WORKDIR /app/backend/cmd
RUN CGO_ENABLED=1 \
  GOOS=linux \
  go build \
  -tags "${BUILD_TAGS}" \
  -o /app/backend/pocket-id-backend \
  .

# Stage 3: Production Image
FROM node:22-alpine
# Delete default node user
RUN deluser --remove-home node

RUN apk add --no-cache caddy curl su-exec
COPY ./reverse-proxy /etc/caddy/

WORKDIR /app
COPY --from=frontend-builder /app/frontend/build ./frontend/build
COPY --from=frontend-builder /app/frontend/node_modules ./frontend/node_modules
COPY --from=frontend-builder /app/frontend/package.json ./frontend/package.json

COPY --from=backend-builder /app/backend/pocket-id-backend ./backend/pocket-id-backend

COPY ./scripts ./scripts
RUN chmod +x ./scripts/**/*.sh

EXPOSE 80
ENV APP_ENV=production

ENTRYPOINT ["sh", "./scripts/docker/create-user.sh"]
CMD ["sh", "./scripts/docker/entrypoint.sh"]
