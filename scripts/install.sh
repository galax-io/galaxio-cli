#!/usr/bin/env sh
set -eu

repo="${GALAXIO_REPO:-galax-io/galaxio-cli}"
version="${GALAXIO_VERSION:-latest}"
bin_dir="${GALAXIO_BIN_DIR:-$HOME/.local/bin}"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

need curl
need shasum
need tar
need unzip

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

api="https://api.github.com/repos/$repo/releases/latest"
if [ "$version" != "latest" ]; then
  version="${version#v}"
  api="https://api.github.com/repos/$repo/releases/tags/v$version"
fi

curl_opts="--connect-timeout 10 --retry 3 --retry-delay 2 --retry-max-time 60 --retry-all-errors"

release="$tmp_dir/release.json"
curl -fsSL $curl_opts --max-time 30 -H "Accept: application/vnd.github+json" "$api" -o "$release"

tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$release" | head -1)"
version="${tag#v}"
asset="galaxio_${version}_${os}_${arch}.tar.gz"
[ "$os" = "windows" ] && asset="galaxio_${version}_${os}_${arch}.zip"

asset_url="$(sed -n "s/.*\"browser_download_url\"[[:space:]]*:[[:space:]]*\"\\([^\"]*${asset}\\)\".*/\\1/p" "$release" | head -1)"
checksums_url="$(sed -n 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*checksums.txt\)".*/\1/p' "$release" | head -1)"

if [ -z "$asset_url" ] || [ -z "$checksums_url" ]; then
  echo "release asset not found for $os/$arch in $repo $tag" >&2
  exit 1
fi

curl -fsSL $curl_opts --max-time 120 "$asset_url" -o "$tmp_dir/$asset"
curl -fsSL $curl_opts --max-time 30 "$checksums_url" -o "$tmp_dir/checksums.txt"

(cd "$tmp_dir" && grep "  $asset\$" checksums.txt | shasum -a 256 -c -)

mkdir -p "$bin_dir"
case "$asset" in
  *.zip) unzip -p "$tmp_dir/$asset" galaxio.exe > "$bin_dir/galaxio.exe"; chmod +x "$bin_dir/galaxio.exe" ;;
  *) tar -xzf "$tmp_dir/$asset" -C "$tmp_dir" galaxio; install -m 0755 "$tmp_dir/galaxio" "$bin_dir/galaxio" ;;
esac

echo "installed galaxio $version to $bin_dir"
echo "ensure PATH contains: $bin_dir"
echo "completion:"
echo "  zsh:  galaxio completion zsh > \"\${fpath[1]}/_galaxio\""
echo "  bash: galaxio completion bash > /usr/local/etc/bash_completion.d/galaxio"
