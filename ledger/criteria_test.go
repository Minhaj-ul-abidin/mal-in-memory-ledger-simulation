package ledger

import (
	"strings"
	"testing"
)

// The eight acceptance criteria, each asserted against the real replay. Four of
// them are wrong; those tests assert what the ledger actually does and name the
// claim it contradicts, so the refusal is checked rather than only argued in
// REJECTED.md.

// feeStanding is the net fee against a value day as known at asOf. Ledger.Assessed
// takes no as-of day, which is enough for a forward replay where nothing is booked
// in the future, but not for asking what Day 2 looked like from Day 5.
func feeStanding(l *Ledger, account string, day, asOf Day) Money {
	total := Zero(l.Currency(account))
	for _, e := range l.Entries() {
		if e.Account != account || e.ValueDate != day || e.BookedOn > asOf {
			continue
		}
		if a, ok := e.Kind.assessment(); ok && a == FeeAssessment {
			total = total.Add(e.Amount)
		}
	}
	return total
}

func entriesFrom(l *Ledger, source string) []Entry {
	var out []Entry
	for _, e := range l.Entries() {
		if e.Source == source {
			out = append(out, e)
		}
	}
	return out
}

// ACCEPT. Needs both dates: value day 2, as of day 5. A balance query taking one
// date could not express the question, let alone answer it.
func TestCriterion1_Day2IsMinus370SeenFromDay5(t *testing.T) {
	l, _, _ := replay(Stream())

	withFee := l.Closing("ACC-001", 2, 5)
	beforeFee := withFee.Sub(feeStanding(l, "ACC-001", 2, 5))

	if want := Amount(AED, "-370.00"); !beforeFee.Equal(want) {
		t.Errorf("day 2 before fees, seen from day 5 = %s, want %s", beforeFee, want)
	}
	if want := Amount(AED, "-395.00"); !withFee.Equal(want) {
		t.Errorf("day 2 after its own fee = %s, want %s", withFee, want)
	}
}

// REFUSE. The claim is one fee, on Day 2. The fee rule is defined over day
// closings, not over entries, so one backdated debit pushes three days negative.
func TestCriterion2_E7CausesThreeFeesNotOne(t *testing.T) {
	l, _, _ := replay(Stream())

	for _, tc := range []struct {
		day      Day
		fee, why string
	}{
		{2, "-25.00", "-370.00 before its own fee"},
		{3, "0.00", "the Day 3 credit clears it to +5.00"},
		{4, "-25.00", "-180.00, carrying the Day 2 fee"},
		{5, "-25.00", "-205.00, carrying the Day 2 and Day 4 fees"},
	} {
		got := feeStanding(l, "ACC-001", tc.day, 5)
		if want := Amount(AED, tc.fee); !got.Equal(want) {
			t.Errorf("day %d fee at end of day 5 = %s, want %s (%s)", tc.day, got, want, tc.why)
		}
	}

	var n int
	for _, e := range l.Entries() {
		if e.Kind == Fee && e.BookedOn <= 5 {
			n++
		}
	}
	if n != 3 {
		t.Errorf("E7 caused %d fees, want 3: the criterion claims exactly one, on Day 2", n)
	}
}

// ACCEPT. Auth-A was approvable when it arrived, and 185.00 is within the hold.
func TestCriterion3_AuthASettlesOnDay4(t *testing.T) {
	l, holds, _ := replay(Stream())

	h := holds.Find("Auth-A")
	if h == nil {
		t.Fatal("Auth-A was never recorded")
	}
	if h.State != HoldPartiallySettled {
		t.Errorf("Auth-A is %s, want %s: 185.00 settles below the 200.00 hold", h.State, HoldPartiallySettled)
	}

	posted := entriesFrom(l, "E5")
	if len(posted) != 1 {
		t.Fatalf("E5 posted %d entries, want 1", len(posted))
	}
	if want := Amount(AED, "-185.00"); !posted[0].Amount.Equal(want) {
		t.Errorf("E5 posted %s, want %s", posted[0].Amount, want)
	}
	if posted[0].BookedOn != 4 || posted[0].ValueDate != 4 {
		t.Errorf("E5 dates = booked %d valued %d, want 4 and 4", posted[0].BookedOn, posted[0].ValueDate)
	}
}

// ACCEPT as specified. Real card networks force-post a late presentment because
// the money has already moved; that divergence is recorded in AMBIGUITIES.md.
func TestCriterion4_UnknownAuthSettlementMovesNoMoney(t *testing.T) {
	l, _, r := replay(Stream())

	if posted := entriesFrom(l, "E6"); len(posted) != 0 {
		t.Errorf("E6 posted %d entries, want none: the funds must not leave", len(posted))
	}

	var recorded bool
	for _, msg := range r.Days[3].Errors {
		if strings.Contains(msg, "E6") && strings.Contains(msg, "Auth-Z") {
			recorded = true
		}
	}
	if !recorded {
		t.Errorf("day 4 errors do not record the rejection: %v", r.Days[3].Errors)
	}
}

// VACUOUS. The statement is true by definition, but the premise never fires:
// -155.00 - 90.00 = -245.00, so Auth-B is declined and no hold ever opens.
func TestCriterion5_AuthBIsNeverApproved(t *testing.T) {
	l, holds, _ := replay(Stream())

	b := holds.Find("Auth-B")
	if b == nil {
		t.Fatal("Auth-B was never recorded; a declined authorization is still a decision")
	}
	if b.State != HoldDeclined {
		t.Errorf("Auth-B is %s, want %s", b.State, HoldDeclined)
	}
	if posted := entriesFrom(l, "E8"); len(posted) != 0 {
		t.Errorf("E8 posted %d entries, want none", len(posted))
	}
	if got := (Holds{b}).ActiveTotal("ACC-001", AED); !got.IsZero() {
		t.Errorf("a declined hold reduces available by %s, want nothing", got)
	}
}

// ACCEPT, and conditional. This holds under back-valuation with restatement. Under
// as-of day end it is false: the fee E7 triggered would still stand after E9.
func TestCriterion6_E9ReturnsBalancesAndFees(t *testing.T) {
	l, _, _ := replay(Stream())

	for _, tc := range []struct {
		day     Day
		closing string
	}{
		{1, "250.00"},
		{2, "250.00"},
		{3, "650.00"},
		{4, "465.00"},
		{5, "465.00"},
		{6, "466.02"},
	} {
		if got, want := l.Closing("ACC-001", tc.day, window), Amount(AED, tc.closing); !got.Equal(want) {
			t.Errorf("day %d restated = %s, want %s", tc.day, got, want)
		}
		if fee := feeStanding(l, "ACC-001", tc.day, window); !fee.IsZero() {
			t.Errorf("day %d still carries %s in fees, want none", tc.day, fee)
		}
	}

	// Nothing was deleted to achieve that. Three fees and three reversals stand.
	var fees, reversals int
	for _, e := range l.Entries() {
		switch e.Kind {
		case Fee:
			fees++
		case FeeReversal:
			reversals++
		}
	}
	if fees != 3 || reversals != 3 {
		t.Errorf("ledger holds %d fees and %d reversals, want 3 and 3", fees, reversals)
	}
}

// REFUSE. 3.334 x 3 = 10.002, which puts 0.002 BHD in the ledger that no event
// ever moved.
func TestCriterion7_InstalmentsAreNotAll3334(t *testing.T) {
	l, _, _ := replay(Stream())

	posted := entriesFrom(l, "E10")
	if len(posted) != 3 {
		t.Fatalf("E10 posted %d entries, want 3", len(posted))
	}

	want := []string{"3.333", "3.333", "3.334"}
	parts := make([]Money, len(posted))
	for i, e := range posted {
		parts[i] = e.Amount
		if e.Amount.String() != want[i] {
			t.Errorf("instalment %d = %s, want %s", i+1, e.Amount, want[i])
		}
	}
	if total := Sum(BHD, parts...); !total.Equal(Amount(BHD, "10.000")) {
		t.Errorf("instalments sum to %s, want exactly 10.000", total)
	}
}

// REFUSE. There is never a remainder to discard: the capitalized credit is built
// out of the rounded daily accruals, so it equals their sum by construction.
func TestCriterion8_NoRemainderExistsToDiscard(t *testing.T) {
	l, _, _ := replay(Stream())

	for _, a := range accounts {
		accrued, capitalized := Zero(a.Ccy), Zero(a.Ccy)
		for _, e := range l.Entries() {
			if e.Account != a.ID {
				continue
			}
			switch e.Kind {
			case Accrual, AccrualAdjustment:
				accrued = accrued.Add(e.Amount)
			case Capitalization:
				capitalized = capitalized.Add(e.Amount)
			}
		}
		if !accrued.Equal(capitalized) {
			t.Errorf("%s accrued %s but capitalized %s", a.ID, accrued, capitalized)
		}
	}
}
