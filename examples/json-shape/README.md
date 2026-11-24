# JSON shape (required fields)

```bash
valguard check --policy valguard.yaml --file ok.json
# expect: pass (exit 0)
```

`ok.json` is valid JSON with a `status` field required by the policy. Missing `status` or non-JSON text should **block**.
