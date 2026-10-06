# Runnable end-to-end quickstart

Requires Go 1.27.1. From the repository root:

```sh
env GOCACHE=/tmp/evaly-go-build go test . -run '^ExampleRun$' -v
```

The complete runnable source is [ExampleRun](../quickstart_test.go). It uses only
the standard library and evaly. Go executes the example and checks its output:

```text
quality pass: true
health pass: true
gate pass: true
```

1. Seal a typed integer dataset with an explicit reference and stock codecs.
2. Supply lifecycle prepare/reset/cleanup and a typed target, with stable revisions.
3. Capture an operation event through a policy that retains no payload. The host
   deliberately permits the integer input/output/reference in its projection.
4. Grade the fresh permitted view using an exact assertion.
5. Run one bounded trial and inspect its retained result.
6. Apply an explicit host gate requiring scored quality, healthy target/cleanup,
   complete relevant evidence and no accounting error.

Changing the target to produce a wrong result fails quality; a successful run is
not itself a quality pass. This simple host gate does not authorize deployment or
prove live provider accuracy. For baseline/candidate thresholds and case-level
uncertainty use Compare with an explicit Objective/GatePolicy, demonstrated by
the existing [calculation example](../examples/calculation/main.go). Stateful
effects and offline/online assessments are covered by the existing
[integration example](../examples/integration/main.go).
