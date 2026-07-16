# Rosetta agent guidance

Rosetta is intended to validate Cedar policies and translate supported policy semantics into target-specific artifacts, beginning with OpenShell. Treat it as security-sensitive infrastructure: correctness, fail-closed behavior, maintainability, and stable contracts take priority over expedient changes.

The repository is currently an early scaffold. The shared implementation only checks that source text is non-empty and prefixes it with a target comment; it does not yet parse Cedar, validate against a Cedar schema, or render an OpenShell policy. Preserve this distinction in code, documentation, examples, and review summaries. Never describe an unimplemented guarantee as operational.

## Repository map

`cmd/rosetta` contains the CLI and `cmd/rosetta-server` contains the HTTP service entry point. Shared request, result, diagnostic, artifact, and compilation behavior belongs in `internal/rosetta`. HTTP routing and the OpenAPI document live in `internal/service`; `internal/api` provides service-facing aliases. `internal/compiler` is a compatibility wrapper and must not become a second source of policy semantics.

Keep Cedar parsing, normalized policy semantics, representability analysis, and target rendering separate as those layers are introduced. Avoid target-specific behavior in transport handlers. Prefer a maintained Cedar implementation over a home-grown parser, and record consequential architectural choices in documentation.

## Safety and compatibility

Generated policy must never grant access that the source policy does not grant. Strict mode must reject unsupported, ambiguous, lossy, or access-broadening translations. Permissive mode may produce a safe narrowing with explicit diagnostics, but it must never broaden access. Preserve Cedar default-deny and forbid behavior, produce deterministic artifacts, and test these properties with executable conformance cases.

Treat policy source, schemas, entities, diagnostics, and generated artifacts as potentially sensitive. Do not log their contents by default or expose them through errors, telemetry, fixtures, or examples without an explicit and reviewed reason.

Treat diagnostic codes, JSON field names, endpoint paths, artifact metadata, CLI output, and compilation semantics as compatibility surfaces. Keep runtime types, OpenAPI schemas, examples, and tests synchronized. Do not advertise targets or options that the implementation rejects.

## Engineering approach

Prefer focused changes that solve the underlying problem and leave clear extension points. Do not add speculative abstractions, duplicate policy logic, or retain compatibility layers without a concrete consumer. Keep dependencies deliberate and favor well-maintained upstream libraries for security-critical parsing, validation, and serialization.

Use prose by default in documentation, comments, commit messages, and pull request descriptions. Use a list when it communicates a genuine sequence or set more clearly than prose; do not turn every explanation into bullets. Comments should explain invariants, tradeoffs, or non-obvious reasoning rather than restate code.

Write pull request descriptions as direct, concise prose for human reviewers. Center the explanation on why the change is needed. Do not restate the diff or enumerate ordinary implementation details; mention unusual choices, pivotal decisions, or reviewer caveats only when they are not self-evident. Avoid template headings such as `Summary`, `Rationale`, or `Validation` when they merely label obvious paragraphs. Do not include iteration scores or ad hoc validation narratives. Let configured CI and repository checks communicate verification; when those checks do not exist, do not portray manual claims as repository validation.

Do not provide time estimates. Describe scope, dependencies, risks, and verification evidence instead. Prefer long-term stability and maintainability when choosing between a temporary shortcut and a durable solution.

## Verification

Format changed Go files and run the checks relevant to the change. The complete local baseline is:

```sh
test -z "$(gofmt -l .)"
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/...
```

Add or update tests whenever behavior changes. Translation work requires positive and negative cases, deterministic golden artifacts, unsupported-input diagnostics, and a regression test for every corrected security or compatibility defect. Do not claim a check passed unless it was run; report any check that could not run and why.

Before finishing, review the entire diff for accidental access broadening, contract drift, duplicated semantics, stale documentation, unnecessary dependencies, and unrelated changes. A change is complete only when behavior, tests, public contracts, and documentation agree.

## Maintaining this file

Use the repository skill at `.agents/skills/update-agents-md/SKILL.md` when creating or substantially revising agent guidance. Keep this file concise and repository-specific. Add durable instructions only after verifying them against the current tree and toolchain; remove obsolete guidance when the repository changes. Put detailed procedures in focused documentation or skills and link to them instead of expanding this file indefinitely.
