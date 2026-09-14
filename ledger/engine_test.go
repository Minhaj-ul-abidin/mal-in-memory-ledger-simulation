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
		{"5.00", "0.00", "0.00", "0.002 is under a fils, carried to the next day"},
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

// The BHD fee is a chosen constant this stream never reaches (NUMBERS.md). A
// change that starts charging it should fail here rather than pass silently.
func TestBHDFeeIsNeverChargedByThisStream(t *testing.T) {
	l, _, _ := replay(Stream())

	for _, e := range l.Entries() {
		if e.Account == "ACC-002" && (e.Kind == Fee || e.Kind == FeeReversal) {
			t.Errorf("ACC-002 charged %s for value day %d, booked day %d", e.Amount.Display(), e.ValueDate, e.BookedOn)
		}
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

// The original is left exactly as it was. A reversal that edited it would be an
// UPDATE on a booked entry, which is the one thing this design forbids.
func TestReversalPostsContraAtOriginalValueDate(t *testing.T) {
	l := New()
	l.Open("ACC-001", AED)
	original := l.Append(Entry{
		Source: "E7", Account: "ACC-001", Kind: Debit,
		Amount: Amount(AED, "-620.00"), BookedOn: 5, ValueDate: 2,
	})

	day := DayReport{Day: 6}
	reverse(l, Event{ID: "E9", Type: ReversalEvent, BookedOn: 6, ValueDate: 2, Account: "ACC-001", Reverses: "E7"}, &day)

	entries := l.Entries()
	if len(entries) != 2 {
		t.Fatalf("ledger holds %d entries, want 2: the original plus its contra", len(entries))
	}
	if got := entries[0]; !got.Amount.Equal(Amount(AED, "-620.00")) || got.ValueDate != 2 || got.BookedOn != 5 {
		t.Errorf("original was modified: %+v", got)
	}

	contra := entries[1]
	if contra.Kind != Reversal {
		t.Errorf("kind = %s, want %s", contra.Kind, Reversal)
	}
	if want := Amount(AED, "620.00"); !contra.Amount.Equal(want) {
		t.Errorf("amount = %s, want %s", contra.Amount, want)
	}
	if contra.ValueDate != 2 {
		t.Errorf("value date = %d, want 2: a contra carries the original's value date", contra.ValueDate)
	}
	if contra.BookedOn != 6 {
		t.Errorf("booked on = %d, want 6", contra.BookedOn)
	}
	if contra.Ref != original.Seq {
		t.Errorf("ref = %d, want %d", contra.Ref, original.Seq)
	}
}

// Criterion 6. Under back-valuation the fees E7 caused do not merely stop being
// charged: each is reversed, and every day closes where it would have closed had
// E7 never been sent.
func TestReversalUndoesEveryFeeItCaused(t *testing.T) {
	l := New()
	l.Open("ACC-001", AED)
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "1200.00"), BookedOn: 1, ValueDate: 1})
	l.Append(Entry{Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-950.00"), BookedOn: 1, ValueDate: 1})
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "400.00"), BookedOn: 3, ValueDate: 3})
	l.Append(Entry{Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-185.00"), BookedOn: 4, ValueDate: 4})
	for d := Day(1); d <= 4; d++ {
		assess(l, "ACC-001", d, d)
	}

	l.Append(Entry{Source: "E7", Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-620.00"), BookedOn: 5, ValueDate: 2})
	for k := reopenFrom(l, "ACC-001", 5); k <= 5; k++ {
		assess(l, "ACC-001", k, 5)
	}

	day := DayReport{Day: 6}
	reverse(l, Event{ID: "E9", Type: ReversalEvent, BookedOn: 6, ValueDate: 2, Account: "ACC-001", Reverses: "E7"}, &day)
	for k := reopenFrom(l, "ACC-001", 6); k <= 6; k++ {
		assess(l, "ACC-001", k, 6)
	}

	total := Zero(AED)
	for _, tc := range []struct {
		day              Day
		closing, accrual string
	}{
		{1, "250.00", "0.10"},
		{2, "250.00", "0.10"},
		{3, "650.00", "0.26"},
		{4, "465.00", "0.19"},
		{5, "465.00", "0.18"},
		{6, "465.00", "0.19"},
	} {
		if got, want := l.Closing("ACC-001", tc.day, 6), Amount(AED, tc.closing); !got.Equal(want) {
			t.Errorf("day %d closing = %s, want %s", tc.day, got, want)
		}
		accrual := l.Assessed("ACC-001", tc.day, InterestAssessment)
		if want := Amount(AED, tc.accrual); !accrual.Equal(want) {
			t.Errorf("day %d accrual = %s, want %s", tc.day, accrual, want)
		}
		if fee := l.Assessed("ACC-001", tc.day, FeeAssessment); !fee.IsZero() {
			t.Errorf("day %d fee = %s, want 0.00: every fee E7 caused must reverse", tc.day, fee)
		}
		total = total.Add(accrual)
	}

	if want := Amount(AED, "1.02"); !total.Equal(want) {
		t.Errorf("accruals total %s, want %s", total, want)
	}
}

// The capitalized credit is built out of the rounded daily accruals, so they sum
// to it exactly by construction. Deriving it from the balances independently is
// what leaves a remainder no event ever moved.
func TestCapitalizationEqualsTheSumOfAccruals(t *testing.T) {
	for _, tc := range []struct {
		ccy     Currency
		account string
		daily   []string
		want    string
	}{
		{AED, "ACC-001", []string{"0.10", "0.10", "0.26", "0.19", "0.18", "0.19"}, "1.02"},
		{BHD, "ACC-002", []string{"0.000", "0.000", "0.000", "0.000", "0.004", "0.004"}, "0.008"},
	} {
		t.Run(tc.ccy.Code, func(t *testing.T) {
			l := New()
			l.Open(tc.account, tc.ccy)
			for i, amount := range tc.daily {
				day := Day(i + 1)
				l.Append(Entry{
					Account: tc.account, Kind: Accrual, Amount: Amount(tc.ccy, amount),
					BookedOn: day, ValueDate: day,
				})
			}

			capitalize(l, tc.account, 6)

			entries := l.Entries()
			credit := entries[len(entries)-1]
			if credit.Kind != Capitalization {
				t.Fatalf("last entry is %s, want %s", credit.Kind, Capitalization)
			}
			if want := Amount(tc.ccy, tc.want); !credit.Amount.Equal(want) {
				t.Errorf("capitalized %s, want %s", credit.Amount, want)
			}
			// Accruals are memo entries, so the balance is the capitalized credit alone.
			if got, want := l.Closing(tc.account, 6, 6), Amount(tc.ccy, tc.want); !got.Equal(want) {
				t.Errorf("closing = %s, want %s", got, want)
			}
		})
	}
}

// The whole stream, end to end. Both accounts close where they would have closed
// had E7 never been sent, plus the interest they earned along the way.
func TestReplayClosesTheWindowWhereItStarted(t *testing.T) {
	r := Replay(Stream())
	if len(r.Days) != window {
		t.Fatalf("replayed %d days, want %d", len(r.Days), window)
	}

	want := map[string]string{"ACC-001": "AED 466.02", "ACC-002": "BHD 10.008"}
	for _, b := range r.Days[window-1].Balances {
		if got := b.Closing.Display(); got != want[b.Account] {
			t.Errorf("%s closed at %s, want %s", b.Account, got, want[b.Account])
		}
	}
}

// A balance too small to earn a whole minor unit in one day must still earn over
// time. AED 5.00 earns 0.002 a day; rounded on its own that is nothing, and the
// customer would earn nothing for as long as the money sat there.
func TestSmallBalanceStillEarnsInterestOverTime(t *testing.T) {
	const days = 100

	l := New()
	l.Open("ACC-001", AED)
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "5.00"), BookedOn: 1, ValueDate: 1})
	for d := Day(1); d <= days; d++ {
		assess(l, "ACC-001", d, d)
	}

	total := Zero(AED)
	for d := Day(1); d <= days; d++ {
		total = total.Add(l.Assessed("ACC-001", d, InterestAssessment))
	}

	// 5.00 at 0.04% for 100 days is 0.20 exactly, with nothing to round away.
	if want := Amount(AED, "0.20"); !total.Equal(want) {
		t.Errorf("100 days on 5.00 earned %s, want %s", total, want)
	}
}
