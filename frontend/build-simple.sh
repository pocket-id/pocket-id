#!/bin/sh
# Build the simple QR login page (standalone, no SvelteKit).
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Bundle QRCode library (exposes window.QRCode.toCanvas)
pnpm exec esbuild "$SCRIPT_DIR/simple/src/qrcode-entry.js" \
  --bundle \
  --minify \
  --target=es2015 \
  --global-name=QRCode \
  --outfile="$SCRIPT_DIR/static/simple/qr/qrcode.min.js"

# Build Svelte app
pnpm exec vite build --config "$SCRIPT_DIR/simple/vite.config.ts"
