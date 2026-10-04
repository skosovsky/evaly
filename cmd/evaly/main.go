package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

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
	policyFile := f.String("policy", "", "versioned comparison policy JSON file (required for compare)")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 || *directory == "" {
		fmt.Fprintln(errout, "invalid arguments")
		return 3
	}
	var policy evaly.ComparisonPolicy
	var objective evaly.Objective
	var e error
	if args[0] == "compare" {
		raw, e := os.ReadFile(*policyFile)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		policy, e = evaly.DecodeWire[evaly.ComparisonPolicy](raw)
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
		objective, e = policy.Resolve()
		if e != nil {
			fmt.Fprintln(errout, e)
			return 3
		}
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
		comp, e := evaly.Compare(b, c, objective, policy.Gate)
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
			fmt.Fprintf(
				out,
				"host trial %s case %s revision %s seed %d target %s fixture %s reset %s evidence revision %s\n",
				trial.TrialID, trial.CaseID, trial.CaseRevision, trial.Seed,
				trial.Target, trial.Fixture, trial.Reset, trial.EvidenceRevision,
			)
		}
		return evaly.ExitCode(comp.Verdict)
	default:
		fmt.Fprintln(errout, "unknown command")
		return 3
	}
}
