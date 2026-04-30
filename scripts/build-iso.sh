#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'USAGE'
usage: scripts/build-iso.sh [output.iso]

Environment:
  BOOTUP_ISO_KERNEL      kernel image to place in the ISO
  BOOTUP_ISO_INITRAMFS   zstd initramfs to place in the ISO
  BOOTUP_ISO_CMDLINE     kernel command line for the GRUB menu entry
  BOOTUP_ISO_UINITCMD    u-root init command when building the default initramfs
  BOOTUP_ISO_VOLUME_ID   ISO volume ID, default BOOTUP
  BOOTUP_ISO_ALLOW_BIOS_ONLY=1
                         allow building when GRUB EFI modules are absent
USAGE
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf '%s not found; install %s to build the bootup ISO\n' "$1" "$1" >&2
		exit 1
	fi
}

latest_kernel() {
	local kernel_dir="$1"
	local -a kernels=()

	if [[ -d "${kernel_dir}" ]]; then
		mapfile -t kernels < <(find "${kernel_dir}" -maxdepth 1 -type f -name 'linux-*-bootup-amd64-bzImage' | sort -V)
	fi
	if [[ "${#kernels[@]}" -eq 0 ]]; then
		printf 'no bootup kernel found under %s; run scripts/build-kernel.sh or set BOOTUP_ISO_KERNEL\n' "${kernel_dir}" >&2
		exit 1
	fi
	printf '%s\n' "${kernels[-1]}"
}

abs_path() {
	local path="$1"
	local dir
	dir="$(cd -- "$(dirname -- "${path}")" && pwd)"
	printf '%s/%s\n' "${dir}" "$(basename -- "${path}")"
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

require_cmd grub-mkrescue
require_cmd xorriso

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ ! -d /usr/lib/grub/x86_64-efi && "${BOOTUP_ISO_ALLOW_BIOS_ONLY:-}" != "1" ]]; then
	cat >&2 <<'EOF'
GRUB x86_64 EFI modules are not installed, so grub-mkrescue would build a
BIOS-only ISO. Install grub-efi-amd64-bin for a hybrid BIOS/UEFI image, or set
BOOTUP_ISO_ALLOW_BIOS_ONLY=1 for a BIOS-only smoke artifact.
EOF
	exit 1
fi

out="${1:-${BOOTUP_ISO_OUT:-${repo_root}/dist/bootup.iso}}"
volume_id="${BOOTUP_ISO_VOLUME_ID:-BOOTUP}"
cmdline="${BOOTUP_ISO_CMDLINE:-console=tty0 console=ttyS0,115200n8 panic=30 ip=::::::dhcp}"

kernel="${BOOTUP_ISO_KERNEL:-$(latest_kernel "${repo_root}/dist/kernel")}"
if [[ ! -r "${kernel}" ]]; then
	printf 'kernel is not readable: %s\n' "${kernel}" >&2
	exit 1
fi
kernel="$(abs_path "${kernel}")"

initramfs="${BOOTUP_ISO_INITRAMFS:-${repo_root}/dist/bootup-iso-initramfs.cpio.zst}"
if [[ -z "${BOOTUP_ISO_INITRAMFS:-}" && ! -f "${initramfs}" ]]; then
	uinitcmd="${BOOTUP_ISO_UINITCMD:-bootup --mode=menu --ui=auto --prepare-runtime}"
	"${repo_root}/scripts/build-initramfs.sh" \
		"${repo_root}/dist/bootup-iso-initramfs.cpio" \
		"${uinitcmd}" \
		"${BOOTUP_ISO_GO_TAGS:-}" \
		"${BOOTUP_ISO_EXTRA_FILES:-}"
fi
if [[ ! -r "${initramfs}" ]]; then
	printf 'initramfs is not readable: %s\n' "${initramfs}" >&2
	exit 1
fi
initramfs="$(abs_path "${initramfs}")"

mkdir -p "$(dirname -- "${out}")"
tmp="$(mktemp -d "$(dirname -- "${out}")/iso.XXXXXX")"
cleanup() {
	rm -rf "${tmp}"
}
trap cleanup EXIT

iso_root="${tmp}/root"
mkdir -p "${iso_root}/boot/bootup" "${iso_root}/boot/grub"
install -m 0644 "${kernel}" "${iso_root}/boot/bootup/vmlinuz"
install -m 0644 "${initramfs}" "${iso_root}/boot/bootup/initramfs.cpio.zst"
cat >"${iso_root}/boot/bootup/manifest" <<EOF
kernel=$(basename -- "${kernel}")
initramfs=$(basename -- "${initramfs}")
cmdline=${cmdline}
EOF
cat >"${iso_root}/boot/grub/grub.cfg" <<EOF
set default=0
set timeout=3

serial --unit=0 --speed=115200 --word=8 --parity=no --stop=1
terminal_input console serial
terminal_output console serial

menuentry "bootup" {
    echo "Loading bootup kernel..."
    linux /boot/bootup/vmlinuz ${cmdline}
    echo "Loading bootup initramfs..."
    initrd /boot/bootup/initramfs.cpio.zst
}
EOF

grub-mkrescue \
	--output="${out}" \
	--product-name=bootup \
	--product-version=0 \
	"${iso_root}" \
	-volid "${volume_id}" \
	>/dev/null

stat -c "iso %n %s bytes" "${out}"
trap - EXIT
cleanup
