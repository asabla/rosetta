#!/bin/sh
set -eu

bad=$(grep -En 'uses: [^[:space:]]+@' .github/workflows/*.yml | grep -Ev '@[0-9a-f]{40}([[:space:]]|$)' || true)
if [ -n "$bad" ]; then
	echo "workflow actions must be pinned to full commit SHAs:" >&2
	echo "$bad" >&2
	exit 1
fi
