#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'USAGE'
usage: scripts/smoke-freebsd-kboot.sh

Build a temporary bootup ISO containing FreeBSD loader.kboot, present either a
FreeBSD bootonly ISO or an embedded FreeBSD/mfsBSD root tree to Linux stage-1,
and run a QEMU UEFI smoke. All downloaded and generated artifacts stay outside
the repository.

Environment:
  BOOTUP_FREEBSD_VERSION              FreeBSD release, default 15.0-RELEASE
  BOOTUP_FREEBSD_ARCH                 FreeBSD architecture, default amd64
  BOOTUP_FREEBSD_BASE_URL             Override release directory URL
  BOOTUP_FREEBSD_KBOOT_LOADER         Existing loader.kboot path
  BOOTUP_FREEBSD_KBOOT_HELP           Existing loader.help.kboot path
  BOOTUP_FREEBSD_KBOOT_ISO            Existing uncompressed bootonly ISO path
  BOOTUP_FREEBSD_KBOOT_MFSBSD_ISO     Existing mfsBSD ISO to extract and embed
  BOOTUP_FREEBSD_KBOOT_PAYLOAD_ROOT   Existing FreeBSD/mfsBSD root tree to embed
  BOOTUP_FREEBSD_KBOOT_WORKDIR        Work directory, default /tmp/...
  BOOTUP_FREEBSD_KBOOT_KERNEL         Bootup Linux kernel for the proof ISO
  BOOTUP_FREEBSD_KBOOT_KERNEL_CONFIG  Config to validate for the proof kernel
  BOOTUP_FREEBSD_KBOOT_OVMF_CODE      OVMF CODE image path
  BOOTUP_FREEBSD_KBOOT_OVMF_VARS      OVMF VARS image path
  BOOTUP_FREEBSD_KBOOT_TIMEOUT        QEMU timeout seconds, default 180
  BOOTUP_FREEBSD_KBOOT_TARGET_PATTERN Extended regexp expected after kernel jump
  BOOTUP_FREEBSD_KBOOT_EXTRA_LOADER_ARGS
                                      Extra loader.kboot key=value args
  BOOTUP_FREEBSD_KBOOT_LOG            Serial log path, default workdir/qemu.log
USAGE
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf '%s not found; install %s or provide prebuilt artifacts\n' "$1" "$1" >&2
		exit 2
	fi
}

abs_path() {
	local path="$1"
	local dir
	dir="$(cd -- "$(dirname -- "${path}")" && pwd)"
	printf '%s/%s\n' "${dir}" "$(basename -- "${path}")"
}

normalize_mfsbsd_file() {
	local path="$1"
	if [[ -r "${path}" ]]; then
		return
	fi
	if [[ ! -r "${path}.gz" ]]; then
		printf 'mfsBSD payload is missing %s or %s.gz\n' "${path}" "${path}" >&2
		exit 2
	fi
	gzip -dkf "${path}.gz"
	if [[ ! -r "${path}" ]]; then
		printf 'mfsBSD payload normalization did not create %s\n' "${path}" >&2
		exit 2
	fi
}

normalize_mfsbsd_root() {
	local root="$1"
	normalize_mfsbsd_file "${root}/boot/kernel/kernel"
	normalize_mfsbsd_file "${root}/mfsroot"
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
version="${BOOTUP_FREEBSD_VERSION:-15.0-RELEASE}"
arch="${BOOTUP_FREEBSD_ARCH:-amd64}"
base_url="${BOOTUP_FREEBSD_BASE_URL:-https://download.freebsd.org/releases/${arch}/${arch}/${version}}"
workdir="${BOOTUP_FREEBSD_KBOOT_WORKDIR:-$(mktemp -d /tmp/bootup-freebsd-kboot-smoke.XXXXXX)}"
mkdir -p "${workdir}"
workdir="$(cd -- "${workdir}" && pwd)"

loader="${BOOTUP_FREEBSD_KBOOT_LOADER:-}"
loader_help="${BOOTUP_FREEBSD_KBOOT_HELP:-}"
if [[ -z "${loader}" || -z "${loader_help}" ]]; then
	require_cmd curl
	require_cmd tar

	base_dir="${workdir}/freebsd-base"
	mkdir -p "${base_dir}"
	curl -fsSL "${base_url}/base.txz" |
		tar -xJf - -C "${base_dir}" ./boot/loader.kboot ./boot/loader.help.kboot
	loader="${base_dir}/boot/loader.kboot"
	loader_help="${base_dir}/boot/loader.help.kboot"
fi
if [[ ! -r "${loader}" ]]; then
	printf 'loader.kboot is not readable: %s\n' "${loader}" >&2
	exit 2
fi
if [[ ! -r "${loader_help}" ]]; then
	printf 'loader.help.kboot is not readable: %s\n' "${loader_help}" >&2
	exit 2
fi
loader="$(abs_path "${loader}")"
loader_help="$(abs_path "${loader_help}")"

payload_root="${BOOTUP_FREEBSD_KBOOT_PAYLOAD_ROOT:-}"
mfsbsd_iso="${BOOTUP_FREEBSD_KBOOT_MFSBSD_ISO:-}"
if [[ -n "${payload_root}" && -n "${mfsbsd_iso}" ]]; then
	printf 'set either BOOTUP_FREEBSD_KBOOT_PAYLOAD_ROOT or BOOTUP_FREEBSD_KBOOT_MFSBSD_ISO, not both\n' >&2
	exit 2
fi
if [[ -n "${mfsbsd_iso}" ]]; then
	if [[ ! -r "${mfsbsd_iso}" ]]; then
		printf 'mfsBSD ISO is not readable: %s\n' "${mfsbsd_iso}" >&2
		exit 2
	fi
	require_cmd gzip
	require_cmd xorriso
	mfsbsd_iso="$(abs_path "${mfsbsd_iso}")"
	payload_root="${workdir}/mfsbsd-root"
	if [[ -e "${payload_root}" ]]; then
		printf 'mfsBSD payload root already exists: %s\n' "${payload_root}" >&2
		exit 2
	fi
	mkdir -p "${payload_root}"
	xorriso -osirrox on -indev "${mfsbsd_iso}" -extract / "${payload_root}"
	normalize_mfsbsd_root "${payload_root}"
fi
if [[ -n "${payload_root}" ]]; then
	if [[ ! -d "${payload_root}" ]]; then
		printf 'FreeBSD payload root is not a directory: %s\n' "${payload_root}" >&2
		exit 2
	fi
	payload_root="$(abs_path "${payload_root}")"
fi

freebsd_iso=""
if [[ -z "${payload_root}" ]]; then
	freebsd_iso="${BOOTUP_FREEBSD_KBOOT_ISO:-}"
	if [[ -z "${freebsd_iso}" ]]; then
		require_cmd curl
		require_cmd xz

		iso_xz="${workdir}/FreeBSD-${version}-${arch}-bootonly.iso.xz"
		freebsd_iso="${workdir}/FreeBSD-${version}-${arch}-bootonly.iso"
		curl -fsSL -o "${iso_xz}" "${base_url}/FreeBSD-${version}-${arch}-bootonly.iso.xz"
		xz -dkf "${iso_xz}"
	fi
	if [[ ! -r "${freebsd_iso}" ]]; then
		printf 'FreeBSD ISO is not readable: %s\n' "${freebsd_iso}" >&2
		exit 2
	fi
	freebsd_iso="$(abs_path "${freebsd_iso}")"
fi

kernel="${BOOTUP_FREEBSD_KBOOT_KERNEL:-${BOOTUP_ISO_KERNEL:-}}"
if [[ -n "${kernel}" ]]; then
	if [[ ! -r "${kernel}" ]]; then
		printf 'bootup kernel is not readable: %s\n' "${kernel}" >&2
		exit 2
	fi
	kernel="$(abs_path "${kernel}")"
	export BOOTUP_ISO_KERNEL="${kernel}"
fi

kernel_config="${BOOTUP_FREEBSD_KBOOT_KERNEL_CONFIG:-}"
if [[ -z "${kernel_config}" && -n "${kernel}" ]]; then
	candidate="${kernel%-bzImage}.config"
	if [[ -r "${candidate}" ]]; then
		kernel_config="${candidate}"
	fi
fi
if [[ -n "${kernel_config}" ]]; then
	if [[ ! -r "${kernel_config}" ]]; then
		printf 'kernel config is not readable: %s\n' "${kernel_config}" >&2
		exit 2
	fi
	"${repo_root}/scripts/check-kernel-config.sh" "${kernel_config}"
else
	printf 'warning: no kernel config supplied; kboot metadata prerequisites were not prevalidated\n' >&2
fi

extra_dir="${workdir}/extra"
mkdir -p "${extra_dir}/bin" "${extra_dir}/boot" "${extra_dir}/mnt/freebsd"
install -m 0555 "${loader}" "${extra_dir}/bin/loader.kboot"
install -m 0444 "${loader_help}" "${extra_dir}/boot/loader.help.kboot"
if [[ -n "${payload_root}" ]]; then
	cp -a "${payload_root}/." "${extra_dir}/mnt/freebsd/"
fi

initramfs="${workdir}/bootup-freebsd-kboot-initramfs.cpio"
initramfs_zst="${initramfs}.zst"
loader_args=(
	hostfs_root=/mnt/freebsd
	bootdev=host:/
	boot_serial=YES
	boot_multicons=YES
	boot_verbose=YES
	autoboot_delay=0
	beastie_disable=YES
)
if [[ -n "${BOOTUP_FREEBSD_KBOOT_EXTRA_LOADER_ARGS:-}" ]]; then
	read -r -a extra_loader_args <<<"${BOOTUP_FREEBSD_KBOOT_EXTRA_LOADER_ARGS}"
	loader_args+=("${extra_loader_args[@]}")
fi
loader_cmd="/bin/loader.kboot ${loader_args[*]}"
if [[ -n "${payload_root}" ]]; then
	uinitcmd="gosh -c 'echo bootup FreeBSD kboot smoke; echo using embedded FreeBSD payload at /mnt/freebsd; echo running ${loader_cmd}; ${loader_cmd}'"
else
	uinitcmd="gosh -c 'echo bootup FreeBSD kboot smoke; echo mounting FreeBSD ISO from /dev/vda; mount -t iso9660 -o ro /dev/vda /mnt/freebsd; echo running ${loader_cmd}; ${loader_cmd}'"
fi

BOOTUP_INITRAMFS_ZSTD="${initramfs_zst}" \
	"${repo_root}/scripts/build-initramfs.sh" \
	"${initramfs}" \
	"${uinitcmd}" \
	"${BOOTUP_FREEBSD_KBOOT_GO_TAGS:-}" \
	"${extra_dir}:/"

bootup_iso="${workdir}/bootup-freebsd-kboot.iso"
BOOTUP_ISO_INITRAMFS="${initramfs_zst}" \
BOOTUP_ISO_CMDLINE="${BOOTUP_FREEBSD_KBOOT_CMDLINE:-console=ttyS0,115200n8 panic=30}" \
	"${repo_root}/scripts/build-iso.sh" "${bootup_iso}"

qemu="${BOOTUP_QEMU:-qemu-system-x86_64}"
require_cmd "${qemu%% *}"
require_cmd timeout

ovmf_code="${BOOTUP_FREEBSD_KBOOT_OVMF_CODE:-${BOOTUP_QEMU_FIRMWARE:-/usr/share/OVMF/OVMF_CODE_4M.fd}}"
ovmf_vars="${BOOTUP_FREEBSD_KBOOT_OVMF_VARS:-${BOOTUP_QEMU_FIRMWARE_VARS:-${ovmf_code/CODE/VARS}}}"
if [[ ! -r "${ovmf_code}" ]]; then
	printf 'OVMF CODE image is not readable: %s\n' "${ovmf_code}" >&2
	exit 2
fi
if [[ ! -r "${ovmf_vars}" ]]; then
	printf 'OVMF VARS image is not readable: %s\n' "${ovmf_vars}" >&2
	exit 2
fi
ovmf_vars_copy="${workdir}/OVMF_VARS.fd"
cp "${ovmf_vars}" "${ovmf_vars_copy}"

log="${BOOTUP_FREEBSD_KBOOT_LOG:-${workdir}/qemu.log}"
timeout_s="${BOOTUP_FREEBSD_KBOOT_TIMEOUT:-180}"
qemu_args=(
	-m "${BOOTUP_MEMORY:-2048}"
	-nographic
	-no-reboot
	-drive "if=pflash,format=raw,unit=0,readonly=on,file=${ovmf_code}"
	-drive "if=pflash,format=raw,unit=1,file=${ovmf_vars_copy}"
	-drive "if=none,id=bootupcd,media=cdrom,readonly=on,file=${bootup_iso}"
	-device "ide-cd,drive=bootupcd,bootindex=1"
)
if [[ -z "${payload_root}" ]]; then
	qemu_args+=(
		-drive "if=none,id=freebsddisk,format=raw,readonly=on,file=${freebsd_iso}"
		-device "virtio-blk-pci,drive=freebsddisk"
	)
fi

printf 'workdir: %s\n' "${workdir}"
printf 'serial log: %s\n' "${log}"

set +e
timeout "${timeout_s}" ${qemu} "${qemu_args[@]}" 2>&1 | tee "${log}"
qemu_status="${PIPESTATUS[0]}"
set -e

after_start="${workdir}/qemu-after-kernel-start.log"
awk 'seen { print } /Start @/ { seen = 1 }' "${log}" >"${after_start}"
target_pattern="${BOOTUP_FREEBSD_KBOOT_TARGET_PATTERN:-Welcome to FreeBSD|bsdinstall|login:|root@}"

if grep -Eq "${target_pattern}" "${after_start}"; then
	printf 'FreeBSD kboot smoke reached target marker after kernel jump.\n'
	exit 0
fi
if grep -Eq "Can't find symbol boot_params|Can't get UEFI memory map" "${log}"; then
	printf 'FreeBSD kboot smoke hit the Linux boot metadata blocker.\n' >&2
	exit 20
fi
if grep -Eq "UEFI SYSTAB PA:|UEFI MMAP:" "${log}" || grep -q "Start @" "${log}"; then
	printf 'FreeBSD kboot smoke cleared the old metadata blocker but did not reach target marker.\n' >&2
	exit 21
fi
if grep -Eq "FreeBSD/amd64 kboot loader|Welcome to FreeBSD" "${log}"; then
	printf 'FreeBSD kboot smoke reached the loader menu but did not start the kernel.\n' >&2
	exit 22
fi
if [[ "${qemu_status}" -eq 124 ]]; then
	printf 'FreeBSD kboot smoke timed out before a known marker.\n' >&2
	exit 23
fi
if [[ "${qemu_status}" -ne 0 ]]; then
	printf 'FreeBSD kboot smoke QEMU exited with status %s before a known marker.\n' "${qemu_status}" >&2
	exit "${qemu_status}"
fi

printf 'FreeBSD kboot smoke ended before a known marker.\n' >&2
exit 24
