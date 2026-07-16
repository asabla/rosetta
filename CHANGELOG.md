# Changelog

Rosetta follows [Semantic Versioning](https://semver.org/).

## Unreleased

Rosetta now validates complete target capability semantics before rendering. Unsafe OpenShell options are rejected, Codex profiles no longer inherit broad read access, Claude Code command and network mappings fail closed until an explicit runtime baseline is available, and permissive CLI compilation reports every diagnostic on stderr.

Generated artifacts are checked against narrow target contracts before being returned. Process-level end-to-end coverage now compiles OpenShell, OpenCode, Codex, and Claude Code through both the CLI and HTTP service.

Compile and explain results now include deterministic provenance metadata with compiler, catalog, and target-contract versions plus input and artifact SHA-256 digests. Target contract versions and maturity are discoverable through the SDK and API, CLI policy operations support structured JSON output, CLI inputs are bounded, and container publication waits for the full source and binary release gate.

## 0.5.0 - 2026-07-16

Rosetta now provides a production-oriented Go SDK, standalone CLI, and HTTP API backed by Cedar-Go parsing, schema validation, and authorization. The versioned capability catalog compiles to deterministic OpenShell, OpenCode, Codex, and Claude Code configuration with fail-closed representability checks.

The release adds Cedar forbid and default-deny preservation tests, target contract tests, fuzz targets, CLI and API equivalence coverage, container packaging, automated releases, and security and architecture documentation.
