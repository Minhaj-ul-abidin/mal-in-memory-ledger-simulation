A six-day replay of a fixed event stream over two accounts, ACC-001 (AED) and
ACC-002 (BHD), both opening at zero. Go, standard library only. No persistence,
no web layer, no database.

The replay prints, per day: closing ledger balance, fee assessments,
authorization states, and errors.

## Entities

**Money, Currency** — `ledger/money.go` *(written)*
An amount is a signed `int64` in the currency's minor units, 1/100 AED and
1/1000 BHD. Precision belongs to the currency, not to the arithmetic. No
`float64` anywhere in the ledger.

**Entry, EntryKind, Day** — `ledger/entry.go`
The unit of record. Every movement, fee, correction and accrual in the ledger is
an Entry. It carries two dates that are allowed to disagree: `BookedOn`, the day
the ledger learned about it, and `ValueDate`, the day it belongs to. EntryKind
separates a customer movement from a fee, a reversal, an accrual and the Day 6
capitalization. Accruals are memo entries, recorded but not moving the balance.

**Ledger** — `ledger/ledger.go`
The store. Holds every Entry and nothing else, and appending is the only way to
change it. Answers balance questions, which take both dates: the Day 2 balance as
known on Day 5 and the same balance as known on Day 6 are different questions and
both have to be answerable.

**Hold** — `ledger/hold.go`
An authorization sitting against an account. Reduces available balance without
moving the ledger balance. Carries the states an authorization can end in,
including the ones this stream never reaches.

**Engine** — `ledger/engine.go`
Owns the replay loop, the day close and the restatement pass. The only component
that decides anything.

**Stream** — `ledger/stream.go`
The ten events as data, in order.

**Report** — `ledger/report.go`
Turns a finished replay into the per-day output.

## The algorithm

For each of the six days:

1. **Book** every event booked that day, in stream order. Authorizations are
   decided against available balance at the moment they arrive. Authorizations
   are real-time; fees are an end-of-day batch.

2. **Reassess.** If anything booked today is value-dated earlier, reopen every
   day from that earliest value date forward and assess each one again against
   what is known now. Otherwise assess today alone.

3. **Capitalize**, on Day 6 only: one interest credit carrying the sum of the
   accruals.

Reassessing a day works out what should have been booked, compares it to what was
booked, and appends the difference as a new entry. Nothing already in the ledger
is edited or removed. The pass runs in ascending day order, which is what lets it
finish in one sweep.

Two rules inside the assessment carry most of the design. A day's own fee is left
out of the test that triggers it, or an overdrawn day would fee itself without
end. Fees from earlier days stay in, because they are ordinary debits with a
value date and they carry forward like anything else.