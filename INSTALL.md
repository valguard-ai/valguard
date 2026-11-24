# Install ValGuard CLI (`valguard`)

Module path: `github.com/valguard-ai/valguard`  
Binary name: **`valguard`**  
Semver: **0.1.1** (git tag `v0.1.1`)  
Go toolchain to build from source: **1.26+**

> Until `https://github.com/valguard-ai/valguard` is public and the `v0.1.1` release
> exists, the install commands below will fail. Build locally instead:
>
> ```bash
> go build -o valguard ./cmd/valguard
> ./valguard version
> ```

## go install

```bash
go install github.com/valguard-ai/valguard/cmd/valguard@v0.1.1
valguard version
```

## GitHub Release archives

GoReleaser publishes (when the tag is pushed):

- `valguard_0.1.1_linux_amd64.tar.gz` / `.zip`
- `valguard_0.1.1_linux_arm64.tar.gz` / `.zip`
- `valguard_0.1.1_darwin_amd64.tar.gz` / `.zip`
- `valguard_0.1.1_darwin_arm64.tar.gz` / `.zip`
- `valguard_0.1.1_windows_amd64.tar.gz` / `.zip`
- `checksums.txt`

Windows arm64 is **not** built. Extract and place `valguard` (or `valguard.exe`) on your `PATH`.

```bash
curl -fsSL "https://github.com/valguard-ai/valguard/releases/download/v0.1.1/valguard_0.1.1_linux_amd64.tar.gz" | tar -xz
./valguard version
```

## Homebrew

After `valguard-ai/homebrew-tap` is published:

```bash
brew tap valguard-ai/tap
brew install valguard
```

## PyPI (`valguard-cli`)

```bash
pip install valguard-cli
valguard version
```

The package name is **`valguard-cli`**; the console script is still **`valguard`**. On first run it downloads the matching GitHub Release archive (override with `VALGUARD_BIN` or `VALGUARD_VERSION`). Windows zip extract in the wrapper is not implemented — use `go install` or a release archive on Windows.

Do **not** `pip install valguard` for this CLI; that PyPI name is an unrelated project.

## GitHub Action

```yaml
- uses: valguard-ai/valguard/action@v0.1.1
  with:
    policy-file: valguard.yaml
    output-file: model-output.json
```

Downloads the release binary and runs `valguard check`.
