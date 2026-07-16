---
name: update-agents-md
description: Review, create, or update repository AGENTS.md guidance while keeping it accurate, concise, durable, and compatible with Codex and OpenCode. Use when agent instructions are missing or stale, repository architecture or commands change, recurring agent mistakes reveal a guidance gap, or a user asks to improve AGENTS.md.
---

# Update AGENTS.md

Maintain agent guidance as an accurate operating contract for the repository. Prefer the smallest change that resolves a durable guidance gap.

## Establish scope and evidence

Read the root `AGENTS.md` and any nested `AGENTS.md` files that govern the affected area. Inspect the repository tree, build manifests, CI configuration, README, contributing guidance, architecture documentation, and the commands used by the current toolchain. Review relevant recent changes when they explain why guidance has become stale.

Treat root guidance as repository-wide and nested guidance as an override for its subtree. Before adding a nested file, confirm that the local rule genuinely differs; do not repeat root instructions merely to make them visible nearby.

Distinguish verified repository facts from proposed future state. Do not add a command, path, capability, or guarantee unless it exists and has been checked. If the repository is an early scaffold, state its limitations plainly.

Classify each proposed instruction before editing:

- Put durable repository conventions, architecture boundaries, verification commands, safety invariants, and completion criteria in `AGENTS.md`.
- Keep one-off task requests and personal preferences out unless the repository has explicitly adopted them as team policy.
- Prefer tests, linters, CI, or hooks for mechanically enforceable rules; describe the enforcement in `AGENTS.md` only when agents need it to work correctly.
- Put detailed repeatable procedures in a focused skill or document and link to them instead of growing `AGENTS.md` indefinitely.

## Edit for both harnesses

Keep repository skills in `.agents/skills`, the shared discovery location supported by Codex and OpenCode. Do not duplicate the same skill under `.opencode`, `.codex`, or another tool-specific directory.

Use standard Markdown and repository-relative paths. Keep shared instructions tool-neutral so Codex and OpenCode receive the same guidance. Do not require a harness-specific command, invocation syntax, configuration file, or tool unless the repository actually depends on it; isolate unavoidable tool-specific guidance and label it clearly.

Write concise prose by default. Use lists for real sequences, checklists, or exact sets, not as the default form for every paragraph. Preserve useful existing instructions and terminology. Remove duplication, vague advice, stale commands, and generic statements that do not change agent behavior.

Link to the repository's canonical architecture, contribution, or operations document when detail already lives there. Summarize only the instruction an agent must know before acting; avoid copying volatile reference material into `AGENTS.md`.

Do not weaken security, correctness, compatibility, or verification requirements. Do not add time estimates. When instructions conflict, resolve the conflict in favor of the closest applicable file and make the scope explicit.

## Review in five passes

For a substantive change, review the complete candidate five times and retain only improvements:

1. Verify factual accuracy against the current repository and runnable commands.
2. Check scope and precedence, including nested instruction files and linked guidance.
3. Test whether the guidance promotes maintainable architecture and durable solutions without prescribing speculative design.
4. Check security, compatibility, and definition-of-done requirements for accidental weakening or ambiguity.
5. Remove repetition, unnecessary lists, harness-specific assumptions, and wording that will become stale quickly.

After each pass, record the defect found, the improvement made, and a concise quality score. A pass that finds no genuine improvement should leave the file unchanged rather than rewrite it for novelty.

## Validate and hand off

Read the final file from top to bottom as a new agent would. Verify referenced paths and run safe documented commands when the change depends on them. Check that the skill remains aligned if the maintenance procedure itself changed.

Summarize what changed, why it is durable, and what was validated. Report commands that were not run and the reason. Do not claim that guidance is enforced unless a corresponding test, CI check, hook, or repository rule provides that enforcement.
