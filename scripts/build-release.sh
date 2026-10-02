#!/bin/sh
# Build release archives + checksums.txt into ./dist
# Usage: scripts/build-release.sh v1.0.0
set -eu

VERSION="${1:?usage: build-release.sh vX.Y.Z}"
cd "$(dirname "$0")/.."
rm -rf dist && mkdir dist

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
  os=${target%/*}; arch=${target#*/}
  bin=gitshiny; [ "$os" = windows ] && bin=gitshiny.exe
  stage=$(mktemp -d)
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$stage/$bin" .
  name="gitshiny_${os}_${arch}"
  if [ "$os" = windows ]; then
    (cd "$stage" && zip -q "$OLDPWD/dist/$name.zip" "$bin")
  else
    tar -czf "dist/$name.tar.gz" -C "$stage" "$bin"
  fi
  rm -rf "$stage"
  echo "built $name"
done

cd dist
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum *.tar.gz *.zip > checksums.txt
else
  shasum -a 256 *.tar.gz *.zip > checksums.txt
fi
cat checksums.txt
