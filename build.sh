#!/bin/bash

cd "$(dirname $0)"
dir="$(pwd)"
name=$(basename "$dir")
tag=$(git describe --abbrev=0 --tags)
gover=$(go env GOVERSION)

set -e

go mod tidy

function build() {
    os=${1:-$(go env GOOS)}
    arch=${2:-$(go env GOARCH)}
    compress=${3}

    version="${tag}-${gover}-${os}-${arch}"
    echo "build $version"
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -ldflags "-X 'main.Version=$version'" -o "$name"

    if [[ "$compress" == "true" ]]; then
        mkdir -p compress
        tar -czf "compress/${name}-${version}.tar.gz" "${name}"
        rm "$name"
    fi
}

if [[ "$1" == "release" ]]; then
    build linux amd64 true
    build linux arm64 true
    build linux arm true

    build darwin amd64 true
    build darwin arm64 true
else
    build
fi
