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

## Source Loading

`--spec` accepts:

- Local directory (loads `<dir>/spec.yaml`)
- Local file (loads that file)
- HTTP(S) URL (loads that URL)
- GitHub URL (supports raw fetch via `raw.githubusercontent.com`)

Use `--token` or `GITHUB_TOKEN` for private GitHub sources.

## Tests

- Unit tests live in `internal/processor/main_test.go`.
