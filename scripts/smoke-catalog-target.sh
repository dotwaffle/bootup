#!/usr/bin/env bash
set -euo pipefail

if [[ "${BOOTUP_LIVE_CATALOG_SMOKE:-}" != "1" ]]; then
	echo "BOOTUP_LIVE_CATALOG_SMOKE=1 is required" >&2
	exit 2
fi

target_id="${1:?usage: scripts/smoke-catalog-target.sh <target-id> [timeout-seconds]}"
if [[ ! "${target_id}" =~ ^[A-Za-z0-9_.-]+$ ]]; then
	echo "target ID contains unsupported characters: ${target_id}" >&2
	exit 2
fi
timeout_seconds="${2:-120}"
initramfs="${BOOTUP_CATALOG_SMOKE_INITRAMFS:-/tmp/bootup-${target_id}-initramfs.cpio}"
net_module="${BOOTUP_NET_MODULE:-}"
extra_files="${BOOTUP_EXTRA_FILES:-}"
preload_net=""
if [[ -z "${net_module}" ]] && command -v modinfo >/dev/null 2>&1; then
	net_module="$(modinfo -n e1000 2>/dev/null || true)"
fi
if [[ -n "${net_module}" ]]; then
	extra_files="${extra_files:+${extra_files},}${net_module}"
	preload_net="insmod ${net_module} || true; "
fi
uinitcmd="gosh -c '${preload_net}bootup --mode=boot-target --target=${target_id} --staging-dir=/tmp/bootup --prepare-runtime --net-iface=${BOOTUP_NET_IFACE:-eth0} --net-address=${BOOTUP_NET_ADDRESS:-10.0.2.15/24} --net-gateway=${BOOTUP_NET_GATEWAY:-10.0.2.2} --net-dns=${BOOTUP_NET_DNS:-10.0.2.3}'"

scripts/build-initramfs.sh \
	"${initramfs}" \
	"${uinitcmd}" \
	"${BOOTUP_BUILD_TAGS:-}" \
	"${extra_files}"

BOOTUP_INITRAMFS="${initramfs}.zst" \
BOOTUP_CMDLINE="${BOOTUP_CMDLINE:-console=ttyS0 ip=dhcp panic=30}" \
timeout "${timeout_seconds}" scripts/run-qemu.sh
