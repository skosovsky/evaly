//go:build integration

package contracttest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegrationConsumerMatrix preserves the published, audited source and
// unsupported SDK boundaries without relying on sibling working trees.
func TestIntegrationConsumerMatrix(t *testing.T) {
	for _, mode := range []string{"published", "source", "unsupported"} {
		t.Run(mode, func(t *testing.T) {
			work, env := prepareConsumer(t, mode)
			// Act: resolve packages before distinguishing unsupported APIs from infrastructure failure.
			missing := missingConsumerCapabilities(t, work, env)
			// Assert: an unsupported source set proves the exact negative boundary only.
			if mode == "unsupported" {
				if !strings.Contains(strings.Join(missing, "\n"), "github.com/skosovsky/prompty.Stream") {
					t.Fatalf("expected missing Stream capability, got %v", missing)
				}
				t.Logf("UNSUPPORTED dependency API: %v; semantic fixtures were not executed", missing)
				return
			}
			if len(missing) != 0 {
				t.Fatalf("UNSUPPORTED dependency API: %v; semantic fixtures were not executed", missing)
			}
			if mode == "published" {
				if replacements := command(
					t,
					work,
					env,
					"go",
					"list",
					"-m",
					"-f",
					"{{if .Replace}}{{.Path}} => {{.Replace.Path}}{{end}}",
					"all",
				); replacements != "" {
					t.Fatalf("published consumer has replacements: %s", replacements)
				}
			}
			command(t, work, env, "go", "test", "-race", "-count=1", "./...")
			command(
				t,
				work,
				env,
				"go",
				"test",
				"-race",
				"-count=1",
				"-tags=integration",
				"-run=^TestIntegration",
				"./...",
			)
			command(t, work, env, "go", "test", "-race", "-count=1", "-tags=e2e", "-run=^TestE2E", "./...")
		})
	}
}

func copyConsumer(t *testing.T, source, destination string) {
	t.Helper()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() ||
			(!strings.HasSuffix(entry.Name(), ".go") && entry.Name() != "go.mod" && entry.Name() != "go.sum") {
			continue
		}
		write(t, filepath.Join(destination, entry.Name()), read(t, filepath.Join(source, entry.Name())))
	}
}

func missingConsumerCapabilities(t *testing.T, dir string, env []string) []string {
	t.Helper()
	var missing []string
	for _, capability := range []string{
		"github.com/skosovsky/prompty.Stream", "github.com/skosovsky/prompty.CaptureReport",
		"github.com/skosovsky/metry/genai.EvaluationRecorder", "github.com/skosovsky/metry/genai.EvaluationSkipped",
	} {
		pkg := capability[:strings.LastIndex(capability, ".")]
		command(t, dir, env, "go", "list", pkg)
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		cmd := exec.CommandContext(ctx, "go", "doc", capability)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			if !strings.Contains(string(out), "no symbol") {
				t.Fatalf("capability lookup infrastructure failure: %v\n%s", err, out)
			}
			missing = append(missing, capability)
			return missing
		}
	}
	return missing
}

func prepareConsumer(t *testing.T, mode string) (string, []string) {
	t.Helper()
	// Arrange: copy the current recipes and use an isolated module graph.
	work := t.TempDir()
	copyConsumer(t, filepath.Join(repoRoot(t), "integrations", "recipes"), work)
	env := []string{
		"GOWORK=off",
		"GOPROXY=https://proxy.golang.org",
		"GONOPROXY=none",
		"GONOSUMDB=",
		"GOSUMDB=sum.golang.org",
	}
	if mode == "published" {
		command(t, work, env, "go", "mod", "edit", "-dropreplace=github.com/skosovsky/evaly")
	} else {
		command(t, work, env, "go", "mod", "edit", "-replace=github.com/skosovsky/evaly="+repoRoot(t))
		for _, sdk := range []struct{ name, supported, unsupported string }{
			{"prompty", "5607832ee869f63bc3c6768a02ed60a019cd691e", "860146dd6e74154abbaad415d9a44e112a05c16d"},
			{"metry", "7ef9f89638f29408f50996236d46d6c105dc8183", "bd61e299a084880437fc79639c2ea14ad75b54a5"},
		} {
			ref := sdk.supported
			if mode == "unsupported" {
				ref = sdk.unsupported
			}
			dir := t.TempDir()
			command(t, dir, nil, "git", "init", "--quiet")
			command(
				t,
				dir,
				nil,
				"git",
				"fetch",
				"--quiet",
				"--depth=1",
				"https://github.com/skosovsky/"+sdk.name+".git",
				ref,
			)
			command(t, dir, nil, "git", "checkout", "--quiet", "--detach", "FETCH_HEAD")
			if actual := command(t, dir, nil, "git", "rev-parse", "HEAD"); actual != ref {
				t.Fatalf("SDK identity %s, want %s", actual, ref)
			}
			t.Logf("%s source: %s", sdk.name, ref)
			command(t, work, env, "go", "mod", "edit", "-replace=github.com/skosovsky/"+sdk.name+"="+dir)
		}
	}
	if mode == "unsupported" {
		// Capability rejection must precede preparation of an unsupported consumer.
		// Module resolution remains mandatory; unrelated SDK test dependencies do not.
		env = append(env, "GOFLAGS=-mod=mod")
	} else {
		command(t, work, env, "go", "mod", "tidy")
	}
	t.Log(command(t, work, env, "go", "list", "-m", "all"))
	return work, env
}
