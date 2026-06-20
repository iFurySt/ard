#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-"$ROOT_DIR/dist"}"
VERSION="${VERSION:-}"
PLATFORMS="${PLATFORMS:-"linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"}"

if [ -z "$VERSION" ]; then
	if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
		VERSION="$(git -C "$ROOT_DIR" describe --tags --always --dirty)"
	else
		VERSION="dev"
	fi
fi

SAFE_VERSION="$(printf '%s' "$VERSION" | tr '/[:space:]' '-')"
TMP_DIR="$(mktemp -d)"
GO_BUILD_FLAGS=(-trimpath -buildvcs=false -ldflags="-s -w")
CREATED="${SOURCE_DATE_EPOCH:-}"

if [ -n "$CREATED" ]; then
	if date -u -d "@$CREATED" '+%Y-%m-%dT%H:%M:%SZ' >/dev/null 2>&1; then
		CREATED="$(date -u -d "@$CREATED" '+%Y-%m-%dT%H:%M:%SZ')"
	else
		CREATED="$(date -u -r "$CREATED" '+%Y-%m-%dT%H:%M:%SZ')"
	fi
elif git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	CREATED="$(git -C "$ROOT_DIR" log -1 --format=%cI)"
else
	CREATED="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
fi

cleanup() {
	rm -rf "$TMP_DIR"
}
trap cleanup EXIT

checksum_file() {
	local path="$1"
	local dir
	local file
	dir="$(dirname "$path")"
	file="$(basename "$path")"

	if command -v sha256sum >/dev/null 2>&1; then
		(cd "$dir" && sha256sum "$file")
	else
		(cd "$dir" && shasum -a 256 "$file")
	fi
}

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for platform in $PLATFORMS; do
	GOOS="${platform%/*}"
	GOARCH="${platform#*/}"
	STAGE_DIR="$TMP_DIR/ard_${SAFE_VERSION}_${GOOS}_${GOARCH}"
	ARCHIVE_NAME="ard_${SAFE_VERSION}_${GOOS}_${GOARCH}.tar.gz"
	ARCHIVE_TAR="$DIST_DIR/${ARCHIVE_NAME%.gz}"
	EXE_SUFFIX=""

	if [ "$GOOS" = "windows" ]; then
		EXE_SUFFIX=".exe"
	fi

	mkdir -p "$STAGE_DIR"

	echo "building $ARCHIVE_NAME"
	CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build "${GO_BUILD_FLAGS[@]}" -o "$STAGE_DIR/ard$EXE_SUFFIX" ./cmd/ard
	CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build "${GO_BUILD_FLAGS[@]}" -o "$STAGE_DIR/ardctl$EXE_SUFFIX" ./cmd/ardctl
	CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build "${GO_BUILD_FLAGS[@]}" -o "$STAGE_DIR/ard-server$EXE_SUFFIX" ./cmd/ard-server

	cp "$ROOT_DIR/README.md" "$STAGE_DIR/README.md"
	cp "$ROOT_DIR/LICENSE" "$STAGE_DIR/LICENSE"
	printf '%s\n' "$VERSION" > "$STAGE_DIR/VERSION"

	MEMBERS=("ard$EXE_SUFFIX" "ardctl$EXE_SUFFIX" "ard-server$EXE_SUFFIX" "README.md" "LICENSE" "VERSION")
	COPYFILE_DISABLE=1 tar -cf "$ARCHIVE_TAR" -C "$STAGE_DIR" "${MEMBERS[@]}"
	gzip -n -f "$ARCHIVE_TAR"
done

echo "generating SPDX SBOM"
go run ./internal/tools/sbom -version "$VERSION" -created "$CREATED" -out "$DIST_DIR/sbom.spdx.json"

CHECKSUMS_TMP="$TMP_DIR/checksums.txt"
: > "$CHECKSUMS_TMP"

for artifact in "$DIST_DIR"/*.tar.gz "$DIST_DIR"/sbom.spdx.json; do
	checksum_file "$artifact" >> "$CHECKSUMS_TMP"
done

sort -k 2 "$CHECKSUMS_TMP" > "$DIST_DIR/checksums.txt"
echo "wrote $DIST_DIR/checksums.txt"
