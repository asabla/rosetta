# Target contracts

Rosetta validates every generated artifact against a deliberately narrow target contract before returning it. These checks are defense in depth: they do not replace upstream parsers or behavioral end-to-end tests, but they prevent renderer changes from silently removing the restrictive defaults on which compilation relies.

Contract identifiers and maturity are public discovery data returned by the SDK `Capabilities` function and `GET /v1/capabilities`. OpenShell (`rosetta/openshell-policy-v1`) and OpenCode (`rosetta/opencode-permissions-v1`) are supported. Codex (`rosetta/codex-permissions-v1beta`) and Claude Code (`rosetta/claude-code-settings-v1beta`) are preview contracts. A preview contract is fail-closed and tested, but may change incompatibly before its first stable contract version.

The OpenShell contract covers schema version 1 policies. It requires `hard_requirement` Landlock behavior, an explicit non-root process identity, absolute filesystem and binary paths, no writable filesystem root, and `enforce` on every network endpoint. Runtime paths are not invented by Rosetta; deployment-specific paths required by an agent or container image must be explicit catalog capabilities.

The OpenCode contract covers the current `permission` object format. Generated configuration starts from a global deny, and every granular filesystem, command, or web-fetch rule includes its own catch-all deny before a specific allow. Rosetta does not treat OpenCode permission prompts as an operating-system sandbox.

The Codex contract covers beta named permission profiles. Generated profiles do not inherit `:read-only` or `:workspace`; they grant the minimal runtime baseline, deny workspace roots, then add Cedar-authorized filesystem entries. MCP servers are emitted only when an allowed tool names that server and the request supplies a complete transport definition.

The Claude Code contract covers project settings with `dontAsk` default permissions, disabled automatic and bypass modes, and a Bash sandbox that fails if enforcement is unavailable. Filesystem reads begin with a root deny and reopen selected paths. Command and network capabilities remain unsupported until Rosetta has a versioned, reviewable runtime baseline.

The references and maturity of upstream formats are tracked in [target support](targets.md). A target format change requires an updated contract check, a golden artifact, an allowed case, a denied case, and process-level CLI/API equivalence coverage. Every compile result records the selected contract identifier so generated artifacts can be audited after compiler upgrades.
