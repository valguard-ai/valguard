# Conformance fixtures (OSS module)

JSON fixtures under `fixtures/` drive offline `valguard test --dir conformance/fixtures` and
GitHub Actions on `github.com/valguard-ai/valguard`.

Each file embeds its own YAML-v1 `policy` object (not a path to a separate policy file). Shape:

```json
{
  "name": "case name",
  "policy": { "version": 1, "name": "…", "mode": "enforce", "rules": [/* … */] },
  "output": "…",
  "input_text": "",
  "expect_blocked": true,
  "expect_would_enforce": true,
  "expect_failed_rules": ["rule_id"]
}
```

In the ValGuard monorepo, the canonical copy lives at repo-root `conformance/fixtures/`. After
editing those files, run `make oss-sync-conformance-fixtures` before syncing the public mirror.

Dual-run OSS vs proxy engine is tested only in the GitLab monorepo:
`apps/proxy/internal/conformance/`.
