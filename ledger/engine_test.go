package ledger

import "testing"

// One event, three entries. Independently rounded parts would put 0.002 BHD in
// the ledger that no event moved.
func TestBookingSplitsInstalments(t *testing.T) {
	l := New()
	l.Open("ACC-002", BHD)
	book(l, Event{
		ID: "E10", Type: CreditEvent, BookedOn: 5, ValueDate: 5,
		Account: "ACC-002", Amount: Amount(BHD, "10.000"), Parts: 3,
	})

	entries := l.Entries()
	if len(entries) != 3 {
		t.Fatalf("booked %d entries, want 3", len(entries))
	}

	want := []string{"3.333", "3.333", "3.334"}
	parts := make([]Money, len(entries))
	for i, e := range entries {
		parts[i] = e.Amount
		if e.Amount.String() != want[i] {
			t.Errorf("instalment %d = %s, want %s", i+1, e.Amount, want[i])
		}
	}
	if total := Sum(BHD, parts...); !total.Equal(Amount(BHD, "10.000")) {
		t.Errorf("instalments sum to %s, want exactly 10.000", total)
	}
}

// A debit carries its direction in the entry, and a backdated event keeps both
// dates: booking E7 on Day 5 must leave it belonging to Day 2.
func TestBookingNegatesDebitAndKeepsBothDates(t *testing.T) {
	l := New()
	l.Open("ACC-001", AED)
	book(l, Event{
		ID: "E7", Type: DebitEvent, BookedOn: 5, ValueDate: 2,
		Account: "ACC-001", Amount: Amount(AED, "620.00"),
	})

	e := l.Entries()[0]
	if want := Amount(AED, "-620.00"); !e.Amount.Equal(want) {
		t.Errorf("amount = %s, want %s", e.Amount, want)
	}
	if e.BookedOn != 5 || e.ValueDate != 2 {
		t.Errorf("dates = booked %d, valued %d; want booked 5, valued 2", e.BookedOn, e.ValueDate)
	}
}
