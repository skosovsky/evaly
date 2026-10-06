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

const (
	compareCommand = "compare"
	fixtureCommand = "fixture"
)

func main() { os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr)) }
func run(ctx context.Context, args []string, out, errout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errout, "usage: evaly fixture|compare [flags]")
		return invalidExitCode
	}
	options, parseErr := parseCommand(args, errout)
	if parseErr != nil {
		fmt.Fprintln(errout, parseErr)
		return invalidExitCode
	}
	var policy evaly.ComparisonPolicy
	var objective evaly.Objective
	var e error
	if options.command == compareCommand {
		// #nosec G703 -- the local CLI operator explicitly selects the policy file.
		raw, eLocal := os.ReadFile(options.policyFile)
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
	store, e := evaly.OpenFileStore(options.directory)
	if e != nil {
		fmt.Fprintln(errout, e)
		return invalidExitCode
	}
	switch options.command {
	case fixtureCommand:
		return runFixture(ctx, store, options.id, options.behavior, options.caseID, options.seed, out, errout)
	case compareCommand:
		return runComparison(ctx, store, options.baseline, options.candidate, objective, policy, out, errout)
	default:
		fmt.Fprintln(errout, "unknown command")
		return invalidExitCode
	}
}

type commandOptions struct {
	command    string
	directory  string
	id         string
	behavior   string
	seed       int64
	caseID     string
	baseline   string
	candidate  string
	policyFile string
}

func parseCommand(args []string, errout io.Writer) (commandOptions, error) {
	var options commandOptions
	options.command = args[0]
	if options.command != fixtureCommand && options.command != compareCommand {
		return options, fmt.Errorf("unknown command %q: %w", options.command, evaly.ErrInvalid)
	}
	flags := flag.NewFlagSet(options.command, flag.ContinueOnError)
	flags.SetOutput(errout)
	flags.StringVar(&options.directory, "store", "", "artifact directory")
	switch options.command {
	case fixtureCommand:
		flags.StringVar(&options.id, "id", "", "experiment identity")
		flags.StringVar(&options.behavior, "behavior", "good", "fixture behavior: good|bad|partial|judge-error")
		flags.Int64Var(&options.seed, "seed", 0, "trial seed for fixture replay")
		flags.StringVar(&options.caseID, "case", "", "single fixture case to replay in a fresh environment")
	case compareCommand:
		flags.StringVar(&options.baseline, "baseline", "", "sealed baseline identity")
		flags.StringVar(&options.candidate, "candidate", "", "sealed candidate identity")
		flags.StringVar(
			&options.policyFile,
			"policy",
			"",
			"versioned comparison policy JSON file (required for compare)",
		)
	}
	if err := flags.Parse(args[1:]); err != nil {
		return options, err
	}
	if flags.NArg() != 0 || options.directory == "" {
		return options, evaly.ErrInvalid
	}
	if err := validateCommandOptions(options); err != nil {
		return options, err
	}
	return options, nil
}

func validateCommandOptions(options commandOptions) error {
	identities := []string{options.baseline, options.candidate}
	if options.command == fixtureCommand {
		identities = []string{options.id}
		switch options.behavior {
		case "good", "bad", "partial", "judge-error":
		default:
			return evaly.ErrInvalid
		}
	} else if options.policyFile == "" {
		return evaly.ErrInvalid
	}
	for _, id := range identities {
		if _, err := evaly.NewEnvelope("experiment", id, struct{}{}); err != nil {
			return err
		}
	}
	return nil
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
