package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltBinaryReportsInjectedVersion(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "chainquery")
	const commit = "test-commit-sha"
	ldflags := "-X github.com/lbryio/chainquery/meta.version=" + commit +
		" -X github.com/lbryio/chainquery/meta.versionLong=" + commit

	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "-ldflags", ldflags, ".")
	buildOutput, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s\n%s", err, buildOutput)
	}

	version := exec.Command(binary, "version")
	versionOutput, err := version.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %s\n%s", err, versionOutput)
	}

	output := string(versionOutput)
	if !strings.Contains(output, "Version: "+commit) {
		t.Fatalf("expected short version %q in output:\n%s", commit, output)
	}
	if !strings.Contains(output, "Version(long): "+commit) {
		t.Fatalf("expected long version %q in output:\n%s", commit, output)
	}
}
