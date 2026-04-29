#!/usr/bin/env bash
set -euo pipefail

keyring="${1:-/usr/share/keyrings/debian-archive-keyring.gpg}"
out="${2:-dist/bootup-debian-smoke-initramfs.cpio}"
kernel="${3:-/boot/vmlinuz-$(uname -r)}"
timeout_seconds="${4:-300}"
net_module="$(modinfo -n e1000 2>/dev/null || true)"
uinitcmd="ip link set eth0 up; ip addr add 10.0.2.15/24 dev eth0 || true; ip route add default via 10.0.2.2 dev eth0 || true; echo nameserver 10.0.2.3 >/etc/resolv.conf; bootup --mode=boot-target --target=debian-trixie-amd64-netboot --staging-dir=/tmp/bootup"

if [[ -n "${net_module}" && -r "${net_module}" ]]; then
	uinitcmd="gosh -c 'insmod ${net_module} || true; ${uinitcmd}'"
else
	uinitcmd="gosh -c '${uinitcmd}'"
fi

scripts/build-debian-initramfs.sh \
	"${keyring}" \
	"${out}" \
	"${uinitcmd}" \
	"${net_module}"

timeout "${timeout_seconds}" qemu-system-x86_64 \
	-m 2048 \
	-nographic \
	-netdev user,id=net0 \
	-device e1000,netdev=net0 \
	-kernel "${kernel}" \
	-initrd "${out}.zst" \
	-append "console=ttyS0 panic=30"
