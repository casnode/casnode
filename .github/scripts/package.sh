#!/usr/bin/env bash
# package.sh <tag>: builds the backend for each platform and packs it with conf/, the built frontend (web/build) and
# the license into dist/casnode_<tag>_<os>_<arch>.tar.gz (.zip for Windows), the files of a GitHub release.
set -euo pipefail

tag="$1"
if [ ! -f web/build/server.txt ]; then
  echo "web/build has no built frontend, run yarn build in web/ first" >&2
  exit 1
fi

rm -rf dist
mkdir -p dist
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  goos="${target%/*}"
  goarch="${target#*/}"
  name="casnode_${tag}_${goos}_${goarch}"
  stage="dist/$name"
  exe="casnode"
  if [ "$goos" = "windows" ]; then
    exe="casnode.exe"
  fi

  mkdir -p "$stage/conf" "$stage/web"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$stage/$exe" .
  cp conf/app.conf "$stage/conf/"
  cp -r web/build "$stage/web/"
  cp LICENSE README.md "$stage/"

  if [ "$goos" = "windows" ]; then
    (cd dist && zip -qr "$name.zip" "$name")
  else
    tar -czf "dist/$name.tar.gz" -C dist "$name"
  fi
  rm -rf "$stage"
  echo "packed $name"
done

(cd dist && sha256sum casnode_* > checksums.txt)
