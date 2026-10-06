package evaly_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/skosovsky/evaly"
	"github.com/skosovsky/evaly/adapters/httpjson"
	"github.com/skosovsky/evaly/conformance"
	"github.com/skosovsky/evaly/internal/fixtures"
	"github.com/skosovsky/evaly/observation"
)

type workflowConfig = evaly.RunConfig[fixtures.WorkflowInput, fixtures.WorkflowOutput, int, *fixtures.WorkflowEnvironment]

func workflowHTTP(t *testing.T, c *workflowConfig, s *fixtures.WorkflowStore, mode string, incomplete bool) {
	t.Helper()
	handler, handlerErr := httpjson.NewHandler(
		fixtures.WorkflowInputCodec(),
		fixtures.WorkflowOutputCodec(),
		8192,
		func(ctx context.Context, i fixtures.WorkflowInput, trial httpjson.Trial) (httpjson.Invocation[fixtures.WorkflowOutput], error) {
			r, events, err := fixtures.WorkflowInvoke(ctx, i, trial.ID, s, mode)
			delivery := httpjson.EvidenceDelivery{Complete: true}
			if incomplete {
				delivery = httpjson.EvidenceDelivery{Reason: "connection_interrupted"}
				err = errors.New("partial host delivery")
			}
			return httpjson.Invocation[fixtures.WorkflowOutput]{
				Output:   r.Output,
				Usage:    r.Usage,
				Events:   events,
				Evidence: delivery,
			}, err
		},
	)
	if handlerErr != nil {
		t.Fatal(handlerErr)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c.Target = httpjson.Target[fixtures.WorkflowInput, fixtures.WorkflowOutput, *fixtures.WorkflowEnvironment]{
		URL:      server.URL,
		Client:   server.Client(),
		Input:    fixtures.WorkflowInputCodec(),
		Output:   fixtures.WorkflowOutputCodec(),
		MaxBytes: 8192,
		Fixture:  "workflow-v1",
		Reset:    "empty-account-v1",
	}
}
func TestWorkflowTransportSemantics(t *testing.T) {
	for _, mode := range []string{"refund", "alternative", "text-only", "tool-error", "effect-error"} {
		t.Run(mode, func(t *testing.T) {
			checkWorkflowTransportSemantics(
				// Arrange: both paths execute the same host tool implementation.
				t, &mode)
		},

		// Act.

		// Assert: actual state, rather than confident text, is observed.

		)
	}
}
func TestWorkflowTargetFailureConformance(t *testing.T) {
	for _, transport := range []string{"inprocess", "http"} {
		t.Run(transport, func(t *testing.T) { checkWorkflowTargetFailureConformance(t, &transport) })
	}
}
func TestWorkflowTransportDisconnectCannotProveAbsence(t *testing.T) {
	// Arrange: host commits action, but connection closes before delivery.
	c, s, _ := fixtures.WorkflowConfig("disconnect", "refund")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The fixture host uses its own namespace, independently verifying effect.
		s.Prepare("remote")
		_, _, _ = fixtures.WorkflowInvoke(r.Context(), fixtures.WorkflowInput{Amount: 100}, "remote", s, "refund")
		s.Cleanup("remote")
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}))
	defer server.Close()
	c.Target = httpjson.Target[fixtures.WorkflowInput, fixtures.WorkflowOutput, *fixtures.WorkflowEnvironment]{
		URL:      server.URL,
		Client:   server.Client(),
		Input:    fixtures.WorkflowInputCodec(),
		Output:   fixtures.WorkflowOutputCodec(),
		MaxBytes: 8192,
		Fixture:  "workflow-v1",
		Reset:    "empty-account-v1",
	}
	// Act.
	e, err := evaly.Run(context.Background(), c)
	// Assert: external effect is real; the client cannot infer absence or known usage.
	if err != nil {
		t.Fatal(err)
	}
	trial := e.Record().Trials[0]
	audit, _, _, _, _ := s.Snapshot()
	projection := savedWorkflow(t, c, e)
	grades := evaly.Assess(context.Background(), c.Graders, projection.View)
	if len(audit) != 1 || trial.Status != evaly.TargetError || trial.TargetUsage.Known ||
		evaly.CompleteFor(trial.Evidence, "tool") ||
		len(grades) != 2 ||
		grades[1].Status != evaly.InsufficientEvidence {
		t.Fatal(audit, trial)
	}
}
func TestWorkflowBudgetCancellationAndIsolation(t *testing.T) {
	for _, transport := range []string{"inprocess", "http"} {
		for _, stop := range []string{"budget", "cancel", "isolation"} {
			t.Run(transport+"-"+stop, func(t *testing.T) {
				checkWorkflowBudgetCancellationAndIsolation(t, &transport, &stop)
			},
			)
		}
	}
}

func savedWorkflow(
	t *testing.T,
	c workflowConfig,
	e evaly.Experiment,
) evaly.SavedView[fixtures.WorkflowInput, fixtures.WorkflowOutput, int] {
	t.Helper()
	cs, err := c.Dataset.CaseAt(0)
	if err != nil {
		t.Fatal(err)
	}
	s, err := evaly.SaveView(
		evaly.View[fixtures.WorkflowInput, fixtures.WorkflowOutput, int]{
			Case:     cs,
			Output:   fixtures.WorkflowOutput{},
			Evidence: e.Record().Trials[0].Evidence,
		},
		c.ProjectionRevision,
		fixtures.WorkflowInputCodec(),
		fixtures.WorkflowOutputCodec(),
		fixtures.WorkflowReferenceCodec(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// captureWorkflowProjection preserves the actual completed target output via the host projection port.
func captureWorkflowProjection(
	t *testing.T,
	c workflowConfig,
) (evaly.Experiment, evaly.SavedView[fixtures.WorkflowInput, fixtures.WorkflowOutput, int]) {
	t.Helper()
	var saved evaly.SavedView[fixtures.WorkflowInput, fixtures.WorkflowOutput, int]
	original := c.Project
	c.Project = func(ctx context.Context, cs evaly.Case[fixtures.WorkflowInput, int], out fixtures.WorkflowOutput, e evaly.EvidenceRecord) (evaly.View[fixtures.WorkflowInput, fixtures.WorkflowOutput, int], error) {
		view, err := original(ctx, cs, out, e)
		if err != nil {
			return view, err
		}
		saved, err = evaly.SaveView(
			view,
			c.ProjectionRevision,
			fixtures.WorkflowInputCodec(),
			fixtures.WorkflowOutputCodec(),
			fixtures.WorkflowReferenceCodec(),
		)
		return view, err
	}
	e, err := evaly.Run(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	view, err := saved.View()
	if err != nil || view.Output.Text != "Refund definitely completed" {
		t.Fatal(view, err)
	}
	return e, saved
}
func TestWorkflowArtifactAssessmentLineage(t *testing.T) {
	// Arrange: source target executes once and is then removed from scope.
	c, s, _ := fixtures.WorkflowConfig("published-workflow", "refund")
	e, saved := captureWorkflowProjection(t, c)
	original := e.Record()
	directory := t.TempDir()
	store, err := evaly.OpenFileStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = evaly.SaveExperiment(context.Background(), store, e); err != nil {
		t.Fatal(err)
	}
	if err = evaly.SaveSavedView(context.Background(), store, "workflow-view", saved); err != nil {
		t.Fatal(err)
	}
	// Act: reopen and rejudge only persisted projections.
	reopened, err := evaly.OpenFileStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := evaly.LoadExperiment(context.Background(), reopened, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	saved, err = evaly.LoadSavedView(
		context.Background(),
		reopened,
		"workflow-view",
		fixtures.WorkflowInputCodec(),
		fixtures.WorkflowOutputCodec(),
		fixtures.WorkflowReferenceCodec(),
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := evaly.Rescore(
		context.Background(),
		saved,
		fixtures.WorkflowGraders("external-balance-v1"),
		e.Revision(),
		"",
		"rescore",
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := evaly.Rescore(
		context.Background(),
		saved,
		fixtures.WorkflowGraders("external-balance-v2"),
		e.Revision(),
		first.Revision,
		"rescore",
	)
	if err != nil {
		t.Fatal(err)
	}
	// Assert: no target/lifecycle capability is passed to either rescore.
	_, _, calls, prepared, cleaned := s.Snapshot()
	if calls != 1 || prepared != 1 || cleaned != 1 || !reflect.DeepEqual(original, restored.Record()) ||
		second.Parent != first.Revision ||
		first.Revision == second.Revision ||
		second.Source != e.Revision() ||
		first.Grades[0].Revision.Rubric != "external-balance-v1" {
		t.Fatal(calls, prepared, cleaned, first, second)
	}
	baseline, _, _ := fixtures.WorkflowConfig("workflow-baseline", "text-only")
	candidate, _, _ := fixtures.WorkflowConfig("workflow-candidate", "alternative")
	b, cc, err := evaly.RunPaired(context.Background(), baseline, candidate, "workflow-pair")
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := evaly.Compare(
		b,
		cc,
		evaly.AssertionObjective{ID: "workflow-outcome", Revision: "1", Policy: "all"},
		fixtures.Gate(),
	)
	if err != nil || comparison.Verdict != evaly.GatePass {
		t.Fatal(comparison, err)
	}
}
func TestWorkflowOnlinePartialAssessment(t *testing.T) {
	// Arrange: observation has one paid grading unit for a two-grader plan.
	c, s, _ := fixtures.WorkflowConfig("workflow-online", "refund")
	e, saved := captureWorkflowProjection(t, c)
	budget, _ := evaly.NewMemoryBudget(1)
	worker, err := observation.Start(
		context.Background(),
		observation.Config[fixtures.WorkflowInput, fixtures.WorkflowOutput, int]{
			Capacity:    1,
			Concurrency: 1,
			Deadline:    time.Second,
			Clock:       observation.RealClock{},
			Graders:     c.Graders,
			Budget:      budget,
			GraderUnits: 1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Cancel(context.Background())
	obs, err := observation.New(
		"workflow-observation",
		e.Revision(),
		observation.Sampling{Rule: "all", Population: "offline-fixture", Window: "test"},
		saved,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Act.
	future, status := worker.Enqueue(obs)
	if status != "accepted" {
		t.Fatal(status)
	}
	result := <-future.Result
	// Assert.
	_, _, calls, prepared, cleaned := s.Snapshot()
	a := result.Assessment
	if a.State != "partial" || len(a.Grades) != 1 || len(a.Skipped) != 1 || a.Grades[0].Usage.Units != 1 ||
		a.Parent != e.Revision() ||
		evaly.ValidateAssessment(a) != nil ||
		calls != 1 ||
		prepared != 1 ||
		cleaned != 1 {
		t.Fatal(a, calls, prepared, cleaned)
	}
}
func checkWorkflowTransportSemantics(t *testing.T, mode *string) {
	t.Helper()
	// Arrange.
	var records []evaly.TrialRecord
	for _, transport := range []string{"inprocess", "http"} {
		c, s, err := fixtures.WorkflowConfig((*mode)+"-"+transport, (*mode))
		if err != nil {
			t.Fatal(err)
		}
		if transport == "http" {
			workflowHTTP(t, &c, s, (*mode), false)
		}

		// Act.
		e, err := evaly.Run(context.Background(), c)
		// Assert.
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, e.Record().Trials[0])

		expected := checkWorkflowStore(t, *mode, s)
		grades := records[len(records)-1].Grades
		if (*mode) == "tool-error" || (*mode) == "effect-error" {
			projection := savedWorkflow(t, c, e)
			grades = evaly.Assess(context.Background(), c.Graders, projection.View)
		}
		if len(grades) != 2 || grades[0].Status != evaly.Scored || len(grades[0].Assertions) != 1 ||
			grades[0].Assertions[0].Pass != expected {
			t.Fatal(grades, (*mode))
		}
	}
	a, b := records[0], records[1]
	if a.Status != b.Status || a.TargetUsage != b.TargetUsage || !reflect.DeepEqual(a.Grades, b.Grades) ||
		!reflect.DeepEqual(a.Evidence.Events, b.Evidence.Events) {
		t.Fatal("transport changed domain semantics", a, b)
	}
	if (*mode) == "effect-error" &&
		(a.Status != evaly.TargetError || a.TargetUsage.Units != 2 || len(a.Evidence.Events) != 3) {
		t.Fatal(a)
	}
}

func checkWorkflowTargetFailureConformance(t *testing.T, transport *string) {
	t.Helper()
	// Arrange.
	conformance.TargetFailures(
		t,
		func(fault conformance.TargetFailure) (conformance.TargetFault[fixtures.WorkflowInput, fixtures.WorkflowOutput, int, *fixtures.WorkflowEnvironment], error) {
			c, s, err := fixtures.WorkflowConfig("failure-"+(*transport)+"-"+string(fault), "effect-error")
			if err != nil {
				return conformance.TargetFault[fixtures.WorkflowInput, fixtures.WorkflowOutput, int, *fixtures.WorkflowEnvironment]{}, err
			}
			if (*transport) == "http" {
				workflowHTTP(t, &c, s, "effect-error", fault == conformance.IncompleteDelivery)
			} else if fault == conformance.IncompleteDelivery {
				underlying := c.Target
				c.Target = evaly.TargetFunc[fixtures.WorkflowInput, fixtures.WorkflowOutput, *fixtures.WorkflowEnvironment](
					func(ctx context.Context, i fixtures.WorkflowInput, tc evaly.TrialContext[*fixtures.WorkflowEnvironment]) (evaly.TargetResult[fixtures.WorkflowOutput], error) {
						r, e := underlying.Run(ctx, i, tc)
						tc.Evidence.MarkIncomplete("connection_interrupted")
						return r, e
					},
				)
			}
			return conformance.TargetFault[fixtures.WorkflowInput, fixtures.WorkflowOutput, int, *fixtures.WorkflowEnvironment]{
				Config:        c,
				ExpectedUsage: evaly.Usage{Known: true, Units: 2},
				Kind:          "tool",
				Verify: func(t *testing.T, e evaly.Experiment, _ error) {
					checkWorkflowFailureEffects(t, c, s, e, fault)
				},
			}, nil
		},
	)
}

func checkWorkflowBudgetCancellationAndIsolation(t *testing.T, transport *string, stop *string) {
	t.Helper()
	// Arrange.
	c, s, _ := fixtures.WorkflowConfig("bounded-"+(*transport)+"-"+(*stop), "refund")
	if (*transport) == "http" {
		workflowHTTP(t, &c, s, "refund", false)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if (*stop) == "budget" {
		c.Budget, _ = evaly.NewMemoryBudget(0)
		c.Plan.DispatchUnits = 2
	}
	if (*stop) == "cancel" {
		cancel()
	}
	if (*stop) == "isolation" {
		c.Plan.Repeats = 2
		c.Plan.Concurrency = 2
	}

	e, err := evaly.Run(ctx, c)

	audit, active, calls, prepared, cleaned := s.Snapshot()
	if active != 0 || prepared != cleaned {
		t.Fatal(active, prepared, cleaned)
	}
	if (*stop) == "isolation" {
		if err != nil || calls != 2 || len(audit) != 2 {
			t.Fatal(err, calls, audit)
		}
		for _, state := range audit {
			if state.Refunded != 100 {
				t.Fatal("namespace leaked prior action", state)
			}
		}
	} else if calls != 0 || len(audit) != 0 {
		t.Fatal("paid target dispatched after stop", calls, audit, e.Record(), err)
	}
}

func checkWorkflowStore(t *testing.T, mode string, s *fixtures.WorkflowStore) bool {
	t.Helper()
	audit, active, calls, prepared, cleaned := s.Snapshot()
	if active != 0 || calls != 1 || prepared != 1 || cleaned != 1 {
		t.Fatal(audit, active, calls, prepared, cleaned)
	}
	expected := mode == "refund" || mode == "alternative" || mode == "effect-error"
	if (len(audit) == 1) != expected {
		t.Fatal(audit, mode)
	}
	return expected
}

func checkWorkflowFailureEffects(
	t *testing.T,
	c evaly.RunConfig[fixtures.WorkflowInput, fixtures.WorkflowOutput, int, *fixtures.WorkflowEnvironment],
	s *fixtures.WorkflowStore,
	e evaly.Experiment,
	fault conformance.TargetFailure,
) {
	t.Helper()
	audit, active, calls, prepared, cleaned := s.Snapshot()
	if len(audit) != 1 || active != 0 || calls != 1 || prepared != 1 || cleaned != 1 {
		t.Fatal(audit, active, calls, prepared, cleaned)
	}
	for _, state := range audit {
		if state.Refunded != 100 {
			t.Fatal(state)
		}
	}
	projection := savedWorkflow(t, c, e)
	grades := evaly.Assess(context.Background(), c.Graders, projection.View)
	if fault == conformance.IncompleteDelivery &&
		(len(grades) != 2 || grades[1].Status != evaly.InsufficientEvidence) {
		t.Fatal("forbidden-action absence must remain unproved", e.Record())
	}
}
