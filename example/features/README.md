# Features examples

This directory demonstrates `ymr` features:

- Array indexing in simple directives (e.g. `.app.ports.0`)
- CLI `--param` now parses floats (in addition to ints/bools/strings)
- Deterministic map merge (`from-param-merge` overrides existing keys)
- parameter source precedence (param-file < param-yaml < --param)
- Simple directives support nested paths (e.g. `.app.image.tag`) while preserving YAML types
- spec-based runs with multiple targets
- Stricter directive detection (comments must start with `from-param:` / `from-param-merge:`)
- `--strict` fails on missing directive keys and directive template errors
- validate-only mode (`run --validate`)
- validation override via `--validation`

## Run spec-based example (renders multiple templates)

From repo root:

```bash
go run . run -s example/features -o -

# Render only one target
go run . run -s example/features -t dev -o -

# Override the template list while still using spec params/targets
go run . run -s example/features -T example/features/nested.yaml -t dev -o -

# Strict should also succeed for the templates referenced by spec.yaml
go run . run -s example/features --strict -o -
```

## Output to files

```bash
rm -rf /tmp/ymr-features-out
go run . run -s example/features -t dev -o /tmp/ymr-features-out
ls /tmp/ymr-features-out
```

## Init command (generates a spec.yaml)

```bash
tmpdir=$(mktemp -d)
(cd "$tmpdir" && \
  go run /path/to/this/repo init --templates service.yaml --target dev --target prd -p minScale=1 -p maxScale=3)
```

Replace `/path/to/this/repo` with the path to your local checkout.

## Validate-only mode (no output files)

```bash
go run . run -s example/features --validate

# Validate with strict processing (should still pass for this example)
go run . run -s example/features --validate --strict
```

## Validation override file

```bash
# Uses example/features/validations-override.yaml instead of spec.yaml validations
go run . run -s example/features --validate --validation example/features/validations-override.yaml

# Show a failure by overriding replicas
go run . run -s example/features --validate --validation example/features/validations-override.yaml -p replicas=1
```

## Run spec-less examples

### CLI param typing (int/bool/float/string)

```bash
go run . run -T example/features/cli-params.yaml -t dev -o - \
  -p replicas=3 \
  -p enabled=true \
  -p ratio=1.5 \
  -p name=myapp
```

## Remote source loading (reference)

`ymr` can load specs/templates from local paths, HTTP(S), and GitHub URLs.
Examples below assume the repo is accessible (and may require `--token` or `GITHUB_TOKEN` for private repos).

```bash
# GitHub repo source (loads spec.yaml from the repo root or subdir)
go run . run -s https://github.com/lnrdll/ymr/example/features@main -t dev -o -

# GitHub blob URL for a single template (spec-less)
go run . run -T https://github.com/lnrdll/ymr/blob/main/example/features/nested.yaml -t dev -o - \
  --param-yaml "app: { image: { tag: 1.2.3 }, ports: [80, 443] }" \
  -p ratio=1.5
```

### Parameter source precedence

`--param` wins over `--param-yaml`, which wins over `--param-file`.

```bash
go run . run -T example/features/precedence.yaml -t dev -o - \
  --param-file example/features/params-precedence.yaml \
  --param-yaml "name: from-yaml" \
  -p name=from-cli
```

### Strict-mode directive failures

Non-strict preserves defaults (and removes directives):

```bash
go run . run -T example/features/strict-failure.yaml -t dev -o - -p name=myapp
```

Strict mode errors (missing key + template execution error):

```bash
go run . run -T example/features/strict-failure.yaml -t dev --strict -o - -p name=myapp
```
