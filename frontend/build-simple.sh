#!/bin/sh
# Bundle the qrcode library for the simple login page.
# Exposes window.QRCode.toCanvas() for use in vanilla JS.
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

pnpm exec esbuild "$SCRIPT_DIR/simple/src/qrcode-entry.js" \
  --bundle \
  --minify \
  --target=es2015 \
  --global-name=QRCode \
  --outfile="$SCRIPT_DIR/static/simple/qr/qrcode.min.js"
