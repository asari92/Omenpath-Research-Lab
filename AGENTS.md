# Omenpath Research Lab — Agent Map

## Sources of truth

1. [`00_FINAL_SPEC_v5.md`](00_FINAL_SPEC_v5.md) is the sole product/domain source of truth and wins every conflict.
2. [`01_AI_WORKLOG_CURRENT.md`](01_AI_WORKLOG_CURRENT.md) records actual implementation history and corrective passes.
3. [`02_IMPLEMENTATION_ROADMAP_TDD.md`](02_IMPLEMENTATION_ROADMAP_TDD.md) defines stage order and architecture boundaries.
4. Detailed stage or block plans (`03_STAGE_00_01_TDD_FOUNDATION.md`, `04_STAGE_02_PORTAL_CORE_TDD.md`, `05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md`, and successors) govern execution only for the current approved scope.
5. [`docs/requirements.md`](docs/requirements.md) catalogs requirement IDs; [`docs/traceability.md`](docs/traceability.md) maps them to tests and implementation.

Work only from the current detailed stage or block plan after checking it against the Final Spec. When the approved execution document covers a whole roadmap block, proceed stage-by-stage inside that block without waiting for additional approval, but do not begin the next block before the required user checkpoint. Do not change semantics of completed stages without evidence of a real conflict, defect, or required integration change.

## TDD and quality protocol

For each meaningful checkpoint: requirements → tests → observed RED → minimal implementation → focused GREEN. Preserve stage-level RED/GREEN evidence in Git history. Before completing every stage, run the full ordinary test suite and update traceability/worklog. At every block boundary, repeat the complete quality suite including the race detector and stop for user review.

Before completing every stage, run:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
```

At the block boundary, repeat those commands and additionally run:

```bash
go test -race -count=1 ./...
```

`GREEN` means the requirement is fully covered at its current promised boundary; `PARTIAL` means a tested helper/subset exists but required orchestration or another half remains for a later stage; `PLANNED` means no implemented, verified coverage yet. Keep [`01_AI_WORKLOG_CURRENT.md`](01_AI_WORKLOG_CURRENT.md) honest and current, and update [`docs/traceability.md`](docs/traceability.md) whenever coverage or implementation status changes.
