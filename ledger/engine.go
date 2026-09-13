package ledger

const window = 6

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

	var r Report
	for d := Day(1); d <= window; d++ {
		for _, e := range events {
			if e.BookedOn == d {
				book(l, e)
			}
		}

		day := DayReport{Day: d}
		for _, a := range accounts {
			day.Balances = append(day.Balances, AccountClosing{
				Account: a.ID,
				Closing: l.Closing(a.ID, d, d),
			})
		}
		r.Days = append(r.Days, day)
	}
	return r
}

// Authorizations, settlements and reversals fall through until holds exist.
func book(l *Ledger, e Event) {
	switch e.Type {
	case CreditEvent:
		post(l, e, Credit, e.Amount)
	case DebitEvent:
		post(l, e, Debit, e.Amount.Neg())
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
