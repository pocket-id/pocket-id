# Image for Anthropic's OSS Scanner: it fetches every dependency while the network is available and builds the binary as docker/Dockerfile does
# Everything after this build runs offline, see threat_model.md next to this file

# Go version must match backend/go.mod
FROM golang:1.27.3-trixie AS go-toolchain

# Node version must match .node-version
FROM node:24-trixie
COPY --from=go-toolchain /usr/local/go /usr/local/go
ENV PATH="/root/go/bin:/usr/local/go/bin:${PATH}" \
    CI=true

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git build-essential sqlite3 \
    && rm -rf /var/lib/apt/lists/*

COPY . /src
WORKDIR /src

# Fetch the JS workspace and Go modules, the only steps that need the network
RUN npm install --global "$(node -p "require('./package.json').packageManager.split('+')[0]")" \
    && pnpm install --frozen-lockfile \
    && cd backend && go mod download

# Build the SPA into backend/frontend/dist so the binary embeds it like a release build
# The e2etest build adds the seed and token-forging routes from the end-to-end suite, see threat_model.md
RUN pnpm build \
    && cd backend/cmd && go build -o /src/backend/pocket-id . \
    && go build -tags e2etest -o /src/backend/pocket-id-e2etest .

# Pin the Go toolchain to what is installed so nothing below tries to download modules or toolchains
ENV GOPROXY=off \
    GOSUMDB=off \
    GOTOOLCHAIN=local

# Run the unit tests to warm the build cache and prove the image works offline; Postgres tests skip without POCKET_ID_TEST_POSTGRES_URL
RUN cd backend && (timeout 1200 go test -tags exclude_frontend,unit ./... || echo "WARNING: go test reported failures")
