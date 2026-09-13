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
	var holds Holds

	var r Report
	for d := Day(1); d <= window; d++ {
		day := DayReport{Day: d}
		for _, e := range events {
			if e.BookedOn == d {
				book(l, &holds, e, &day)
			}
		}

		for _, a := range accounts {
			day.Balances = append(day.Balances, AccountClosing{
				Account: a.ID,
				Closing: l.Closing(a.ID, d, d),
			})
		}
		for _, h := range holds {
			day.Auths = append(day.Auths, h.ID+" "+h.State.String()+" "+h.Amount.Display())
		}
		r.Days = append(r.Days, day)
	}
	return r
}

// Settlements and reversals fall through until the next step.
func book(l *Ledger, holds *Holds, e Event, day *DayReport) {
	switch e.Type {
	case CreditEvent:
		post(l, e, Credit, e.Amount)
	case DebitEvent:
		post(l, e, Debit, e.Amount.Neg())
	case AuthorizationEvent:
		authorize(l, holds, e, day)
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
