# Cedar profile

Rosetta uses the `Rosetta` Cedar namespace. A catalog principal becomes `Rosetta::Principal::<id>` and every catalog entry becomes `Rosetta::Capability::<id>`. The resource exposes `kind`, `selector`, `access`, `binaries`, `targets`, and optional network or MCP fields as Cedar attributes.

Actions are `Rosetta::Action::"read"`, `"write"`, `"use"`, `"execute"`, and `"connect"`. The action on a catalog entry must agree with its capability kind. Filesystem capabilities use read or write and their selectors name directory roots without glob syntax. Renderers add recursive target patterns where the target requires them. Tools use use, commands use execute, and network endpoints use connect.

This policy permits reading catalogued paths and use of one tool for principals with the developer role:

```cedar
permit (
    principal is Rosetta::Principal,
    action in [Rosetta::Action::"read", Rosetta::Action::"use"],
    resource is Rosetta::Capability
)
when {
    principal.roles.contains("developer") &&
    (resource.kind == "filesystem" || resource.selector == "read")
};
```

The catalog remains explicit because a static compiler cannot safely discover the universe of paths, commands, tools, or endpoints from an arbitrary Cedar expression. Adding a new capability requires a catalog change and therefore produces a reviewable diff.
