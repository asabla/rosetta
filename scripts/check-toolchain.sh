#!/bin/sh
set -eu

version=$(awk '$1 == "toolchain" { sub(/^go/, "", $2); print $2 }' go.mod)
if [ -z "$version" ]; then
	echo "go.mod must declare a toolchain version" >&2
	exit 1
fi

if ! grep -F "FROM golang:${version}-alpine AS build" Dockerfile >/dev/null; then
	echo "Dockerfile must use Go ${version} from go.mod" >&2
	exit 1
fi

for workflow in .github/workflows/ci.yml .github/workflows/release.yml; do
	if ! grep -F "go-version: '${version}'" "$workflow" >/dev/null; then
		echo "$workflow must use Go ${version} from go.mod" >&2
		exit 1
	fi
	if grep -F "go-version:" "$workflow" | grep -Fv "go-version: '${version}'" >/dev/null; then
		echo "$workflow contains a Go version that differs from ${version}" >&2
		exit 1
	fi
done
