#!/bin/sh
# Prints the Francis version Pocket ID depends on, formatted as the tag of the runtime's container image.
#
# The Go module is the single source of truth for the version: the standalone runtime the E2E tests run against
# has to be the same version as the client Pocket ID is built with, and pinning it in two places lets them drift.
#
# Usage, from this directory:
#   echo "FRANCIS_VERSION=$(./francis-version.sh)" > .env
#
# This is a plain shell script rather than "go list" because the E2E workflow does not set up Go.
set -eu

go_mod="$(CDPATH='' cd -- "$(dirname -- "$0")/../../backend" && pwd)/go.mod"

if [ ! -f "$go_mod" ]; then
	echo "francis-version.sh: cannot find $go_mod" >&2
	exit 1
fi

# Module versions carry a leading "v" that container tags do not, so it is stripped
version=$(sed -n 's|^[[:space:]]*github\.com/italypaleale/francis v\([^[:space:]]*\).*|\1|p' "$go_mod" | head -n 1)

if [ -z "$version" ]; then
	echo "francis-version.sh: no github.com/italypaleale/francis requirement found in $go_mod" >&2
	exit 1
fi

printf '%s\n' "$version"
