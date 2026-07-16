# Releasing

Rosetta follows Semantic Versioning. Public SDK types, JSON fields, CLI behavior, target identifiers, diagnostic codes, artifact paths, and compilation semantics are compatibility surfaces.

A release is created from a signed or reviewed `vMAJOR.MINOR.PATCH` tag whose value matches `rosetta.Version` and the changelog. Release binaries and containers use the patched Go toolchain declared in `go.mod`; do not lower that floor without a vulnerability review. The release workflow tests the repository, builds platform archives with GoReleaser, publishes checksums and provenance, and publishes the service image to GitHub Container Registry. The Go SDK is distributed by the same Git tag through the Go module ecosystem.

Pre-1.0 minor versions may intentionally revise public contracts, but such changes still require a migration note. Patch releases must remain backward compatible and must not broaden generated access.
