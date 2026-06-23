#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-}"
PLATFORMS="${PLATFORMS:-"linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"}"

if [ -z "$VERSION" ]; then
	echo "usage: VERSION=v0.1.0 make release-dry-run" >&2
	exit 1
fi

if ! printf '%s' "$VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z][0-9A-Za-z.-]*)?$'; then
	echo "release version must look like v0.1.0 or v0.1.0-rc.1: $VERSION" >&2
	exit 1
fi

if printf '%s' "$VERSION" | grep -q 'dirty'; then
	echo "release dry run refuses dirty version strings: $VERSION" >&2
	exit 1
fi

verify_checksums() {
	if command -v sha256sum >/dev/null 2>&1; then
		(cd "$ROOT_DIR/dist" && sha256sum -c checksums.txt)
	else
		(cd "$ROOT_DIR/dist" && shasum -a 256 -c checksums.txt)
	fi
}

require_archive_member() {
	local archive="$1"
	local member="$2"

	if ! tar -tzf "$archive" | grep -qx "$member"; then
		echo "missing $member in $archive" >&2
		exit 1
	fi
}

verify_local_archive() {
	local goos
	local goarch
	local safe_version
	local archive
	local extract_dir

	goos="$(go env GOOS)"
	goarch="$(go env GOARCH)"

	case " $PLATFORMS " in
		*" $goos/$goarch "*) ;;
		*)
			echo "skipping local binary execution; $goos/$goarch is not in PLATFORMS"
			return
			;;
	esac

	safe_version="$(printf '%s' "$VERSION" | tr '/[:space:]' '-')"
	archive="$ROOT_DIR/dist/ard_${safe_version}_${goos}_${goarch}.tar.gz"
	extract_dir="$(mktemp -d)"
	trap 'rm -rf "$extract_dir"' RETURN

	if [ ! -f "$archive" ]; then
		echo "missing local platform archive: $archive" >&2
		exit 1
	fi

	require_archive_member "$archive" "ard"
	require_archive_member "$archive" "ardctl"
	require_archive_member "$archive" "ard-server"
	require_archive_member "$archive" "README.md"
	require_archive_member "$archive" "LICENSE"
	require_archive_member "$archive" "VERSION"

	tar -xzf "$archive" -C "$extract_dir"
	grep -qx "$VERSION" "$extract_dir/VERSION"
	"$extract_dir/ard" version | grep -Fq "$VERSION"
	"$extract_dir/ardctl" version --json | grep -Fq "$VERSION"
	"$extract_dir/ard-server" --version | grep -Fq "$VERSION"
	rm -rf "$extract_dir"
	trap - RETURN
}

cd "$ROOT_DIR"

echo "release dry run: $VERSION"
make fmt-check
make check-public-surface
make test-public-go-client
VERSION="$VERSION" PLATFORMS="$PLATFORMS" make package
verify_checksums
verify_local_archive
echo "release dry run completed: $VERSION"
