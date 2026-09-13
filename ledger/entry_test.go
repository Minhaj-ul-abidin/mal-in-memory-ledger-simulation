package ledger

import "testing"

// An accrual that moved the balance would be counted twice: once as it accrues,
// again when the total capitalizes.
func TestOnlyAccrualsAreMemoEntries(t *testing.T) {
	cases := []struct {
		kind  EntryKind
		moves bool
	}{
		{Credit, true},
		{Debit, true},
		{Reversal, true},
		{Fee, true},
		{FeeReversal, true},
		{Accrual, false},
		{AccrualAdjustment, false},
		{Capitalization, true},
	}

	for _, c := range cases {
		t.Run(c.kind.String(), func(t *testing.T) {
			if got := c.kind.AffectsBalance(); got != c.moves {
				t.Fatalf("%s AffectsBalance() = %v, want %v", c.kind, got, c.moves)
			}
		})
	}
}
