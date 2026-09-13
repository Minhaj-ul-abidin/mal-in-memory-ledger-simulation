package ledger

import (
	"fmt"
	"slices"
)

const window = 6

// 0.04% per day, held as a pair so the interest never leaves integer arithmetic.
const interestNum, interestDen = 4, 10000

// The AED fee is given. The BHD fee is the same economic penalty carried across
// two fixed dollar pegs; see NUMBERS.md. It is never reached by this stream.
func overdraftFee(c Currency) Money {
	switch c.Code {
	case AED.Code:
		return Amount(AED, "25.00")
	case BHD.Code:
		return Amount(BHD, "2.560")
	}
	panic("ledger: no overdraft fee defined for " + c.Code)
}

// assess brings one value day's fee and interest into line with what is known at
// asOf, posting the difference rather than the figure. Called with day == asOf it
// is an ordinary day close; called over a range it is restatement.
func assess(l *Ledger, account string, day, asOf Day) {
	c := l.Currency(account)

	// A day's own fee is excluded from the balance that triggers it, or an
	// overdrawn day would fee itself without end. Earlier days' fees stay in.
	standing := l.Assessed(account, day, FeeAssessment)
	trigger := l.Closing(account, day, asOf).Sub(standing)

	target := Zero(c)
	if trigger.IsNegative() {
		target = overdraftFee(c).Neg()
	}
	if delta := target.Sub(standing); !delta.IsZero() {
		kind := Fee
		if delta.IsPositive() {
			kind = FeeReversal
		}
		l.Append(Entry{Account: account, Kind: kind, Amount: delta, BookedOn: asOf, ValueDate: day})
	}

	accrued := l.Assessed(account, day, InterestAssessment)
	earned := Zero(c)
	if base := trigger.Add(target); base.IsPositive() {
		earned = base.Rate(interestNum, interestDen)
	}
	if delta := earned.Sub(accrued); !delta.IsZero() {
		kind := Accrual
		if !accrued.IsZero() {
			kind = AccrualAdjustment
		}
		l.Append(Entry{Account: account, Kind: kind, Amount: delta, BookedOn: asOf, ValueDate: day})
	}
}

var accounts = []struct {
	ID  string
	Ccy Currency
}{
	{"ACC-001", AED},
	{"ACC-002", BHD},
}

func Replay(events []Event) Report {
	l := New()
	for _, a := range accounts {
		l.Open(a.ID, a.Ccy)
	}
	var holds Holds

	var r Report
	for d := Day(1); d <= window; d++ {
		day := DayReport{Day: d}
		for _, e := range events {
			if e.BookedOn == d {
				book(l, &holds, e, &day)
			}
		}

		// Ascending from the earliest day today's bookings belong to. Day k reads
		// only days <= k, so by the time the loop reaches k+1 the day-k delta is
		// already posted and one forward pass is enough.
		for _, a := range accounts {
			for k := reopenFrom(l, a.ID, d); k <= d; k++ {
				assess(l, a.ID, k, d)
			}
		}

		if d == window {
			for _, a := range accounts {
				capitalize(l, a.ID, d)
			}
		}

		for _, a := range accounts {
			day.Balances = append(day.Balances, AccountClosing{
				Account: a.ID,
				Closing: l.Closing(a.ID, d, d),
			})
		}
		day.Fees = bookedOn(l, d, Fee, FeeReversal)
		day.Interest = bookedOn(l, d, Accrual, AccrualAdjustment, Capitalization)
		for _, h := range holds {
			day.Auths = append(day.Auths, h.ID+" "+h.State.String()+" "+h.Amount.Display())
		}
		r.Days = append(r.Days, day)
	}

	// The same days again, as they stand once every backdated entry has landed.
	for d := Day(1); d <= window; d++ {
		rd := RestatedDay{Day: d}
		for _, a := range accounts {
			rd.Balances = append(rd.Balances, AccountClosing{
				Account: a.ID,
				Closing: l.Closing(a.ID, d, window),
			})
		}
		r.Restated = append(r.Restated, rd)
	}
	return r
}

// reopenFrom is the earliest day that anything booked today belongs to. Called
// before assessing, so it sees only booked events and not the deltas to come.
func reopenFrom(l *Ledger, account string, today Day) Day {
	from := today
	for _, e := range l.Entries() {
		if e.Account == account && e.BookedOn == today && e.ValueDate < from {
			from = e.ValueDate
		}
	}
	return from
}

// capitalize posts one credit carrying the sum of the rounded daily accruals.
// Building the total out of them is what makes them sum to it exactly; deriving
// it independently would leave a remainder to reconcile away.
//
// It runs after the last assessment, so the capitalized credit never lands in a
// trigger and interest never earns interest.
func capitalize(l *Ledger, account string, day Day) {
	total := Zero(l.Currency(account))
	for k := Day(1); k <= day; k++ {
		total = total.Add(l.Assessed(account, k, InterestAssessment))
	}
	if total.IsZero() {
		return
	}
	l.Append(Entry{Account: account, Kind: Capitalization, Amount: total, BookedOn: day, ValueDate: day})
}

var entryLabels = map[EntryKind]string{
	Fee:               "fee",
	FeeReversal:       "fee reversed",
	Accrual:           "interest accrued",
	AccrualAdjustment: "interest adjusted",
	Capitalization:    "interest capitalized",
}

// Assessments are reported on the day they are booked, carrying the day they
// belong to. A restatement assesses an earlier day today, and that has to show.
func bookedOn(l *Ledger, today Day, kinds ...EntryKind) []string {
	var out []string
	for _, e := range l.Entries() {
		if e.BookedOn != today || !slices.Contains(kinds, e.Kind) {
			continue
		}
		out = append(out, fmt.Sprintf("%s %s %s (value date Day %d)",
			e.Account, entryLabels[e.Kind], e.Amount.Display(), e.ValueDate))
	}
	return out
}

func book(l *Ledger, holds *Holds, e Event, day *DayReport) {
	switch e.Type {
	case CreditEvent:
		post(l, e, Credit, e.Amount)
	case DebitEvent:
		post(l, e, Debit, e.Amount.Neg())
	case AuthorizationEvent:
		authorize(l, holds, e, day)
	case SettlementEvent:
		settle(l, holds, e, day)
	case ReversalEvent:
		reverse(l, e, day)
	}
}

// A reversal is a new contra entry carrying the original's value date and a
// reference to it. The original is left exactly as it was. One event can have
// posted several entries, and reversing it reverses all of them.
func reverse(l *Ledger, e Event, day *DayReport) {
	var found bool
	for _, o := range l.Entries() {
		if o.Source != e.Reverses {
			continue
		}
		found = true
		l.Append(Entry{
			Source:    e.ID,
			Account:   o.Account,
			Kind:      Reversal,
			Amount:    o.Amount.Neg(),
			BookedOn:  e.BookedOn,
			ValueDate: o.ValueDate,
			Ref:       o.Seq,
		})
	}
	if !found {
		day.Errors = append(day.Errors,
			e.ID+": nothing to reverse, "+e.Reverses+" is not in the ledger")
	}
}

// An authorization is decided against available balance at the moment it
// arrives. Authorizations are real-time; fees are an end-of-day batch.
func authorize(l *Ledger, holds *Holds, e Event, day *DayReport) {
	h := &Hold{ID: e.Auth, Account: e.Account, Amount: e.Amount, OpenedOn: e.BookedOn}
	if l.Available(*holds, e.Account, e.BookedOn).Sub(e.Amount).IsNegative() {
		h.State = HoldDeclined
		day.Errors = append(day.Errors,
			e.ID+": authorization "+e.Auth+" declined, insufficient available balance")
	} else {
		h.State = HoldActive
	}
	*holds = append(*holds, h)
}

// A settlement moves money only against a hold that is open and large enough.
// Every rejection is recorded and nothing leaves the account.
func settle(l *Ledger, holds *Holds, e Event, day *DayReport) {
	h := holds.Find(e.Auth)
	switch {
	case h == nil:
		day.Errors = append(day.Errors,
			e.ID+": settlement references unknown authorization "+e.Auth+", rejected")
		return
	case h.State != HoldActive:
		day.Errors = append(day.Errors,
			e.ID+": authorization "+e.Auth+" is already "+h.State.String()+", rejected")
		return
	case e.Amount.Sub(h.Amount).IsPositive():
		day.Errors = append(day.Errors,
			e.ID+": settlement "+e.Amount.Display()+" exceeds the "+h.Amount.Display()+" hold on "+e.Auth+", rejected")
		return
	}

	post(l, e, Debit, e.Amount.Neg())
	if e.Amount.Equal(h.Amount) {
		h.State = HoldSettled
	} else {
		h.State = HoldPartiallySettled
	}
}

func post(l *Ledger, e Event, k EntryKind, amount Money) {
	parts := []Money{amount}
	if e.Parts > 1 {
		parts = Split(amount, e.Parts)
	}
	for _, p := range parts {
		l.Append(Entry{
			Source:    e.ID,
			Account:   e.Account,
			Kind:      k,
			Amount:    p,
			BookedOn:  e.BookedOn,
			ValueDate: e.ValueDate,
		})
	}
}
