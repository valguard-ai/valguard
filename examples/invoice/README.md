# Invoice output validation (OSS example)

Validates JSON invoice agent output offline with the same rule IDs as ValGuard production.

```bash
valguard check --policy valguard.yaml --output sample-output.json
valguard report --shadow --policy valguard.yaml --output sample-output.json
```

Policy uses YAML v1 (`docs/oss/policy-yaml-v1.md` in the monorepo). Import the exported playbook fragment into ValGuard Cloud when you are ready to enforce in production.
