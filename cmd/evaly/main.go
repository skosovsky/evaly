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
		return invalidExitCode
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
		return invalidExitCode
	}
	var policy evaly.ComparisonPolicy
	var objective evaly.Objective
	var e error
	if args[0] == "compare" {
		// #nosec G703 -- the local CLI operator explicitly selects the policy file.
		raw, eLocal := os.ReadFile(*policyFile)
		if eLocal != nil {
			fmt.Fprintln(errout, eLocal)
			return invalidExitCode
		}
		policy, eLocal = evaly.DecodeWire[evaly.ComparisonPolicy](raw)
		if eLocal != nil {
			fmt.Fprintln(errout, eLocal)
			return invalidExitCode
		}
		objective, eLocal = policy.Resolve()
		if eLocal != nil {
			fmt.Fprintln(errout, eLocal)
			return invalidExitCode
		}
	}
	// #nosec G703 -- the local CLI operator explicitly selects the artifact directory.
	store, e := evaly.OpenFileStore(*directory)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	switch args[0] {
	case "fixture":
		return runFixture(ctx, store, *id, *behavior, *caseID, *seed, out, errout)
	case "compare":
		return runComparison(ctx, store, *baseline, *candidate, objective, policy, out, errout)
	default:
		fmt.Fprintln(errout, "unknown command")
		return invalidExitCode
	}
}

func runFixture(
	ctx context.Context,
	store evaly.ArtifactStore,
	id, behavior, caseID string,
	seed int64,
	out, errout io.Writer,
) int {
	c, e := fixtures.CalculationConfig(id, behavior, caseID)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	c.Plan.Seed = seed
	exp, e := evaly.Run(ctx, c)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	if e = evaly.SaveExperiment(ctx, store, exp); e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	fmt.Fprintf(out, "sealed experiment %s revision %s\n", exp.ID(), exp.Revision())
	for _, t := range exp.Record().Trials {
		fmt.Fprintf(out, "trial %s: %s; cleanup %s\n", t.ID, t.Status, t.Cleanup.State)
	}
	return 0
}

func runComparison(
	ctx context.Context,
	store evaly.ArtifactStore,
	baseline, candidate string,
	objective evaly.Objective,
	policy evaly.ComparisonPolicy,
	out, errout io.Writer,
) int {
	b, e := evaly.LoadExperiment(ctx, store, baseline)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	c, e := evaly.LoadExperiment(ctx, store, candidate)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	comp, e := evaly.Compare(b, c, objective, policy.Gate)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	artifact, e := evaly.NewEnvelope("comparison", comp.Revision, comp)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	if e = store.Put(ctx, artifact); e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
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
}
