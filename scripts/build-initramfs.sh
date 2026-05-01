#!/usr/bin/env bash
set -euo pipefail

out="${1:-dist/bootup-initramfs.cpio}"
zstd_out="${BOOTUP_INITRAMFS_ZSTD:-${out}.zst}"
mkdir -p "$(dirname "${out}")"
mkdir -p "$(dirname "${zstd_out}")"

export GOOS="${GOOS:-linux}"
export GOARCH="${GOARCH:-amd64}"
export GOAMD64="${GOAMD64:-v1}"

uinitcmd="${2:-${BOOTUP_UINITCMD:-bootup --hold}}"
go_build_tags="${3:-}"
extra_files="${4:-}"

u_root_args=(
	-build=gbb
	-o "${out}"
	-uinitcmd="${uinitcmd}"
)
if [[ -n "${go_build_tags}" ]]; then
	u_root_args+=(-go-build-tags "${go_build_tags}")
fi
if [[ -n "${extra_files}" ]]; then
	u_root_args+=(-files "${extra_files}")
fi

go run github.com/u-root/u-root \
	"${u_root_args[@]}" \
	github.com/u-root/u-root/cmds/core/init \
	github.com/u-root/u-root/cmds/core/gosh \
	github.com/u-root/u-root/cmds/core/ls \
	github.com/u-root/u-root/cmds/core/cat \
	github.com/u-root/u-root/cmds/core/ip \
	github.com/u-root/u-root/cmds/core/wget \
	github.com/u-root/u-root/cmds/core/mount \
	github.com/u-root/u-root/cmds/core/insmod \
	github.com/u-root/u-root/cmds/boot/boot \
	./cmd/bootup

zstd -q -f -19 --keep "${out}" -o "${zstd_out}"
