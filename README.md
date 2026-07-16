# rosetta

Service layer for running Cedar policy validation and Openshell policy rendering based on it.

## Shared API models

Rosetta exposes shared API models through `internal/rosetta` and re-exports them from `internal/api` for service callers.

### Diagnostics

Diagnostics describe validation or translation messages. The `code` field is stable enough for automation and programmatic handling. The `message` field remains human-readable and may change as wording improves.

Diagnostic fields include:

* `severity`
* `code`
* `message`
* `details`
* `sourceSpan`
* `target`
* `ruleId`
* `recoverable`
* `documentationUrl`

### Artifacts

Artifacts describe generated target outputs. Artifact `content` is plain text by default when `encoding` is `plain`. Encoded payloads should set `encoding` to an explicit value such as `base64`.

Artifact fields include:

* `name`
* `pathHint`
* `mediaType`
* `target`
* `content`
* `encoding`
* `description`
