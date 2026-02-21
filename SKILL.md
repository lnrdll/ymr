# Skill: ymr

Use `ymr` to render YAML templates by applying substitutions encoded as YAML comment directives.

This file is written for automated agents (code assistants) so they can use `ymr` predictably and safely.

## What ymr Does

- Input: YAML template(s) with directives in comments plus a parameter source.
- Output: rendered YAML content written to stdout (`-o -`) or to files (`-o <dir>`).

Directives:

- `# from-param: <go-template>`: replace the node value with the rendered result.
- `# from-param-merge: <go-template>`: merge lists/maps into the existing node.

Template language: Go `text/template` with helpers `lower`, `upper`, `replace`.

## Invocation Modes

### Spec Mode (recommended when you have multiple templates/targets)

Provide `--spec` (`-s`) pointing at a directory containing `spec.yaml` or a `spec.yaml` file itself.

Examples:

```bash
# spec.yaml in current directory
ymr run -s . -o rendered

# spec.yaml file path
ymr run -s ./spec.yaml -o -

# spec.yaml in an example directory
ymr run -s example/k8s -t dev -o -
```

### Spec-less Mode (default)

If you omit `--spec`, `ymr` runs in spec-less mode.

Requirements:

- `--template` (`-T`) is required
- at least one `--param` (`-p`) is required
 - at least one of `--param` (`-p`), `--param-file`, or `--param-yaml` is required

Examples:

```bash
# Print rendered content to stdout
ymr run -T ./example/k8s/deployment.yaml -p name=myapp -o -

# Load a YAML params mapping from a file
ymr run -T ./example/k8s/deployment.yaml -t dev --param-file ./params.yaml -o -

# Inline YAML/JSON mapping
ymr run -T ./example/k8s/deployment.yaml -t dev --param-yaml 'labels: { env: dev }' -o -

# Write to files; pass a target label for stable filenames
ymr run -T ./example/k8s/deployment.yaml -t dev -p name=myapp -o rendered
```

## Inputs / Outputs for Agent Workflows

**Preferred for agents**: Use stdout first (`-o -`), inspect/validate, then (only if needed) write to files.

When writing to files:

- In spec mode, filenames are `${targetId}-${templateBaseName}${ext}`.
- In spec-less mode, pass at least one `--target` or you will get an empty target id in filenames (e.g. `-deployment.yaml`).

## Parameter Rules

- Spec mode parameters come from `spec.yaml` (`parameters[].values`) merged by `targetId`.
- `--param key=value` overrides parameters for the current run.
- `--param-file <path|url>` merges in a YAML/JSON mapping (can be repeated).
- `--param-yaml <yaml>` merges in an inline YAML/JSON mapping (can be repeated).
- Type inference for CLI params: ints and bools are parsed; otherwise values are strings.

## Validation Rules

- Validations are CEL expressions.
- Only validations that explicitly list the current `targetId` apply.
- `--validation <path>` overrides validations defined in `spec.yaml`.

## Source Rules (Local / HTTP / GitHub)

- Templates and spec can be loaded from local paths or HTTP(S) URLs.
- GitHub sources can require auth: use `--token` or `GITHUB_TOKEN`.

Agent safety:

- Never print or persist tokens in logs, rendered output, or commit messages.

## Troubleshooting Checklist

- YAML parse errors: verify the template is valid YAML before directives.
- Missing keys: `ymr` falls back to the original YAML value when template evaluation fails.
- Validation failures: the error message comes from the rule's `message` when set.
- For more signal: re-run with `--debug`.

## Minimal Prompt Template (for tool-using agents)

When invoking `ymr` from an agent, supply:

- mode: spec or spec-less
- source paths (spec directory/file or template path)
- targets to render
- params to override
- output destination (`-o -` vs `-o <dir>`)

Example agent intent:

"Render `example/k8s` for targets `dev` and `prd` to stdout; override `name=myapp`; ensure validations pass." 
