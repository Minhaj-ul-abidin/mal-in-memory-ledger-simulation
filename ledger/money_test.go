package ledger

import "testing"

func TestAmountHoldsCurrencyPrecision(t *testing.T) {
	for _, tc := range []struct {
		ccy   Currency
		in    string
		units int64
		out   string
	}{
		{AED, "1200.00", 120000, "1200.00"},
		{AED, "1,200.00", 120000, "1200.00"},
		{AED, "-370.00", -37000, "-370.00"},
		{AED, "0.04", 4, "0.04"},
		{BHD, "10.000", 10000, "10.000"},
		{BHD, "3.334", 3334, "3.334"},
		{BHD, "0.004", 4, "0.004"},
	} {
		got := Amount(tc.ccy, tc.in)
		if got.Units() != tc.units {
			t.Errorf("Amount(%s, %q).Units() = %d, want %d", tc.ccy.Code, tc.in, got.Units(), tc.units)
		}
		if got.String() != tc.out {
			t.Errorf("Amount(%s, %q).String() = %q, want %q", tc.ccy.Code, tc.in, got.String(), tc.out)
		}
	}
}

// An amount finer than the currency is a caller bug. Rounding it away silently
// is how a ledger starts holding money it cannot name.
func TestAmountRejectsExcessPrecision(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Amount(AED, \"1.005\") should panic: AED holds 2 decimals")
		}
	}()
	Amount(AED, "1.005")
}

func TestRateRoundsHalfUp(t *testing.T) {
	// 0.04% per day = 4/10000.
	for _, tc := range []struct {
		ccy      Currency
		balance  string
		expected string
		why      string
	}{
		{AED, "250.00", "0.10", "exact, no rounding"},
		{AED, "650.00", "0.26", "exact"},
		{AED, "465.00", "0.19", "0.186 rounds up"},
		{AED, "440.00", "0.18", "0.176 rounds up"},
		{AED, "390.00", "0.16", "0.156 rounds up"},
		{AED, "5.00", "0.00", "0.002 rounds down to nothing"},
		{BHD, "10.000", "0.004", "exact at 3dp"},
	} {
		got := Amount(tc.ccy, tc.balance).Rate(4, 10000)
		if want := Amount(tc.ccy, tc.expected); !got.Equal(want) {
			t.Errorf("%s %s @ 0.04%% = %s, want %s (%s)", tc.ccy.Code, tc.balance, got, want, tc.why)
		}
	}
}

// The invariant behind acceptance criterion 7.
func TestSplitPartsSumToWhole(t *testing.T) {
	whole := Amount(BHD, "10.000")
	parts := Split(whole, 3)

	want := []string{"3.333", "3.333", "3.334"}
	for i, p := range parts {
		if p.String() != want[i] {
			t.Errorf("instalment %d = %s, want %s", i+1, p, want[i])
		}
	}
	if total := Sum(BHD, parts...); !total.Equal(whole) {
		t.Errorf("instalments sum to %s, want exactly %s", total, whole)
	}
}

func TestSplitSumsForAwkwardCases(t *testing.T) {
	for _, tc := range []struct {
		ccy Currency
		amt string
		n   int
	}{
		{BHD, "10.000", 3}, {AED, "0.01", 3}, {AED, "100.00", 7},
		{AED, "-370.00", 3}, {BHD, "0.001", 2}, {AED, "1200.00", 11},
	} {
		whole := Amount(tc.ccy, tc.amt)
		if total := Sum(tc.ccy, Split(whole, tc.n)...); !total.Equal(whole) {
			t.Errorf("Split(%s, %d) sums to %s, want %s", whole, tc.n, total, whole)
		}
	}
}

// No tie occurs anywhere in the replay, so nothing in the output shows which way
// one resolves. AED 12.50 at 0.04% is exactly half a fils, which is the case the
// rounding mode has to be pinned against.
func TestRateResolvesTiesAwayFromZero(t *testing.T) {
	for _, tc := range []struct{ in, want, why string }{
		{"12.50", "0.01", "0.005 exactly"},
		{"-12.50", "-0.01", "0.005 exactly, away from zero rather than toward it"},
		{"37.50", "0.02", "0.015 exactly, and not 0.01 as half-even would give"},
	} {
		got := Amount(AED, tc.in).Rate(4, 10000)
		if want := Amount(AED, tc.want); !got.Equal(want) {
			t.Errorf("%s at 0.04%% = %s, want %s (%s)", tc.in, got, want, tc.why)
		}
	}
}
