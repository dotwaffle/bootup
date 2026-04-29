#!/usr/bin/env bash
set -euo pipefail

keyring="${1:-/usr/share/keyrings/debian-archive-keyring.gpg}"
out="${2:-dist/bootup-debian-initramfs.cpio}"
uinitcmd="${3:-bootup --mode=menu --prepare-runtime}"
extra_files="${4:-}"
generated="internal/trustmaterial/debian_archive_keyring_generated.go"

if [[ ! -r "${keyring}" ]]; then
	echo "keyring is not readable: ${keyring}" >&2
	exit 1
fi

cleanup() {
	rm -f "${generated}"
}
trap cleanup EXIT

go run ./cmd/bootup-keyring-source -o "${generated}" "${keyring}"
GOCACHE="${GOCACHE:-/tmp/bootup-go-build-cache}" scripts/build-initramfs.sh "${out}" "${uinitcmd}" "" "${extra_files}"
