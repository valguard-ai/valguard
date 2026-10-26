# ValGuard — open-source validation CLI

**Apache-2.0** · [`github.com/valguard-ai/valguard`](https://github.com/valguard-ai/valguard)

Offline, deterministic **output** validation for LLM agents: ~**40 MVP rules**, same rule IDs as ValGuard production, with no network calls in `check`, `test`, or `report`.

Hosted proxy, re-ask, repair, input validators, and the full ~160-rule catalog remain on [ValGuard Cloud](https://www.valguard.ai).

## Install (after v0.1.0 release)

```bash
# Go
go install github.com/valguard-ai/valguard/cmd/valguard@v0.1.0

# Or download from GitHub Releases (linux/darwin/windows)
```

PyPI package name: **`valguard-cli`** (not `valguard` on PyPI).

## Quickstart

```bash
valguard init --path valguard.yaml
valguard check --policy valguard.yaml --output '{"message":"ok"}'
valguard test --dir conformance/fixtures
valguard rules list
valguard rules explain valid_format
valguard report --shadow --policy valguard.yaml --output out.json
valguard export --policy valguard.yaml
valguard verify-proof -- evidence.json
```

Policy format: YAML v1 (`version`, `name`, `mode`, `rules[]`; `on_fail`: `block` | `warn` | `log`; no `reask`).

## Local vs hosted CI

| | `valguard test` (OSS) | Host `valguard-cli test` |
|---|------------------------|---------------------------|
| Policy | `valguard.yaml` | Agent in DB (`agent_slug`) |
| Network | **None** | Yes (internal API) |
| Rules | 40 MVP output rules | Full catalog + input path |

## Development (monorepo)

This tree is synced from the ValGuard GitLab monorepo (`oss/valguard/`). Gate:

```bash
make oss-phase-c-gate          # from repo root
make oss-sync-conformance-fixtures
```

The **GitHub mirror** is filtered on sync (`scripts/oss-public-filter.sh`): only **~40 MVP** validator sources are published. The monorepo `oss/valguard/` tree still contains the full catalog for hosted proxy builds.

## Telemetry

The OSS CLI does **not** phone home. CI asserts no `net/http` dependency in `cmd/valguard`.

## License

Apache-2.0 — see [LICENSE](./LICENSE).
