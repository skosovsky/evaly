package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIExitCodesAndSealedBaseline(t *testing.T) {
	// Arrange.
	bin := filepath.Join(t.TempDir(), "evaly")
	build := exec.Command("go", "build", "-o", bin, ".")
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build %v: %s", e, b)
	}
	store := t.TempDir()
	run := func(want int, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(bin, args...)
		b, e := cmd.CombinedOutput()
		code := 0
		exit := &exec.ExitError{}
		if errors.As(e, &exit) {
			code = exit.ExitCode()
		} else if e != nil {
			t.Fatal(e)
		}
		if code != want {
			t.Fatalf("exit %d want %d: %s", code, want, b)
		}
		return b
	}
	// Act / Assert.
	for _, behavior := range []string{"good", "bad", "partial", "judge-error"} {
		run(0, "fixture", "--store", store, "--id", behavior, "--behavior", behavior)
	}
	pass := run(0, "compare", "--store", store, "--baseline", "good", "--candidate", "good")
	fail := run(1, "compare", "--store", store, "--baseline", "good", "--candidate", "bad")
	inconclusive := run(2, "compare", "--store", store, "--baseline", "good", "--candidate", "partial")
	run(2, "compare", "--store", store, "--baseline", "good", "--candidate", "judge-error")
	run(3, "compare", "--store", store, "--baseline", "missing", "--candidate", "good")
	run(3, "fixture", "--store", store, "--id", "../bad")
	if !bytes.Contains(fail, []byte("--case case-4 --behavior bad --seed 0")) ||
		!bytes.Contains(fail, []byte("trial bad/case-4/0/0")) ||
		!bytes.Contains(fail, []byte("evidence ")) {
		t.Fatal("missing accurate replay/trial/evidence report", string(fail))
	}
	if !bytes.Contains(pass, []byte("evaly pass")) || !bytes.Contains(fail, []byte("evaly fail")) ||
		!bytes.Contains(inconclusive, []byte("scored 2/4")) {
		t.Fatal("report/exit mismatch")
	}
}
