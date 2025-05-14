# Tags passed to "go build"
ARG BUILD_TAGS=""

# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY ./frontend/package*.json ./
RUN npm ci
COPY ./frontend ./
RUN BUILD_OUTPUT_PATH=dist npm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend-builder
ARG BUILD_TAGS
WORKDIR /app/backend
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download

RUN apk add --no-cache gcc musl-dev

COPY ./backend ./
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY .version .version


WORKDIR /app/backend/cmd
RUN VERSION=$(cat /app/backend/.version) \ 
  CGO_ENABLED=0 \
  GOOS=linux \
  go build \
  -tags "${BUILD_TAGS}" \
  -ldflags="-X github.com/pocket-id/pocket-id/backend/internal/common.Version=${VERSION}" \
  -o /app/backend/pocket-id-backend \
  .

# Stage 3: Production Image
FROM alpine

RUN apk add --no-cache curl su-exec

WORKDIR /app

COPY --from=backend-builder /app/backend/pocket-id-backend ./backend/pocket-id

COPY ./scripts ./scripts
RUN find ./scripts -name "*.sh" -exec chmod +x {} \;
RUN chmod +x ./backend/pocket-id

EXPOSE 80
ENV APP_ENV=production 

ENTRYPOINT ["sh", "./scripts/docker/entrypoint.sh"]
CMD ["/app/backend/pocket-id"]
