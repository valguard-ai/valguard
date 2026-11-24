# Invoice output validation (OSS example)

Validates JSON invoice agent output offline with the same rule IDs as ValGuard production.

```bash
# From this directory (or pass absolute paths)
valguard check --policy valguard.yaml --file sample-output.json

# Inline JSON instead of a file:
valguard check --policy valguard.yaml --output '{"total":1250.5,"currency":"USD"}'

# Shadow report expects JSONL records (one JSON object per line with an "output" field),
# or a fixtures directory via --dir — not a raw response file via --output.
printf '%s\n' '{"output":"{\"total\":1250.5,\"currency\":\"USD\"}"}' > runs.jsonl
valguard report --shadow --policy valguard.yaml --file runs.jsonl
```

`--output` is for **inline response text**. Use `--file` for a path on disk.

Policy is YAML v1 (see monorepo `docs/oss/policy-yaml-v1.md`, or the schema in the root README). Import an exported playbook fragment into ValGuard Cloud when you are ready to enforce in production:

```bash
valguard export --policy valguard.yaml --out playbook.json
```
