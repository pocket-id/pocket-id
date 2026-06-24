#!/bin/bash

docker buildx build --push --file docker/Dockerfile --tag pocketid/pocket-id:development --tag ghcr.io/pocket-id/pocket-id:development --platform linux/amd64,linux/arm64 .
