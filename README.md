# Rosetta

Rosetta compiles Cedar authorization policy into restrictive configuration for OpenShell, OpenCode, Codex, and Claude Code. Version 0.5 provides a Go SDK, a standalone CLI, and a stateless HTTP API. The CLI links the SDK directly; it never needs the service to validate or generate policy files.

Rosetta evaluates Cedar against a finite capability catalog. This makes the compilation boundary explicit and reviewable: Cedar decides whether a principal may read a path, use a tool, execute a command, or connect to an endpoint, while each renderer maps only representable decisions into its target format. Missing permits and Cedar forbids are denied. Evaluation errors fail compilation.

## Install

Build the CLI or service with Go 1.25.12 or a newer supported release:

```sh
go install github.com/asabla/rosetta/cmd/rosetta@v0.5.0
go install github.com/asabla/rosetta/cmd/rosetta-server@v0.5.0
```

Release pages provide platform archives and checksums. The service container is published as `ghcr.io/asabla/rosetta:v0.5.0` after a version tag is released.

## Compile a policy

The repository includes a complete [Cedar policy](examples/developer.cedar) and [capability catalog](examples/catalog.json). Compile any supported target without running the API:

```sh
rosetta check < examples/developer.cedar
rosetta compile --target openshell --catalog examples/catalog.json < examples/developer.cedar
rosetta compile --target opencode --catalog examples/catalog.json < examples/developer.cedar
rosetta compile --target codex --catalog examples/catalog.json --options examples/options.json < examples/developer.cedar
rosetta compile --target claude-code --mode permissive --catalog examples/catalog.json < examples/developer.cedar
```

Strict mode is the default and rejects allowed capabilities the target cannot represent without broadening access. `--mode permissive` may omit such a capability only when omission is a safe deny. The CLI writes every warning to stderr; SDK and HTTP callers receive the same diagnostics in the result.

## Go SDK

The root module is the public SDK:

```go
result, err := rosetta.Compile(ctx, rosetta.CompileRequest{
    Source:  policy,
    Target:  rosetta.TargetOpenCode,
    Catalog: catalog,
})
if err != nil {
    return err
}
fmt.Print(result.Artifacts[0].Content)
```

`Check` parses and schema-validates Cedar without a catalog. `Explain` compiles the same request and returns the capability decisions behind the artifact. `Targets` and `Capabilities` expose stable discovery metadata. Target rendering options are accepted through `--options`; the example uses them to define the transport for a Cedar-restricted Codex MCP server without embedding credentials.

## HTTP API

Run `rosetta-server` or the container and send the same SDK request model to `POST /v1/compile`. The service also exposes `POST /v1/check`, `POST /v1/explain`, discovery endpoints, a health endpoint, and `/v1/openapi.json`. Request bodies are limited to 2 MiB and unknown JSON fields are rejected.

The [architecture](docs/architecture.md), [Cedar profile](docs/cedar-profile.md), [target support](docs/targets.md), executable [target contracts](docs/target-contracts.md), and [security model](docs/security.md) document the compatibility and trust boundaries.
