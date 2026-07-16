# Releasing

Rosetta follows Semantic Versioning. Public SDK types, JSON fields, CLI behavior, target identifiers, diagnostic codes, artifact paths, and compilation semantics are compatibility surfaces.

A release is created from a signed or reviewed `vMAJOR.MINOR.PATCH` tag whose value matches `rosetta.Version` and the changelog. Release binaries and containers use the patched Go toolchain declared in `go.mod`; do not lower that floor without a vulnerability review. Before publishing, the release workflow runs the full formatting, test, race, vet, and build gate. It then builds platform archives with GoReleaser and publishes checksums and provenance. The service image is published to GitHub Container Registry only after that source and binary release job succeeds. The Go SDK is distributed by the same Git tag through the Go module ecosystem.

Release review must verify that every target has an explicit contract identifier and maturity, that the OpenAPI document describes current response fields, and that repeated compilation of the same request produces identical metadata and artifacts. The input digest covers Cedar source bytes, target, normalized mode, catalog, and target options; the artifact digest covers the emitted file content. These digests support audit and reproducibility checks and are not signatures.

Pre-1.0 minor versions may intentionally revise public contracts, but such changes still require a migration note. Patch releases must remain backward compatible and must not broaden generated access.
