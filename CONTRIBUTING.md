# Contributing

Rosetta changes must preserve Cedar default-deny and forbid semantics and must not broaden target access. Start with an issue for public contract or target-format changes so the compatibility boundary is explicit before implementation.

Use Conventional Commits and keep changes focused. Update executable contract tests and documentation together when behavior changes. Run `make check` before opening a pull request. Renderer changes need an allowed case, a denied case, an unsupported case, and deterministic output coverage.

Target behavior should be grounded in current upstream documentation. Prefer a maintained parser or serializer over custom security-sensitive parsing, and explain any unavoidable approximation in the target documentation.
