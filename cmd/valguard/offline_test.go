package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	osspolicy "github.com/valguard-ai/valguard/pkg/policy"
)

// Bramka C: offline CLI must not link net/http (no outbound client in the binary).
func TestValguardCLINoHTTPDependency(t *testing.T) {
	modRoot := moduleRoot(t)
	cmd := exec.Command("go", "list", "-deps", "./cmd/valguard")
	cmd.Dir = modRoot
	cmd.Env = append(os.Environ(), "GOFLAGS=-buildvcs=false")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "net/http" {
			t.Fatal("cmd/valguard must not depend on net/http")
		}
	}
}

func TestValguardCLI_offlineSmoke(t *testing.T) {
	bin := buildValguard(t)
	modRoot := moduleRoot(t)
	fixDir := filepath.Join(modRoot, "conformance", "fixtures")
	if _, err := os.Stat(fixDir); err != nil {
		fixDir = filepath.Join(repoRoot(t), "conformance", "fixtures")
	}
	if _, err := os.Stat(fixDir); err != nil {
		t.Skip("conformance/fixtures not found")
	}

	// Bogus proxy env — any accidental HTTP client would fail fast.
	env := append(os.Environ(),
		"HTTP_PROXY=http://127.0.0.1:1",
		"HTTPS_PROXY=http://127.0.0.1:1",
		"ALL_PROXY=http://127.0.0.1:1",
	)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Env = env
		cmd.Dir = modRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}

	tmp := t.TempDir()
	policyPath := filepath.Join(tmp, "valguard.yaml")
	if err := os.WriteFile(policyPath, []byte(osspolicy.SampleYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	run("test", "--dir", fixDir)
	run("check", "--policy", policyPath, "--output", `{"message":"offline-smoke","ok":true}`)
	run("report", "--shadow", "--dir", fixDir, "--format", "json")
	run("export", "--policy", policyPath, "--slug", "demo")
	run("rules", "list")
}

func buildValguard(t *testing.T) string {
	t.Helper()
	modRoot := moduleRoot(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "valguard")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", bin, "./cmd/valguard")
	build.Dir = modRoot
	build.Env = append(os.Environ(), "GOFLAGS=-buildvcs=false")
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

// moduleRoot is the directory containing go.mod (standalone OSS repo or oss/valguard in monorepo).
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	// oss/valguard/cmd/valguard/offline_test.go → monorepo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}
