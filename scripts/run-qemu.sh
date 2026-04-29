#!/usr/bin/env bash
set -euo pipefail

kernel="${BOOTUP_KERNEL:-/boot/vmlinuz-$(uname -r)}"
initramfs="${BOOTUP_INITRAMFS:-dist/bootup-initramfs.cpio.zst}"
cmdline="${BOOTUP_CMDLINE:-console=ttyS0 panic=30}"

qemu-system-x86_64 \
	-m "${BOOTUP_MEMORY:-2048}" \
	-nographic \
	-kernel "${kernel}" \
	-initrd "${initramfs}" \
	-append "${cmdline}"
