# Architecture

Rosetta v0.5 compiles Cedar authorization decisions into restrictive configuration for agent runtimes. It does not rewrite Cedar syntax into target syntax. Such a rewrite would be misleading because Cedar is an authorization language while the targets expose different combinations of filesystem sandboxes, tool gates, command approval rules, and network policy.

Each compilation combines Cedar policy source with a finite capability catalog. Rosetta validates both against its versioned Cedar schema, authorizes every capability for one principal, normalizes the decisions, and passes only those decisions to a target renderer. Cedar forbids override permits. A missing permit is a deny. Evaluation errors stop compilation.

The catalog is the completeness boundary. A generated artifact only makes claims about catalogued capabilities. Renderers start from the target's most restrictive practical defaults, omit denied capabilities where omission means deny, and reject mappings that would broaden access. Representability is checked against every security-relevant capability field, not only its kind. Strict mode rejects unsupported capabilities. Permissive mode may omit an unsupported capability only when omission is a safe deny and emits a diagnostic.

Compilation mode never weakens Cedar parsing or schema validation. Strict and permissive compilation validate the same Cedar profile; the mode controls only whether an allowed but unrepresentable target capability may be safely omitted.

The public Go package is the SDK and owns parsing, validation, authorization, normalization, and rendering. The CLI calls that package directly and has no HTTP dependency. The HTTP service is a stateless adapter over the same package, which keeps local and remote behavior equivalent.

OpenShell sandbox policy combines host filesystem enforcement with network rules enforced through the OpenShell gateway. Rosetta therefore emits one schema-version-1 sandbox policy rather than inventing a separate Cedar-to-gateway configuration format. Gateway deployment settings such as TLS, OIDC, storage, and compute drivers are operational configuration, not authorization decisions, and remain outside this compiler.

The target contracts are based on the current official references for [OpenShell policy schema](https://docs.nvidia.com/openshell/reference/policy-schema), [OpenCode permissions](https://opencode.ai/docs/permissions/), [Codex permissions](https://developers.openai.com/codex/permissions), [Codex command rules](https://developers.openai.com/codex/rules), and [Claude Code settings](https://code.claude.com/docs/en/settings).
