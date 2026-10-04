package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/internal/fixtures"
)

func main() { os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr)) }
func run(ctx context.Context, args []string, out, errout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errout, "usage: evaly fixture|compare [flags]")
		return 3
	}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.SetOutput(errout)
	directory := f.String("store", "", "artifact directory")
	id := f.String("id", "", "experiment identity")
	behavior := f.String("behavior", "good", "fixture behavior: good|bad|partial|judge-error")
	seed := f.Int64("seed", 0, "trial seed for fixture replay")
	caseID := f.String("case", "", "single fixture case to replay in a fresh environment")
	baseline := f.String("baseline", "", "sealed baseline identity")
	candidate := f.String("candidate", "", "sealed candidate identity")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 || *directory == "" {
		fmt.Fprintln(errout, "invalid arguments")
		return 3
	}
	store, e := evaly.OpenFileStore(*directory)
	if e != nil {
		fmt.Fprintln(errout, e)
		return 3
	}
	switch args[0] {
	case "fixture":
		c, e := fixtures.CalculationConfig(*id, *behavior, *caseID)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		c.Plan.Seed = *seed
		exp, e := evaly.Run(ctx, c)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		if e = evaly.SaveExperiment(ctx, store, exp); e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		fmt.Fprintf(out, "sealed experiment %s revision %s\n", exp.ID(), exp.Revision())
		for _, t := range exp.Record().Trials {
			fmt.Fprintf(out, "trial %s: %s; cleanup %s\n", t.ID, t.Status, t.Cleanup.State)
		}
		return 0
	case "compare":
		b, e := evaly.LoadExperiment(ctx, store, *baseline)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		c, e := evaly.LoadExperiment(ctx, store, *candidate)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		comp, e := evaly.Compare(b, c, fixtures.Objective(), fixtures.Gate())
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		artifact, e := evaly.NewEnvelope("comparison", comp.Revision, comp)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		if e = store.Put(ctx, artifact); e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		fmt.Fprint(out, evaly.Report(comp))
		for _, trial := range comp.Trials {
			fixtureBehavior := strings.TrimSuffix(trial.Target, "-v1")
			fmt.Fprintf(
				out,
				"replay %s: evaly fixture --store DIR --id NEW_ID --case %s --behavior %s --seed %d\n",
				trial.TrialID,
				trial.CaseID,
				fixtureBehavior,
				trial.Seed,
			)
		}
		return evaly.ExitCode(comp.Verdict)
	default:
		fmt.Fprintln(errout, "unknown command")
		return 3
	}
}
