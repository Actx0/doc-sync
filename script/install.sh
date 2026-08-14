#!/usr/bin/env bash
# Download the GoReleaser binary for a tagged action ref, or build from source.
set -euo pipefail

repo="${DOC_SYNC_REPO:-Actx0/doc-sync}"
ref="${DOC_SYNC_REF:-}"
action_path="${DOC_SYNC_ACTION_PATH:-.}"
dest="${DOC_SYNC_BIN:-${RUNNER_TEMP:-/tmp}/actx0-doc-sync}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${os}" in
linux*) os=linux ;;
darwin*) os=darwin ;;
mingw*|msys*|cygwin*) os=windows ;;
esac

arch="$(uname -m)"
case "${arch}" in
x86_64|amd64) arch=amd64 ;;
aarch64|arm64) arch=arm64 ;;
*)
	echo "unsupported architecture: ${arch}" >&2
	exit 1
	;;
esac

[[ "${os}" == windows && "${dest}" != *.exe ]] && dest="${dest}.exe"
mkdir -p "$(dirname "${dest}")"

tag=""
if [[ "${ref}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-].*)?$ ]]; then
	tag="${ref}"
elif [[ "${ref}" =~ ^v[0-9]+$ ]]; then
	tag="$(
		gh release list --repo "${repo}" --limit 50 --json tagName,isDraft,isPrerelease \
			--jq '.[] | select(.isDraft == false and .isPrerelease == false) | .tagName' |
			grep -E "^${ref}\.[0-9]+\.[0-9]+" | head -n 1
	)"
fi

if [[ "${ref}" =~ ^v[0-9]+(\.[0-9]+)*([.-].*)?$ ]]; then
	if [[ -z "${tag}" ]]; then
		echo "could not resolve release for ${ref}" >&2
		exit 1
	fi
	echo "installing doc-sync ${tag} (${os}/${arch})"
	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp}"' EXIT
	gh release download "${tag}" --repo "${repo}" --dir "${tmp}" \
		--pattern "doc-sync_${os}_${arch}.*" --pattern checksums.txt

	archive="$(find "${tmp}" -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) | head -n 1)"
	[[ -n "${archive}" ]] || {
		echo "no archive found for ${os}/${arch} in ${tag}" >&2
		exit 1
	}

	name="$(basename "${archive}")"
	want="$(awk -v f="${name}" '$2 == f || $2 == "*"f { print $1; exit }' "${tmp}/checksums.txt")"
	got="$(openssl dgst -sha256 "${archive}" | awk '{print $NF}')"
	[[ -n "${want}" && "${want}" == "${got}" ]] || {
		echo "checksum mismatch for ${name}" >&2
		exit 1
	}

	mkdir -p "${tmp}/out"
	case "${archive}" in
	*.zip) unzip -qo "${archive}" -d "${tmp}/out" ;;
	*) tar -xzf "${archive}" -C "${tmp}/out" ;;
	esac
	bin="$(find "${tmp}/out" -type f \( -name doc-sync -o -name doc-sync.exe \) | head -n 1)"
	[[ -n "${bin}" ]] || {
		echo "doc-sync binary missing from ${name}" >&2
		exit 1
	}
	cp "${bin}" "${dest}"
else
	echo "building doc-sync from source (${ref:-local})"
	command -v go >/dev/null || {
		echo "go is required to build doc-sync from ${ref:-source}" >&2
		exit 1
	}
	(cd "${action_path}" && go build -ldflags "-s -w" -o "${dest}" ./cmd)
fi

chmod +x "${dest}"
[[ -n "${GITHUB_OUTPUT:-}" ]] && echo "bin=${dest}" >>"${GITHUB_OUTPUT}"
echo "installed ${dest}"
