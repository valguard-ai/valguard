# Install ValGuard CLI (`valguard`)

Binary: **`valguard`** · module `github.com/valguard-ai/valguard` · release **`v0.1.3`**

Pick **one** command for your machine.

## Linux or macOS (recommended)

No Go or Python required. Verifies the release archive against `checksums.txt` and installs to `~/.local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/valguard-ai/valguard/main/scripts/install.sh | bash
valguard version
```

Override install dir or version:

```bash
VALGUARD_INSTALL_DIR=/usr/local/bin VALGUARD_VERSION=v0.1.3 \
  curl -fsSL https://raw.githubusercontent.com/valguard-ai/valguard/main/scripts/install.sh | bash
```

## macOS (Homebrew)

```bash
brew install valguard-ai/tap/valguard
valguard version
```

## Python (any OS with pip)

Package name **`valguard-cli`**, command still **`valguard`**. First run downloads and verifies the release binary.

```bash
pip install valguard-cli
valguard version
```

Do **not** `pip install valguard` — unrelated project.

## Go developers

Requires Go **1.26+**.

```bash
go install github.com/valguard-ai/valguard/cmd/valguard@v0.1.3
```

## Manual download

Archives and `checksums.txt` on [GitHub Releases](https://github.com/valguard-ai/valguard/releases) (`linux` / `darwin` / `windows`, `amd64` and `arm64`; no `windows/arm64`).

## GitHub Action

```yaml
- uses: valguard-ai/valguard/action@v0.1.3
  with:
    policy-file: valguard.yaml
    output-file: model-output.json
```