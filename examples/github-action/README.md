# GitHub Action

Composite action at `action/action.yml` (repo root after clone). It downloads the **`valguard`** binary from GitHub Releases (default tag `v0.1.3`) and runs `valguard check`.

```yaml
- uses: valguard-ai/valguard/action@v0.1.3
  with:
    version: v0.1.3          # optional; default v0.1.3
    policy-file: valguard.yaml
    output-file: model-output.json
```

Requires the public release archive for the runner OS/arch (`linux`/`darwin` × `amd64`/`arm64`). Windows runners are not supported by this action wrapper.
