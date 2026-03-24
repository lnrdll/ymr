# Agent Guide (ymr)

This repository builds `ymr`, a lightweight, spec-driven CLI for rendering YAML templates by applying substitutions encoded as YAML comments.

If you are an automated agent working in this repo:

- Read `README.md` for end-user usage.
- Use this file for repo navigation, invariants, and reliable commands.

## Quick Commands

- Show CLI help: `go run . --help`
- Run unit tests: `go test ./...`
- Build local binary: `go build -trimpath -o dist/ymr .`

## Product Behavior (Non-Obvious)

- Spec-less is the default: `ymr run` only loads a spec when `--spec` (`-s`) is provided.
- Spec-less mode requires `--template` (`-T`) and at least one `--param` (`-p`).
- If you output to files in spec-less mode (`-o rendered/`), pass at least one `--target` so output filenames are stable.
- Template directives are embedded in YAML comments and are removed from the output after processing.

## Repository Map

- `main.go`: program entrypoint, calls Cobra.
- `cmd/`: Cobra command definitions.
  - `cmd/run.go`: CLI flags and spec/spec-less mode gate.
  - `cmd/init.go`: generates boilerplate `spec.yaml`.
  - `cmd/version.go`: prints build info.
- `internal/app/main.go`: top-level orchestration for `ymr run`.
- `internal/spec/`: spec schema and CLI `key=value` parsing.
- `internal/source/`: loads `spec.yaml`, templates, and validations from local paths, HTTP(S), or GitHub.
- `internal/processor/`: YAML AST traversal and directive application (`from-param`, `from-param-merge`).
- `internal/validation/`: CEL-based policy checks.
- `example/`: runnable sample specs/templates.
- `scripts/`: release build scripts (cross-compile + archive + sha256).

## Core Flow (ymr run)

1. Parse CLI flags (`cmd/run.go`) into `internal/app.Config`.
2. If `--spec` is set:
   - `internal/source.NewSourceLoader()` chooses loader: local dir/file, HTTP(S), or GitHub.
   - loader reads and parses `spec.yaml` into `internal/spec.SpecConfig`.
3. Else (spec-less):
   - a minimal in-memory spec is constructed using `--template`.
4. Parameter map is built per target from `spec.yaml` (`internal/spec.BuildParamLookup`).
5. CLI overrides apply:
   - `--param key=value` is parsed with type inference (int, bool, string).
   - `--target` filters targets to render.
6. Validations:
   - CEL rules come from spec, or `--validation` overrides with an external file.
   - Rules only apply when the rule explicitly lists the current `targetId`.
7. For each template x target:
   - template content is loaded via `internal/source.LoadTemplate()`.
   - YAML is parsed to an AST and directives are applied (`internal/processor.ProcessContent`).
   - output filename is `${targetId}-${templateBaseName}${ext}`.
8. Output:
   - `-o -` prints concatenated content to stdout.
   - otherwise writes files (creates output dir if needed).

## Template Directives

`ymr` looks for these comment directives:

- `from-param: <go-template>`
- `from-param-merge: <go-template>`

Directives can appear as line comments on mapping keys, mapping values, or sequence items.

Rendering rules:

- Simple form `{{ .key }}` preserves types (lists/maps) when present in params.
- Missing keys or template errors preserve the original YAML value (fallback behavior).
- Functions available in templates: `lower`, `upper`, `replace`.

Merge behavior:

- For sequences: `from-param-merge` appends items.
- For mappings: `from-param-merge` appends key/value pairs.

=== My Engineering preferences ===

# Overall

## Think before coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them -- don't pick silently.
- If a simpler approach exists, say so. Push back when warratend.
- If something is unclear, stop. Name what's confusing. Ask.

## Simplicity first

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- DRY is important -- flag repetition aggressively.
- I want to code that's "engineered enough" -- not under-engineered (fragile, hacky) and not over-engineered (premature abstraction, unnecessary complexity).
- I err on the side of handling more edge cases, not fewer; throughfulness > speed.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it -- don't delete it.

When your changes create orphans:
- Remove imports/variables/functions/classes that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## Goal-Driven Execution

**Define succeess criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" -> "Write tests for invalid inputs, then make them pass"
- "Fix the bug" -> "Write a test that reproduces it, then make it pass"
- "Refactor X" -> "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]
```

# Architecture

- Overall system design and component boundaries.
- Dependency graph and coupling concerns.
- Data flow patterns and potential bottlenecks.
- Scaling characteristics and single point of failure.
- Security architecture (auth, data access, api boundaries).
- Bias toward explicit over clever.

# Code quality

- Code organization and module structure.
- DRY violations -- be aggressive here.
- Error handling patterns and missing edge cases (call these out explicity).
- Technical debt hotspots.

# Testing

- Well-tested code is non-negotiable; I'd rather have too many tests than too few.
- Test coverage gaps (unit, integration, e2e).
- Test quality and assertion strength.
- Missing edge case coverage--be thorough.
- Untested failure modes and error paths.
- Once a working task case is implement, provide a justification for modifying the case.

# Perfomance

- N+1 queries and database access patterns.
- Memory-usage concerns.
- Caching opportunities.
- Slow or high-complexity code paths.
