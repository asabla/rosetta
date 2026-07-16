# Migrating to v1

Rosetta v1 establishes the first stable SDK, CLI, HTTP, catalog, and generated-artifact compatibility boundary. Existing 0.5 integrations should make the following changes before upgrading.

## Catalog version

Set every catalog's `version` field to `rosetta/v1`. The compiler rejects older catalog identifiers so that callers cannot silently use pre-v1 semantics.

## Target options

Remove `workspaceRoot` from Codex options. A caller-provided root could activate permissions outside the catalog's authorization decision, so v1 never emits Codex `workspace_roots`.

Remove `includeWorkdir` and `landlockCompatibility` from OpenShell options. Rosetta v1 always emits `include_workdir: false` and `landlock_compatibility: hard_requirement`; these fail-closed settings are no longer configurable.

The CLI and HTTP API reject all three removed fields as unknown JSON input.

## Decision isolation

Allowed and denied command or network globs must be disjoint. Rosetta v1 checks exact intersections for `*` and `?`, including crossing wildcard patterns, before rendering. Validation has a deterministic work limit and fails closed when that limit is exceeded.

## Codex artifact contract

The Codex target contract identifier is `rosetta/codex-permissions-v1beta2`. Codex remains preview in v1, and generated artifacts explicitly reject activated `workspace_roots`.
