# In-memory account ledger

Design and entity at design.md

## Running it

Go 1.25 or later (see `go.mod`), standard library only. From the repository root:

```
go run ./cmd/replay     # replay the six days and print the report
go test ./...           # run the test suite
go vet ./...
```

To run a single test and see its output:

```
go test -run TestCriterion2_E7CausesThreeFeesNotOne -v ./ledger/
```

`go test ./...` reports one failure and that is expected.
`TestEveryDeclineIsJustifiableFromTheRestatedLedger` in
`ledger/restatement_test.go` is the failing test the brief asks for. It is aimed
at this design, not at a wrong criterion, and what it reveals is annotated above
it. Every other test passes.

## Reading the output

The replay prints one block per day, then a restated grid. Day 5 is the one worth
reading closely:

```
Day 5
  ACC-001   closing  AED -230.00
  ACC-002   closing  BHD 10.000
  fees      ACC-001 fee AED -25.00 (value date Day 2)
            ACC-001 fee AED -25.00 (value date Day 4)
            ACC-001 fee AED -25.00 (value date Day 5)
  interest  ACC-001 interest adjusted AED -0.10 (value date Day 2)
            ACC-001 interest adjusted AED -0.26 (value date Day 3)
            ACC-001 interest adjusted AED -0.19 (value date Day 4)
            ACC-002 interest accrued BHD 0.004 (value date Day 5)
  auths     Auth-A PARTIALLY_SETTLED AED 200.00
            Auth-B DECLINED AED 90.00
  errors    E8: authorization Auth-B declined, insufficient available balance
```

| Line | What it shows |
|---|---|
| `closing` | The balance at the end of that day, as the ledger knew it that day, fees included. Interest accruals do not count until they capitalize. |
| `fees` | Fees charged and reversed that day. The value date in brackets is the day each one belongs to, so a fee booked today for an earlier day is a restatement. |
| `interest` | Accruals and adjustments booked that day, and on Day 6 the capitalized credit. Only the capitalized credit moves `closing`. |
| `auths` | Every authorization so far, with its state at the end of that day and the amount originally held. |
| `errors` | Events the ledger refused that day, and why. |

`none` means nothing of that kind happened that day. Each block lists what was
booked on that day, so Day 5 shows three fees because a debit booked on Day 5 but
belonging to Day 2 reopened Days 2 to 5, not because Day 5 was charged three times.

The **Restated at close of Day 6** grid repeats every day's closing as the ledger
knows it at the end of the window, once every backdated entry has landed. A day can
differ between its own block and the grid: Day 5 closed at AED -230.00 on the day
and stands at AED 465.00 once the Day 6 reversal is known. Both are correct; they
answer different questions.

## The other documents

| File | What it holds |
|---|---|
| `AMBIGUITIES.md` | every gap in the spec and how it was resolved |
| `REJECTED.md` | criteria refused, and approaches dropped |
| `NUMBERS.md` | every constant, why that value and not half it |
| `WORKLOG.md` | timestamped build log |

Read `AMBIGUITIES.md` entry 4 first. The backdating policy decides the answer to
half the acceptance criteria.
