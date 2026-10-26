# Secrets in model output

```bash
valguard check --policy valguard.yaml --file bad-output.txt
```

Expect **blocked** (enforce mode) when `secrets_detection` matches.
