#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 /path/to/kernel.config" >&2
	exit 2
fi

config="$1"
if [[ ! -r "${config}" ]]; then
	echo "kernel config is not readable: ${config}" >&2
	exit 2
fi

required_y=(
	CONFIG_BLK_DEV_INITRD
	CONFIG_DEVTMPFS
	CONFIG_DEVTMPFS_MOUNT
	CONFIG_PROC_FS
	CONFIG_SYSFS
	CONFIG_TMPFS
	CONFIG_KEXEC
	CONFIG_KEXEC_FILE
	CONFIG_NET
	CONFIG_NETDEVICES
	CONFIG_ETHERNET
	CONFIG_INET
	CONFIG_IP_PNP
	CONFIG_IP_PNP_DHCP
	CONFIG_PCI
	CONFIG_E1000
	CONFIG_VIRTIO
	CONFIG_VIRTIO_PCI
	CONFIG_VIRTIO_NET
	CONFIG_RD_ZSTD
)

status=0
for symbol in "${required_y[@]}"; do
	if grep -qx "${symbol}=y" "${config}"; then
		continue
	fi
	if grep -qx "${symbol}=m" "${config}"; then
		echo "${symbol}=m, want ${symbol}=y for early boot" >&2
	elif grep -qx "# ${symbol} is not set" "${config}"; then
		echo "${symbol} is not set, want ${symbol}=y" >&2
	else
		echo "${symbol} is missing, want ${symbol}=y" >&2
	fi
	status=1
done

exit "${status}"
