# Omenpath Research Lab — Agent Map

## Sources of truth

1. [`00_FINAL_SPEC_v5.md`](00_FINAL_SPEC_v5.md) is the sole product/domain source of truth and wins every conflict.
2. [`01_AI_WORKLOG_CURRENT.md`](01_AI_WORKLOG_CURRENT.md) records actual implementation history and corrective passes.
3. [`02_IMPLEMENTATION_ROADMAP_TDD.md`](02_IMPLEMENTATION_ROADMAP_TDD.md) defines stage order and architecture boundaries.
4. Detailed stage plans (`03_STAGE_00_01_TDD_FOUNDATION.md`, `04_STAGE_02_PORTAL_CORE_TDD.md`, `05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md`, and successors) govern execution only for the current stage.
5. [`docs/requirements.md`](docs/requirements.md) catalogs requirement IDs; [`docs/traceability.md`](docs/traceability.md) maps them to tests and implementation.

Work only from the current detailed stage plan after checking it against the Final Spec. Do not begin the next stage in the same pass. Do not change semantics of completed stages without evidence of a real conflict, defect, or required integration change.

## TDD and quality protocol

For each checkpoint: requirements → tests → observed RED → minimal implementation → GREEN → full tests → race detector → traceability/worklog update → commit. Preserve RED/GREEN evidence in Git history.

Before completing a checkpoint or stage, run:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

`GREEN` means the requirement is fully covered at its current promised boundary; `PARTIAL` means a tested helper/subset exists but required orchestration or another half remains for a later stage; `PLANNED` means no implemented, verified coverage yet. Keep [`01_AI_WORKLOG_CURRENT.md`](01_AI_WORKLOG_CURRENT.md) honest and current, and update [`docs/traceability.md`](docs/traceability.md) whenever coverage or implementation status changes.
