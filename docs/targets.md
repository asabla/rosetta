# Target support

Rosetta emits configuration that targets the current documented formats. Target configuration can merge with settings from other scopes, so deployment must account for the target's precedence rules. Repository-level files are not a substitute for centrally managed policy when users can override them.

| Capability | OpenShell | OpenCode | Codex | Claude Code |
| --- | --- | --- | --- | --- |
| Filesystem read/write | Sandbox `filesystem_policy` | `read`, `edit`, and `external_directory` permissions | Named permission-profile filesystem rules | Permission rules and the Bash sandbox filesystem |
| Tool use | Not represented | Tool permission keys | MCP `enabled_tools` per server | Canonical tool or MCP permission rules |
| Command execution | Not represented | Granular `bash` rules | Not represented because Codex command allow rules authorize execution outside the sandbox | Not represented without an explicit runtime baseline |
| Network access | Gateway-enforced `network_policies` | REST read-only `webfetch` rules when port, path, and executable constraints are representable | Not represented in v0.5 because a complete profile requires deployment-specific proxy configuration | Not represented without an explicit runtime baseline |

OpenShell output is schema version 1 YAML. Filesystem paths must be absolute, root write access is rejected, Landlock is fixed to `hard_requirement`, the process cannot run as root, and every rendered endpoint uses `enforcement: enforce`. `includeWorkdir` is rejected because it would grant uncatalogued write access. REST, WebSocket, and GraphQL access presets are supported. MCP and JSON-RPC require explicit method or tool rules that the v0.5 catalog does not yet model, so compilation rejects them.

OpenCode output is `opencode.json`. It begins from a global deny and places catch-all rules before specific allows, matching OpenCode's last-match-wins behavior. Absolute and home-relative filesystem selectors also generate `external_directory` access.

Codex output is `.codex/config.toml` using the beta named permission-profile format. The profile starts from a restricted custom filesystem policy, grants only minimal runtime reads, denies the workspace root, and adds specific read or write selectors. Tool capabilities require an MCP server identifier and a matching stdio or HTTP transport definition in `options.codex.mcpServers`; they become that server's `enabled_tools` allowlist. Secrets are referenced by environment-variable name and never embedded. Rosetta intentionally does not emit command `allow` rules because those rules authorize matching commands outside the sandbox.

Claude Code output is `.claude/settings.json`. Uncatalogued tools are auto-denied with `permissions.defaultMode: dontAsk`; automatic and bypass modes are disabled. The Bash sandbox denies reads from the filesystem root and reopens only Cedar-authorized paths. Command and network capabilities remain unrepresentable until Rosetta can include a reviewable runtime baseline, so strict mode rejects them and permissive mode omits them with diagnostics.
