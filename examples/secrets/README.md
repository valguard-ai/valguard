# Secrets in model output

```bash
valguard check --policy valguard.yaml --file bad-output.txt
```

Expect **blocked** (exit `1`) when `secrets_detection` matches.

The MVP matcher includes OpenAI-style keys matching `sk-` + 20+ alphanumeric characters (for example `sk-abcdefghijklmnopqrstuvwxyz1234567890`). Patterns with extra hyphens after `sk-` (such as `sk-live-…`) are **not** matched by the v0.1 regex.
