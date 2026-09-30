#!/usr/bin/env bash
# publish.sh — build this fork and publish the npm/poc package as a tarball to
# the Oasis rustfs bucket, consumed by the poc monorepo as a tarball-URL dependency.
set -euo pipefail

S3_ENDPOINT="${S3_ENDPOINT:-http://127.0.0.1:9100}"
S3_BUCKET="${S3_BUCKET:-oasis-npm}"
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-minioadmin}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-minioadmin}"
export AWS_REGION="${AWS_REGION:-us-east-1}"
unset AWS_PROFILE

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$repo_root"

# `just init` minus the pnpm steps, which only the e2e suite and `just fmt` need.
# Guarded because `git am` is not idempotent.
if [[ ! -f typescript-go/go.mod ]]; then
  echo ">> initialising typescript-go submodule"
  git submodule update --init
  (cd typescript-go && git am --3way --no-gpg-sign ../patches/*.patch)
  mkdir -p internal/collections
  find ./typescript-go/internal/collections -type f ! -name '*_test.go' \
    -exec cp {} internal/collections/ \;
fi

version="$(git describe --tags --abbrev=0 --match 'v*' | sed 's/^v//')-poc-$(git rev-parse HEAD)"
tarball="oxlint-tsgolint-${version}.tgz"

stage="$(mktemp -d)/package"
trap 'rm -rf "$(dirname "$stage")"' EXIT
cp -r npm/poc "$stage"
cp LICENSE README.md "$stage/"
npm --prefix "$stage" version "$version" --no-git-tag-version --allow-same-version >/dev/null

# Go cross-compiles cleanly here: tsgolint has no cgo.
export CGO_ENABLED=0
for target in darwin/arm64/darwin-arm64 linux/amd64/linux-x64; do
  IFS=/ read -r goos goarch npm_platform <<<"$target"
  echo ">> building tsgolint for ${npm_platform}"
  GOOS="$goos" GOARCH="$goarch" go build \
    -ldflags="-s -w" -trimpath \
    -o "$stage/bin/tsgolint-${npm_platform}" ./cmd/tsgolint
done

out="$(dirname "$stage")/${tarball}"
tar -czf "$out" -C "$(dirname "$stage")" package
aws --endpoint-url "$S3_ENDPOINT" s3 cp "$out" "s3://${S3_BUCKET}/${tarball}" --no-progress

echo
echo "poc package.json -> \"oxlint-tsgolint\": \"${S3_ENDPOINT}/${S3_BUCKET}/${tarball}\""
