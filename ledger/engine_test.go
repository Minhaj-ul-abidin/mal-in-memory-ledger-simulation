package ledger

import "testing"

// One event, three entries. Independently rounded parts would put 0.002 BHD in
// the ledger that no event moved.
func TestBookingSplitsInstalments(t *testing.T) {
	l := New()
	l.Open("ACC-002", BHD)
	var holds Holds
	day := DayReport{Day: 5}
	book(l, &holds, Event{
		ID: "E10", Type: CreditEvent, BookedOn: 5, ValueDate: 5,
		Account: "ACC-002", Amount: Amount(BHD, "10.000"), Parts: 3,
	}, &day)

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
	var holds Holds
	day := DayReport{Day: 5}
	book(l, &holds, Event{
		ID: "E7", Type: DebitEvent, BookedOn: 5, ValueDate: 2,
		Account: "ACC-001", Amount: Amount(AED, "620.00"),
	}, &day)

	e := l.Entries()[0]
	if want := Amount(AED, "-620.00"); !e.Amount.Equal(want) {
		t.Errorf("amount = %s, want %s", e.Amount, want)
	}
	if e.BookedOn != 5 || e.ValueDate != 2 {
		t.Errorf("dates = booked %d, valued %d; want booked 5, valued 2", e.BookedOn, e.ValueDate)
	}
}

func ledgerAt(t *testing.T, day Day, amount string) *Ledger {
	t.Helper()
	l := New()
	l.Open("ACC-001", AED)
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, amount), BookedOn: day, ValueDate: day})
	return l
}

func TestDayCloseFeeAndInterest(t *testing.T) {
	for _, tc := range []struct {
		balance, fee, accrual, why string
	}{
		{"250.00", "0.00", "0.10", "positive, exact"},
		{"650.00", "0.00", "0.26", "positive, exact"},
		{"465.00", "0.00", "0.19", "0.186 rounds half-up"},
		{"5.00", "0.00", "0.00", "0.002 rounds away to nothing"},
		{"0.00", "0.00", "0.00", "zero is neither negative nor positive"},
		{"-155.00", "-25.00", "0.00", "negative takes the fee and earns nothing"},
	} {
		t.Run(tc.why, func(t *testing.T) {
			l := ledgerAt(t, 5, tc.balance)
			assess(l, "ACC-001", 5, 5)

			if got, want := l.Assessed("ACC-001", 5, FeeAssessment), Amount(AED, tc.fee); !got.Equal(want) {
				t.Errorf("fee = %s, want %s", got, want)
			}
			if got, want := l.Assessed("ACC-001", 5, InterestAssessment), Amount(AED, tc.accrual); !got.Equal(want) {
				t.Errorf("accrual = %s, want %s", got, want)
			}
		})
	}
}

// A day's own fee is excluded from the balance that triggers it. Without that,
// re-closing an overdrawn day charges it again, and again.
func TestOwnFeeDoesNotTriggerItself(t *testing.T) {
	l := ledgerAt(t, 5, "-155.00")
	assess(l, "ACC-001", 5, 5)
	assess(l, "ACC-001", 5, 5)
	assess(l, "ACC-001", 5, 5)

	if got, want := l.Assessed("ACC-001", 5, FeeAssessment), Amount(AED, "-25.00"); !got.Equal(want) {
		t.Fatalf("fee after three closes = %s, want %s", got, want)
	}
	if got, want := l.Closing("ACC-001", 5, 5), Amount(AED, "-180.00"); !got.Equal(want) {
		t.Errorf("closing = %s, want %s", got, want)
	}
}

// Fees from earlier days are ordinary debits and carry forward into later
// triggers. This is what puts Day 4 at -180.00 rather than -155.00.
func TestEarlierFeesCarryIntoLaterTriggers(t *testing.T) {
	l := ledgerAt(t, 1, "-10.00")
	assess(l, "ACC-001", 1, 1)
	assess(l, "ACC-001", 2, 2)

	if got, want := l.Assessed("ACC-001", 2, FeeAssessment), Amount(AED, "-25.00"); !got.Equal(want) {
		t.Errorf("day 2 fee = %s, want %s", got, want)
	}
	if got, want := l.Closing("ACC-001", 2, 2), Amount(AED, "-60.00"); !got.Equal(want) {
		t.Errorf("closing = %s, want %s: -10.00 less two fees", got, want)
	}
}

// A backdated debit reopens its own day and every day after it. Three of those
// close negative so three fees stand, and Day 3 escapes: the Day 3 credit clears
// it even after the Day 2 fee has been taken out of it. This is criterion 2.
func TestBackdatedEntryReopensEveryDayAfterIt(t *testing.T) {
	l := New()
	l.Open("ACC-001", AED)

	// Days 1 to 4, closed with what was known at the time.
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "1200.00"), BookedOn: 1, ValueDate: 1})
	l.Append(Entry{Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-950.00"), BookedOn: 1, ValueDate: 1})
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "400.00"), BookedOn: 3, ValueDate: 3})
	l.Append(Entry{Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-185.00"), BookedOn: 4, ValueDate: 4})
	for d := Day(1); d <= 4; d++ {
		assess(l, "ACC-001", d, d)
	}
	if got, want := l.Assessed("ACC-001", 2, InterestAssessment), Amount(AED, "0.10"); !got.Equal(want) {
		t.Fatalf("day 2 accrued %s before the backdated debit, want %s", got, want)
	}

	// E7 arrives on Day 5 belonging to Day 2.
	l.Append(Entry{Source: "E7", Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-620.00"), BookedOn: 5, ValueDate: 2})
	if got := reopenFrom(l, "ACC-001", 5); got != 2 {
		t.Fatalf("reopenFrom = %d, want 2", got)
	}
	for k := reopenFrom(l, "ACC-001", 5); k <= 5; k++ {
		assess(l, "ACC-001", k, 5)
	}

	for _, tc := range []struct {
		day               Day
		fee, closing, why string
	}{
		{2, "-25.00", "-395.00", "-370.00 before its own fee"},
		{3, "0.00", "5.00", "+400 clears it, the Day 2 fee cuts it to +5.00"},
		{4, "-25.00", "-205.00", "-180.00 before its own fee"},
		{5, "-25.00", "-230.00", "-205.00 before its own fee"},
	} {
		if got, want := l.Assessed("ACC-001", tc.day, FeeAssessment), Amount(AED, tc.fee); !got.Equal(want) {
			t.Errorf("day %d fee = %s, want %s (%s)", tc.day, got, want, tc.why)
		}
		if got, want := l.Closing("ACC-001", tc.day, 5), Amount(AED, tc.closing); !got.Equal(want) {
			t.Errorf("day %d closing = %s, want %s (%s)", tc.day, got, want, tc.why)
		}
	}

	// Day 2 closed negative once reopened, so the interest it had accrued is
	// adjusted away rather than left standing beside a fee.
	if got := l.Assessed("ACC-001", 2, InterestAssessment); !got.IsZero() {
		t.Errorf("day 2 accrual = %s, want 0.00", got)
	}
}
