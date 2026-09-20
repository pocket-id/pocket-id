#!/bin/sh

set -eu

artifact_path=$1
document_path=$2
source_name=$3
source_version=$4
frontend_sbom_path="../.tmp/frontend.cdx.json"

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/pocket-id-binary-sbom.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT

cp "$artifact_path" "$work_dir/$(basename "$artifact_path")"
cp "$frontend_sbom_path" "$work_dir/frontend.cdx.json"

syft "dir:$work_dir" \
    --select-catalogers "+sbom-cataloger" \
    --source-name "$source_name" \
    --source-version "$source_version" \
    --output "spdx-json=$document_path" \
    --enrich all
