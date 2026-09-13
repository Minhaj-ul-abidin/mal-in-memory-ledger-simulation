# In-memory account ledger

Design and entity at design.md

## Running it

```
go run ./cmd/replay
go test ./...
```

`go test ./...` reports one failure and that is expected.
`TestEveryDeclineIsJustifiableFromTheRestatedLedger` in
`ledger/restatement_test.go` is the failing test the brief asks for. It is aimed
at this design, not at a wrong criterion, and what it reveals is annotated above
it. Every other test passes.

## The other documents

| File | What it holds |
|---|---|
| `AMBIGUITIES.md` | every gap in the spec and how it was resolved |
| `REJECTED.md` | criteria refused, and approaches dropped |
| `NUMBERS.md` | every constant, why that value and not half it |
| `WORKLOG.md` | timestamped build log |

Read `AMBIGUITIES.md` entry 4 first. The backdating policy decides the answer to
half the acceptance criteria.
