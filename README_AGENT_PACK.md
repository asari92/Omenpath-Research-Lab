# Omenpath Agent Pack

## Приоритет документов

1. `00_FINAL_SPEC_v5.md` — source of truth по продуктовой и domain-механике.
2. `02_IMPLEMENTATION_ROADMAP_TDD.md` — общий порядок реализации.
3. Детальные `STAGE_*` планы — инструкции ближайшего этапа.
4. `01_AI_WORKLOG_CURRENT.md` — честный журнал работы, который дополняется по мере реализации.

Если детальный план или реализация противоречат Final Spec, нельзя молча менять правило. Сначала фиксируется противоречие и принимается явное решение.

## Workflow

```text
Read Final Spec
→ Read Roadmap
→ Read current Stage Plan
→ Check contradictions
→ Write test
→ RED
→ Implement minimally
→ GREEN
→ Refactor
→ Run go test ./...
→ Run go test -race ./...
→ Update traceability
→ Add real AI Worklog notes
→ Stop at Stage DoD
```

Используем rolling-wave planning: весь roadmap определён заранее, а подробно расписываются ближайшие 1–2 этапа. После реализации этапа документы сверяются с Final Spec, затем детализируется следующий этап.

Перед сдачей обязательно проводится полный consistency audit: Final Spec, Requirement Catalog, Traceability, Stage Plans, Domain Models, DB Schema, REST/WS contracts, Tutorial, Frontend, Tests, AI Worklog, README и фактическая реализация.

- `04_STAGE_02_PORTAL_CORE_TDD.md` — детальный Stage 2: Portal Core via TDD.
