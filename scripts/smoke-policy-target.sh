#!/usr/bin/env bash
set -euo pipefail

if [[ "${BOOTUP_POLICY_SMOKE:-}" != "1" ]]; then
	echo "BOOTUP_POLICY_SMOKE=1 is required" >&2
	exit 2
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/.." && pwd)"
cd "${repo_root}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

private_key="${tmpdir}/policy.key"
public_key="${tmpdir}/policy.pub"
policy_json="${tmpdir}/policy.json"
policy_sig="${tmpdir}/policy.json.sig"
bad_policy_json="${tmpdir}/bad-policy.json"
bad_policy_sig="${tmpdir}/bad-policy.json.sig"
diagnostics_dir="${tmpdir}/diagnostics"

go run ./cmd/bootup-policy-sign \
	--generate-key \
	--private-key "${private_key}" \
	--public-key "${public_key}" >/dev/null

cat >"${policy_json}" <<'JSON'
{
  "schema_version": 1,
  "decision_id": "policy-smoke-opensuse",
  "target_id": "opensuse-leap-160-amd64-netboot",
  "options": {
    "text-install": "true",
    "mirror-url": "https://mirror.example/opensuse"
  },
  "expires_at": "2099-01-01T00:00:00Z"
}
JSON

go run ./cmd/bootup-policy-sign \
	--policy "${policy_json}" \
	--private-key "${private_key}" \
	--signature "${policy_sig}" >/dev/null

plan_output="$(go run ./cmd/bootup \
	--mode=policy-target \
	--policy-file "${policy_json}" \
	--policy-signature "${policy_sig}" \
	--policy-public-key "${public_key}")"

require_contains() {
	local haystack="${1}"
	local needle="${2}"
	if [[ "${haystack}" != *"${needle}"* ]]; then
		echo "policy smoke output is missing: ${needle}" >&2
		echo "${haystack}" >&2
		exit 1
	fi
}

require_contains "${plan_output}" "[planning] openSUSE Leap 16.0 amd64 installer"
require_contains "${plan_output}" "textmode=1"
require_contains "${plan_output}" "install=https://mirror.example/opensuse"

cat >"${bad_policy_json}" <<'JSON'
{
  "schema_version": 1,
  "decision_id": "policy-smoke-invalid-option",
  "target_id": "opensuse-leap-160-amd64-netboot",
  "options": {
    "unsupported-option": "true"
  },
  "expires_at": "2099-01-01T00:00:00Z"
}
JSON

go run ./cmd/bootup-policy-sign \
	--policy "${bad_policy_json}" \
	--private-key "${private_key}" \
	--signature "${bad_policy_sig}" >/dev/null

if go run ./cmd/bootup \
	--diagnostics-dir "${diagnostics_dir}" \
	--mode=policy-target \
	--policy-file "${bad_policy_json}" \
	--policy-signature "${bad_policy_sig}" \
	--policy-public-key "${public_key}" >"${tmpdir}/diagnostics.stdout" 2>"${tmpdir}/diagnostics.stderr"; then
	echo "policy smoke expected diagnostics run to fail" >&2
	exit 1
fi

summary_path="$(find "${diagnostics_dir}" -name summary.json -type f -print -quit)"
if [[ -z "${summary_path}" ]]; then
	echo "policy smoke did not write a diagnostics summary" >&2
	exit 1
fi

grep -Fq '"source": "local"' "${summary_path}"
grep -Fq '"ed25519": true' "${summary_path}"
grep -Fq 'unsupported-option' "${summary_path}"
if grep -Fq "${private_key}" "${summary_path}" || grep -Fq "${public_key}" "${summary_path}" || grep -Fq "${bad_policy_sig}" "${summary_path}"; then
	echo "policy diagnostics summary exposed policy path material" >&2
	exit 1
fi

echo "policy smoke passed"
