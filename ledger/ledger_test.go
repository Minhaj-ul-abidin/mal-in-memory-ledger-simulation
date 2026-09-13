package ledger

import "testing"

func testLedger() *Ledger {
	l := New()
	l.Open("ACC-001", AED)
	return l
}

// Criterion 1 in miniature. The same value day answers differently depending on
// when it is asked, and a query taking one date could not tell the two apart.
func TestClosingSeparatesValueDateFromBookingDate(t *testing.T) {
	l := testLedger()
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "1200.00"), BookedOn: 1, ValueDate: 1})
	l.Append(Entry{Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-950.00"), BookedOn: 1, ValueDate: 1})
	l.Append(Entry{Account: "ACC-001", Kind: Debit, Amount: Amount(AED, "-620.00"), BookedOn: 5, ValueDate: 2})

	for _, tc := range []struct {
		valueDay, asOf Day
		want           string
		why            string
	}{
		{2, 4, "250.00", "the backdated debit is not known yet"},
		{2, 5, "-370.00", "criterion 1: known on Day 5, valued Day 2"},
		{1, 5, "250.00", "value date 1 excludes the Day 2 debit"},
		{5, 5, "-370.00", "nothing is dated between Day 3 and Day 5"},
	} {
		got := l.Closing("ACC-001", tc.valueDay, tc.asOf)
		if want := Amount(AED, tc.want); !got.Equal(want) {
			t.Errorf("Closing(vd %d, asOf %d) = %s, want %s (%s)", tc.valueDay, tc.asOf, got, want, tc.why)
		}
	}
}

// An accrual that moved the balance would be counted twice, once as it accrues
// and again when the total capitalizes.
func TestClosingIgnoresAccruals(t *testing.T) {
	l := testLedger()
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "250.00"), BookedOn: 1, ValueDate: 1})
	l.Append(Entry{Account: "ACC-001", Kind: Accrual, Amount: Amount(AED, "0.10"), BookedOn: 1, ValueDate: 1})

	if got, want := l.Closing("ACC-001", 1, 1), Amount(AED, "250.00"); !got.Equal(want) {
		t.Errorf("closing = %s, want %s", got, want)
	}
}

func TestAppendNumbersFromOne(t *testing.T) {
	e := testLedger().Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, "1.00"), BookedOn: 1, ValueDate: 1})
	if e.Seq != 1 {
		t.Fatalf("first Seq = %d, want 1: zero has to mean no reference", e.Seq)
	}
}

// A BHD amount on an AED account is a caller bug, and silently holding it is how
// two currencies end up summed into one meaningless figure.
func TestAppendRejectsForeignCurrency(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("appending BHD to an AED account should panic")
		}
	}()
	testLedger().Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(BHD, "1.000"), BookedOn: 1, ValueDate: 1})
}
