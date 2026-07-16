# Security model

Rosetta is a policy compiler, not an enforcement point. Its responsibility ends after producing deterministic target configuration that does not grant more than the evaluated Cedar decisions. The target runtime, its configuration precedence, the operating system sandbox, and deployment controls remain part of the enforcement boundary.

Compilation fails closed. Cedar parse, schema, entity, request, and evaluation errors stop generation. A forbid overrides every permit, and the absence of a matching permit is a deny. Strict mode rejects a target mapping when it cannot preserve every security-relevant field of an allowed capability. Permissive mode can only remove access and reports every omitted allowed capability.

Allowed and denied wildcard capabilities are checked for language intersection before rendering. The check has a deterministic work limit and honors request cancellation; compilation fails without an artifact when Rosetta cannot prove decision isolation within that limit.

Target options cannot add authorization. OpenShell `includeWorkdir` and `best_effort` Landlock operation are rejected because they can introduce access or continued execution that is absent from the Cedar decisions.

The catalog is trusted input and the completeness boundary. Rosetta cannot infer all possible paths or tools from arbitrary Cedar. Review catalog changes with the same care as policy changes, pin the catalog version, and keep target-specific entries scoped with `targets` when one selector cannot be shared safely.

Generated project configuration may be widened by higher-precedence user or command-line configuration in some runtimes. Use managed settings or an outer sandbox when policy must be non-bypassable. In particular, Codex named permission profiles are beta, Claude project settings are below managed and command-line scopes, and OpenCode permissions can be changed by other configuration layers.

The service does not log Cedar source, catalogs, decisions, or generated artifacts. Operators should apply authentication, TLS, rate limiting, and request logging redaction at the deployment edge. The built-in body limit protects memory but is not a substitute for those controls.

Please report suspected vulnerabilities privately through GitHub's security advisory workflow rather than a public issue.
