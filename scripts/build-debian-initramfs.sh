#!/usr/bin/env bash
set -euo pipefail

keyring="${1:-/usr/share/keyrings/debian-archive-keyring.gpg}"
out="${2:-dist/bootup-debian-initramfs.cpio}"
config_path="/etc/bootup/providers.json"
trust_path="/etc/bootup/trust/debian-archive-keyring.gpg"
uinitcmd="${3:-bootup --mode=menu --prepare-runtime --provider-config=${config_path}}"
extra_files="${4:-}"

if [[ ! -r "${keyring}" ]]; then
	echo "keyring is not readable: ${keyring}" >&2
	exit 1
fi

tmpdir="$(mktemp -d)"
cleanup() {
	rm -rf "${tmpdir}"
}
trap cleanup EXIT

provider_config="${tmpdir}/providers.json"
cat >"${provider_config}" <<EOF
{
  "providers": {
    "debian": {
      "keyring_path": "${trust_path}"
    }
  }
}
EOF

provider_files="${provider_config}:${config_path},${keyring}:${trust_path}"
if [[ -n "${extra_files}" ]]; then
	provider_files="${provider_files},${extra_files}"
fi

GOCACHE="${GOCACHE:-/tmp/bootup-go-build-cache}" scripts/build-initramfs.sh "${out}" "${uinitcmd}" "" "${provider_files}"
