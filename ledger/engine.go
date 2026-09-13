package ledger

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

		for _, a := range accounts {
			assess(l, a.ID, d, d)
		}

		for _, a := range accounts {
			day.Balances = append(day.Balances, AccountClosing{
				Account: a.ID,
				Closing: l.Closing(a.ID, d, d),
			})
			if fee := l.Assessed(a.ID, d, FeeAssessment); !fee.IsZero() {
				day.Fees = append(day.Fees, a.ID+" "+fee.Neg().Display())
			}
		}
		for _, h := range holds {
			day.Auths = append(day.Auths, h.ID+" "+h.State.String()+" "+h.Amount.Display())
		}
		r.Days = append(r.Days, day)
	}
	return r
}

// Reversals fall through until the next step.
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
