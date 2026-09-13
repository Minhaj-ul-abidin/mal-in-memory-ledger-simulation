package ledger

import "slices"

type Ledger struct {
	entries []Entry
	ccy     map[string]Currency
}

func New() *Ledger {
	return &Ledger{ccy: map[string]Currency{}}
}

func (l *Ledger) Open(account string, c Currency) {
	l.ccy[account] = c
}

func (l *Ledger) Currency(account string) Currency {
	c, ok := l.ccy[account]
	if !ok {
		panic("ledger: unknown account " + account)
	}
	return c
}

// Append is the only mutator. Seq counts from 1, so a zero Ref means no
// reference rather than a reference to the first entry.
func (l *Ledger) Append(e Entry) Entry {
	if code := l.Currency(e.Account).Code; e.Amount.Currency().Code != code {
		panic("ledger: " + e.Amount.Currency().Code + " entry on " + code + " account " + e.Account)
	}
	e.Seq = len(l.entries) + 1
	l.entries = append(l.entries, e)
	return e
}

// Closing sums the entries belonging to valueDay or earlier that the ledger had
// learned about by asOf. Both dates are needed: the Day 2 balance as known on
// Day 5 is a different figure from the same balance as known on Day 6.
func (l *Ledger) Closing(account string, valueDay, asOf Day) Money {
	total := Zero(l.Currency(account))
	for _, e := range l.entries {
		if e.Account != account || !e.Kind.AffectsBalance() {
			continue
		}
		if e.ValueDate > valueDay || e.BookedOn > asOf {
			continue
		}
		total = total.Add(e.Amount)
	}
	return total
}

// Entries returns a copy, so the store stays append-only from the outside.
func (l *Ledger) Entries() []Entry { return slices.Clone(l.entries) }

// Assessed is the net of what day closes have already booked for this account
// and value day. Restatement diffs against it rather than re-posting a figure.
func (l *Ledger) Assessed(account string, day Day, a Assessment) Money {
	total := Zero(l.Currency(account))
	for _, e := range l.entries {
		if e.Account != account || e.ValueDate != day {
			continue
		}
		if got, ok := e.Kind.assessment(); ok && got == a {
			total = total.Add(e.Amount)
		}
	}
	return total
}
