package ledger

type EventType int

const (
	CreditEvent EventType = iota
	DebitEvent
	AuthorizationEvent
	SettlementEvent
	ReversalEvent
)

var eventTypeNames = [...]string{
	CreditEvent:        "CREDIT",
	DebitEvent:         "DEBIT",
	AuthorizationEvent: "AUTHORIZATION",
	SettlementEvent:    "SETTLEMENT",
	ReversalEvent:      "REVERSAL",
}

func (t EventType) String() string { return eventTypeNames[t] }

// An Event is an instruction, not a record. An authorization writes no entry at
// all, and a split credit writes several.
type Event struct {
	ID        string
	Type      EventType
	BookedOn  Day
	ValueDate Day
	Account   string
	Amount    Money // magnitude; Type carries the direction
	Auth      string
	Reverses  string
	Parts     int // instalments to split Amount into; 0 means a single entry
}

// Stream is the event stream in the order given. List order is not booking
// order: E10 is listed last but books on Day 5, before E9 on Day 6.
func Stream() []Event {
	return []Event{
		{ID: "E1", Type: CreditEvent, BookedOn: 1, ValueDate: 1, Account: "ACC-001", Amount: Amount(AED, "1200.00")},
		{ID: "E2", Type: DebitEvent, BookedOn: 1, ValueDate: 1, Account: "ACC-001", Amount: Amount(AED, "950.00")},
		{ID: "E3", Type: AuthorizationEvent, BookedOn: 2, ValueDate: 2, Account: "ACC-001", Amount: Amount(AED, "200.00"), Auth: "Auth-A"},
		{ID: "E4", Type: CreditEvent, BookedOn: 3, ValueDate: 3, Account: "ACC-001", Amount: Amount(AED, "400.00")},
		{ID: "E5", Type: SettlementEvent, BookedOn: 4, ValueDate: 4, Account: "ACC-001", Amount: Amount(AED, "185.00"), Auth: "Auth-A"},
		{ID: "E6", Type: SettlementEvent, BookedOn: 4, ValueDate: 4, Account: "ACC-001", Amount: Amount(AED, "180.00"), Auth: "Auth-Z"},
		{ID: "E7", Type: DebitEvent, BookedOn: 5, ValueDate: 2, Account: "ACC-001", Amount: Amount(AED, "620.00")},
		{ID: "E8", Type: AuthorizationEvent, BookedOn: 5, ValueDate: 5, Account: "ACC-001", Amount: Amount(AED, "90.00"), Auth: "Auth-B"},
		{ID: "E9", Type: ReversalEvent, BookedOn: 6, ValueDate: 2, Account: "ACC-001", Amount: Zero(AED), Reverses: "E7"},
		{ID: "E10", Type: CreditEvent, BookedOn: 5, ValueDate: 5, Account: "ACC-002", Amount: Amount(BHD, "10.000"), Parts: 3},
	}
}
