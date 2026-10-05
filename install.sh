#!/bin/sh
# GitShiny installer
#   curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | sh
#
# Environment overrides:
#   GITSHINY_VERSION      install a specific tag (default: latest release)
#   GITSHINY_INSTALL_DIR  install location (default: ~/.local/bin)
#   GITSHINY_REPO         owner/repo (default: sun01822/gitshiny)
#   GITSHINY_BASE_URL     release download base URL (mirrors / testing)
set -eu

REPO="${GITSHINY_REPO:-sun01822/gitshiny}"
INSTALL_DIR="${GITSHINY_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${GITSHINY_VERSION:-}"
BASE_URL="${GITSHINY_BASE_URL:-https://github.com/$REPO/releases/download}"

ok()  { printf '✓ %s\n' "$*"; }
die() { printf 'Error: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed"; }

need curl; need tar; need uname; need awk

echo "Installing GitShiny..."
echo

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *) die "unsupported OS '$os'. On Windows, run in PowerShell: irm https://raw.githubusercontent.com/$REPO/main/install.ps1 | iex" ;;
esac
ok "OS detected: $os"

arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "unsupported architecture '$arch'" ;;
esac
ok "Architecture detected: $arch"

if [ -z "$VERSION" ]; then
  final=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest") \
    || die "could not reach GitHub to find the latest release"
  case "$final" in
    */releases/tag/*) VERSION=${final##*/} ;;
    *) die "no release found for $REPO yet. Try: go install github.com/$REPO@latest" ;;
  esac
fi
case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

file="gitshiny_${os}_${arch}.tar.gz"
curl -fsSL "$BASE_URL/$VERSION/$file" -o "$tmp/$file" \
  || die "download failed: $BASE_URL/$VERSION/$file"
curl -fsSL "$BASE_URL/$VERSION/checksums.txt" -o "$tmp/checksums.txt" \
  || die "could not download checksums.txt"
ok "Downloaded GitShiny $VERSION"

expected=$(awk -v f="$file" '$2==f || $2=="*" f {print $1}' "$tmp/checksums.txt")
[ -n "$expected" ] || die "no checksum listed for $file"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/$file" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmp/$file" | awk '{print $1}')
else
  die "sha256sum or shasum is required to verify the download"
fi
[ "$expected" = "$actual" ] || die "checksum mismatch for $file (expected $expected, got $actual)"
ok "Checksum verified"

tar -xzf "$tmp/$file" -C "$tmp" gitshiny
mkdir -p "$INSTALL_DIR"
install -m 755 "$tmp/gitshiny" "$INSTALL_DIR/gitshiny"
ok "Installed $INSTALL_DIR/gitshiny"

echo
echo "GitShiny installed successfully."
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo
     echo "Add it to your PATH (Zsh/Bash):"
     echo "    export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac
echo
echo "Run:"
echo
echo "    gitshiny"
