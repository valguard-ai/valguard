# Conformance fixtures (OSS module)

JSON fixtures under `fixtures/` drive offline `valguard test --dir conformance/fixtures` and GitHub Actions on `github.com/valguard-ai/valguard`.

The monorepo canonical copy lives at `conformance/fixtures/` (repo root). After editing root fixtures, run `make oss-sync-conformance-fixtures` before syncing the public mirror.

Dual-run OSS vs proxy engine is tested in the GitLab monorepo: `apps/proxy/internal/conformance/`.
