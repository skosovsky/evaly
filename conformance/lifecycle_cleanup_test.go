package conformance_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/conformance"
)

type cleanupMarker struct {
	Handle    int  `json:"handle"`
	Deadline  bool `json:"deadline"`
	Cancelled bool `json:"cancelled"`
}

func TestLifecycleCleanupOwnsEveryReturnedHandle(t *testing.T) {
	for _, mode := range []string{"success", "reset_failure", "partial_prepare", "goexit", "cleanup_error", "cleanup_only_error", "cleanup_deadline"} {
		t.Run(mode, func(t *testing.T) {
			// Arrange: an isolated process lets us assert behavior after Fatal/Goexit.
			marker := filepath.Join(t.TempDir(), "cleanup.json")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLifecycleCleanupProcess$", "-test.v")
			command.Env = append(os.Environ(), "EVALY_LIFECYCLE_MODE="+mode, "EVALY_CLEANUP_MARKER="+marker)
			// Act.
			output, err := command.CombinedOutput()
			// Assert: cleanup always happens exactly once, detached with a real deadline.
			if ctx.Err() != nil {
				t.Fatal("child did not finish", ctx.Err(), string(output))
			}
			if (err == nil) != (mode == "success") {
				t.Fatal(mode, err, string(output))
			}
			verifyCleanupMarker(t, marker)
			verifyLifecycleFailureOutput(t, mode, string(output))
		})
	}
}

func verifyCleanupMarker(t *testing.T, path string) {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cleanup marker absent", err)
	}
	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	if len(lines) != 1 {
		t.Fatal("cleanup count", len(lines), string(bytes))
	}
	var marker cleanupMarker
	if err := json.Unmarshal([]byte(lines[0]), &marker); err != nil {
		t.Fatal(err)
	}
	if marker.Handle != 7 || !marker.Deadline || marker.Cancelled {
		t.Fatal("invalid cleanup ownership/context", marker)
	}
}

func TestLifecycleCleanupProcess(t *testing.T) {
	mode := os.Getenv("EVALY_LIFECYCLE_MODE")
	if mode == "" {
		return
	}
	// Arrange: fault callbacks operate on a known partial handle.
	lifecycle := evaly.LifecycleFuncs[int]{
		IdentityValue: evaly.LifecycleIdentity{Fixture: "fixture-v1", Reset: "reset-v1", Isolation: evaly.SerialShared},
		PrepareFunc: func(context.Context, string) (int, error) {
			if mode == "partial_prepare" {
				return 7, errors.New("primary_prepare_failure")
			}
			return 7, nil
		},
		ResetFunc: func(context.Context, int) error {
			if mode == "goexit" {
				t.Log("primary_reset_goexit")
				runtime.Goexit()
			}
			if mode == "reset_failure" || mode == "cleanup_error" {
				return errors.New("primary_reset_failure")
			}
			return nil
		},
		CleanupFunc: func(ctx context.Context, handle int) error {
			_, deadline := ctx.Deadline()
			bytes, err := json.Marshal(cleanupMarker{Handle: handle, Deadline: deadline, Cancelled: ctx.Err() != nil})
			if err != nil {
				return err
			}
			// #nosec G703 -- parent test creates this marker path in its private TempDir.
			file, err := os.OpenFile(os.Getenv("EVALY_CLEANUP_MARKER"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			_, writeErr := file.Write(append(bytes, '\n'))
			closeErr := file.Close()
			if err = errors.Join(writeErr, closeErr); err != nil {
				return err
			}
			if mode == "cleanup_deadline" {
				<-ctx.Done()
				return ctx.Err()
			}
			if mode == "cleanup_error" || mode == "cleanup_only_error" {
				return errors.New("secondary_cleanup_failure")
			}
			return nil
		},
	}
	// Act / Assert: child Fatal/Goexit is observed by the parent process.
	conformance.LifecycleWithOptions(
		t,
		lifecycle,
		conformance.LifecycleOptions{ID: "child-" + mode, Timeout: time.Second, CleanupTimeout: 40 * time.Millisecond},
	)
}

func verifyLifecycleFailureOutput(t *testing.T, mode, output string) {
	t.Helper()
	primary := map[string]string{
		"partial_prepare": "primary_prepare_failure",
		"reset_failure":   "primary_reset_failure",
		"cleanup_error":   "primary_reset_failure",
		"goexit":          "primary_reset_goexit",
	}[mode]
	if primary != "" && !strings.Contains(output, primary) {
		t.Fatal(output)
	}
	if strings.HasPrefix(mode, "cleanup") && !strings.Contains(output, "cleanup failure") {
		t.Fatal(output)
	}
	if strings.HasSuffix(mode, "error") && !strings.Contains(output, "secondary_cleanup_failure") {
		t.Fatal(output)
	}
}
