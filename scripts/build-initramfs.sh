#!/usr/bin/env bash
set -euo pipefail

out="${1:-dist/bootup-initramfs.cpio}"
mkdir -p "$(dirname "${out}")"

export GOOS="${GOOS:-linux}"
export GOARCH="${GOARCH:-amd64}"
export GOAMD64="${GOAMD64:-v1}"

uinitcmd="${BOOTUP_UINITCMD:-bootup}"

go run github.com/u-root/u-root \
	-build=gbb \
	-o "${out}" \
	-uinitcmd="${uinitcmd}" \
	github.com/u-root/u-root/cmds/core/init \
	github.com/u-root/u-root/cmds/core/gosh \
	github.com/u-root/u-root/cmds/core/ls \
	github.com/u-root/u-root/cmds/core/cat \
	github.com/u-root/u-root/cmds/core/ip \
	github.com/u-root/u-root/cmds/core/dhclient \
	github.com/u-root/u-root/cmds/core/wget \
	github.com/u-root/u-root/cmds/core/mount \
	./cmd/bootup
