package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunVerifyProof_rejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runVerifyProof([]string{"--file", path}); code != 2 {
		t.Fatalf("exit %d want 2", code)
	}
}
