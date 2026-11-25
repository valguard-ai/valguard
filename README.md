# ValGuard — open-source validation CLI

**Apache-2.0** · module [`github.com/valguard-ai/valguard`](https://github.com/valguard-ai/valguard) · binary **`valguard`**

Offline, deterministic **output** validation for LLM agents: **40 MVP rules**, same rule IDs as ValGuard production, with no network calls in `check`, `test`, `report`, `export`, or `verify-proof`.

Hosted proxy, re-ask, repair, input validators, and the full ~160-rule catalog remain on [ValGuard Cloud](https://www.valguard.ai).

Requires **Go 1.26+** to build from source (`go 1.26.0` in `go.mod`).

## Install

> **Launch note:** `go install …@v0.1.2`, GitHub Release archives, Homebrew, and `pip install valguard-cli` work only after the public repo and tag **v0.1.2** exist. Until then, build from this tree (`go build -o valguard ./cmd/valguard`) or from the ValGuard monorepo (`make oss-valguard-build`).

```bash
# Go module (after v0.1.2 is published)
go install github.com/valguard-ai/valguard/cmd/valguard@v0.1.2

# Or download a GoReleaser archive from GitHub Releases
#   valguard_0.1.2_<os>_<arch>.tar.gz  (linux/darwin/windows; amd64/arm64; no windows/arm64)

# Homebrew (after valguard-ai/homebrew-tap is published)
#   brew tap valguard-ai/tap && brew install valguard

# PyPI wrapper downloads the same release binary (console script still named valguard)
#   pip install valguard-cli
```

PyPI package name: **`valguard-cli`** (not `valguard` on PyPI — that name is an unrelated project).

See [INSTALL.md](./INSTALL.md) for platform details and [SECURITY.md](./SECURITY.md) for vulnerability reporting.

## Quickstart

```bash
valguard init --path valguard.yaml
valguard check --policy valguard.yaml --output '{"message":"ok"}'
valguard check --policy valguard.yaml --file response.json
valguard test --dir conformance/fixtures
valguard rules list
valguard rules explain valid_format
valguard report --shadow --policy valguard.yaml --file runs.jsonl
valguard export --policy valguard.yaml --out playbook.json
valguard verify-proof --file envelope.json
valguard version   # 0.1.2
```

Policy format: YAML v1 (`version`, `name`, `mode`, `rules[]`; `on_fail`: `block` | `warn` | `log`; no `reask`).

### Exit codes (check / test)

| Code | Meaning |
|------|---------|
| `0` | Pass / all fixtures match |
| `1` | Enforce block, shadow would-enforce on `check`, or fixture mismatch on `test` |
| `2` | Usage / config (bad policy, missing file/dir, output over 4 MiB) |

`report --shadow` exits `0` when it can write the report (including would-block rows). `verify-proof` uses its own split (`0` = `ok`, `1` = mismatch/IO/prod secret, `2` = invalid envelope/flags).

## Local vs hosted CI

| | `valguard test` (OSS) | Hosted helper `valguard-cli test` |
|---|------------------------|-------------------------------------|
| Policy | Embedded in each fixture JSON, or YAML for `check` | Agent in DB (`agent_slug`) |
| Network | **None** | Yes (internal API) |
| Rules | 40 MVP output rules | Full catalog + input path |
| Binary | `valguard` | Built from ValGuard monorepo `apps/proxy/cmd/valguard-cli` |

## Development

**Public clone** (this repo as root):

```bash
go test -race -count=1 ./...
go run ./cmd/valguard test --dir conformance/fixtures
```

**Monorepo** (ValGuard GitLab tree; this directory is `oss/valguard/`):

```bash
make oss-phase-c-gate
make oss-sync-conformance-fixtures
```

The **GitHub mirror** is filtered on sync (`scripts/oss-public-filter.sh`): only **40 MVP** validator sources are published. The monorepo `oss/valguard/` tree still contains the full catalog for hosted proxy builds.

## Telemetry

The OSS CLI does **not** phone home. CI asserts no `net/http` dependency in `cmd/valguard`.

## License

Apache-2.0 — see [LICENSE](./LICENSE). Marks: [TRADEMARKS.md](./TRADEMARKS.md). Support: [SUPPORT.md](./SUPPORT.md).
