# Contributing

Thanks for helping with the ValGuard offline CLI.

## Public repository (`github.com/valguard-ai/valguard`)

- Open pull requests against **`main`**.
- Before review:

  ```bash
  go test -race -count=1 ./...
  go run ./cmd/valguard test --dir conformance/fixtures
  ```

- Keep new files under **Apache-2.0**. Do not add network calls to `cmd/valguard` paths used by `check` / `test` / `report` / `export` / `verify-proof`.
- Public sources are **MVP-only** (~40 rules). Enterprise catalog files are stripped by the monorepo sync filter and must not be reintroduced here.

## Monorepo development

Product development of the full catalog and hosted proxy still happens in the ValGuard GitLab monorepo. The public tree is a filtered mirror of `oss/valguard/`. Prefer fixing validators and fixtures there, then sync:

```bash
make oss-phase-c-gate
make oss-public-filter-smoke
bash scripts/sync-oss-github-mirror.sh v0.1.1   # maintainers
```

Publication steps: monorepo `docs/runbooks/oss-v0.1.0-publication.md`.

## Conduct and security

- [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md) — `conduct@valguard.ai`
- [SECURITY.md](./SECURITY.md) — report vulnerabilities to `security@valguard.ai` (not public issues)
