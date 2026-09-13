package ledger

import "testing"

// THIS TEST FAILS ON PURPOSE. It is the one failing test the brief asks for, and
// it is aimed at this design rather than at a wrong acceptance criterion.
//
// What it reveals: restatement is value-correct and risk-blind.
//
// Policy C exists so that final values do not depend on the order the news
// arrived in. It delivers that. Every day closes where it would have closed had
// E7 never been sent, and the three fees it caused are reversed. Criterion 6
// holds, and the ledger is honest about money.
//
// But the bank's exposure was real. From the end of Day 5 the account stood at
// -230.00, and the ledger acted on that: Auth-B was declined because available
// balance was -155.00 and a 90.00 hold would have taken it to -245.00. A customer
// was refused. That refusal is preserved, because authorization decisions are
// never restated (AMBIGUITIES.md entry 6).
//
// What is not preserved is the reason for it. The restated ledger shows Day 5 at
// 465.00, which covers a 90.00 hold with 375.00 to spare. So the ledger now holds
// a decline it cannot justify from its own authoritative view, and answers "no"
// to "was this account ever overdrawn" while carrying the evidence that it was.
//
// The information is not destroyed: Closing(account, 5, 5) still returns -230.00,
// because entries keep both dates. The problem is that the restated view is the
// one the design treats as the answer, and credit risk has to read a different
// view to get a different answer. Nothing in the design says which view a given
// question belongs to.
//
// Fixing it is out of scope here and would not be a patch. Value and risk are two
// different books over the same entries, and this design only ever built one.
func TestEveryDeclineIsJustifiableFromTheRestatedLedger(t *testing.T) {
	l, holds, _ := replay(Stream())

	var declined int
	for _, h := range holds {
		if h.State != HoldDeclined {
			continue
		}
		declined++

		restated := l.Closing(h.Account, h.OpenedOn, window)
		if headroom := restated.Sub(h.Amount); !headroom.IsNegative() {
			t.Errorf("%s was declined on Day %d, but the restated ledger shows %s on that day, "+
				"which covers the %s hold with %s to spare. The decline stands in the record "+
				"with nothing in the authoritative view to justify it.",
				h.ID, h.OpenedOn, restated, h.Amount, headroom)
		}
	}

	if declined == 0 {
		t.Fatal("no declined authorization in the window; this test has nothing to say")
	}
}

// The evidence the failing test is complaining about, kept here so the two can be
// read together: the same day, asked from two vantages, gives two answers.
func TestTheOverdrawnDayIsStillVisibleFromItsOwnVantage(t *testing.T) {
	l, _, _ := replay(Stream())

	if got, want := l.Closing("ACC-001", 5, 5), Amount(AED, "-230.00"); !got.Equal(want) {
		t.Errorf("day 5 seen from day 5 = %s, want %s", got, want)
	}
	if got, want := l.Closing("ACC-001", 5, window), Amount(AED, "465.00"); !got.Equal(want) {
		t.Errorf("day 5 seen from the close = %s, want %s", got, want)
	}
}
