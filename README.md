# ValGuard OSS module (Phase C staging)

Public module path: **`github.com/valguard-ai/valguard`** (separate GitHub repo at launch, Apache-2.0).

## Monorepo development

**Branch:** `feature/oss-phase-c-prep` (supersedes `feature/oss-valguard-full-implementation` for integration with current `master`).

| Artifact | Path |
|----------|------|
| Offline CLI | `apps/proxy/cmd/valguard` — `make oss-valguard-build` |
| Conformance | `make oss-conformance-test` |
| Fixtures | `conformance/fixtures/` |
| Runbook | [docs/opensource-cli-phase-c.md](../../docs/opensource-cli-phase-c.md) |

This directory is the **C1 skeleton** for the future public repo: module metadata, LICENSE, CI workflow template, and standalone `pkg/evidence` (no proxy import).

During Phase C the CLI still builds from **`apps/proxy`** and uses `replace github.com/valguard-ai/valguard/apps/proxy => ../../apps/proxy` in `go.mod` until validators/engine are extracted here (C2–C4, C9).

Docs: [policy YAML v1](../../docs/oss/policy-yaml-v1.md), [gate 0 decisions](../../docs/decisions/oss-cli-v1.md).
