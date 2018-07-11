#!/bin/bash

# build binary distributions for linux/amd64 and darwin/amd64
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "working dir $DIR"

version='0.0.3'
go_arch=$(go env GOARCH)
go_os=$(go env GOOS)
go_version=$(go version | awk '{print $3}')

echo "[INFO] Starting $(basename "${0}") v${version}"
for os in linux darwin; do
    echo "[INFO]   building for $os/$go_arch"
    BUILD="$(mktemp -d -t mapnificent_generator_XXXXXX)"
    TARGET="mapnificent_generator-$version.$os-$go_arch.$go_version"
    GOOS=$os GOARCH=$go_arch CGO_ENABLED=0 go build -o $BUILD/$TARGET/mapnificent_generator
    mkdir -p $BUILD/$TARGET
    if [ "$os" = "$go_os" ]; then
    	echo "[INFO]     copying mapnificent_generator for $os/$go_arch"
        cp $BUILD/$TARGET/mapnificent_generator mapnificent_generator
    fi
    pushd $BUILD >/dev/null
    tar czf $TARGET.tar.gz $TARGET >/dev/null
    if [ -e "$DIR/dist/$TARGET.tar.gz" ]; then
        echo "[WARN]     overwriting dist/$TARGET.tar.gz"
    fi
    mkdir -p "${DIR}/dist"
    mv $TARGET.tar.gz $DIR/dist
    echo "[INFO]     built dist/$TARGET.tar.gz"
    popd >/dev/null
done
