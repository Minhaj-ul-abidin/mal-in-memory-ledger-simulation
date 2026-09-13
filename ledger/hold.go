package ledger

// Expired and Released are never reached in this window: the spec gives no
// expiry horizon and the stream sends no merchant void. They are modelled
// because an authorization that never settles still has to end somewhere.
type HoldState int

const (
	HoldActive HoldState = iota
	HoldSettled
	HoldPartiallySettled
	HoldDeclined
	HoldExpired
	HoldReleased
)

var holdStateNames = [...]string{
	HoldActive:           "ACTIVE",
	HoldSettled:          "SETTLED",
	HoldPartiallySettled: "PARTIALLY_SETTLED",
	HoldDeclined:         "DECLINED",
	HoldExpired:          "EXPIRED",
	HoldReleased:         "RELEASED",
}

func (s HoldState) String() string { return holdStateNames[s] }

// A declined authorization is still recorded. The decision is never restated, so
// this record is the only evidence that it was made or what it was made against.
type Hold struct {
	ID       string
	Account  string
	Amount   Money
	State    HoldState
	OpenedOn Day
}

type Holds []*Hold

func (hs Holds) Find(id string) *Hold {
	for _, h := range hs {
		if h.ID == id {
			return h
		}
	}
	return nil
}

// ActiveTotal is what is held against an account right now. A declined hold
// never reduced anything, and a settled one has stopped.
func (hs Holds) ActiveTotal(account string, c Currency) Money {
	total := Zero(c)
	for _, h := range hs {
		if h.Account == account && h.State == HoldActive {
			total = total.Add(h.Amount)
		}
	}
	return total
}

// Available is the ledger balance less everything held against it. A hold does
// not move the ledger balance, only what can be spent from it.
func (l *Ledger) Available(hs Holds, account string, asOf Day) Money {
	return l.Closing(account, asOf, asOf).Sub(hs.ActiveTotal(account, l.Currency(account)))
}
