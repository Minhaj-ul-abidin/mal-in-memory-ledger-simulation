package ledger

import "testing"

func fundedLedger(t *testing.T, amount string) *Ledger {
	t.Helper()
	l := New()
	l.Open("ACC-001", AED)
	l.Append(Entry{Account: "ACC-001", Kind: Credit, Amount: Amount(AED, amount), BookedOn: 1, ValueDate: 1})
	return l
}

// Sizing an authorization on the ledger balance alone lets two of them pass the
// same check, and the second one only fails after the money has already moved.
func TestAvailableSubtractsActiveHolds(t *testing.T) {
	l := fundedLedger(t, "250.00")
	holds := Holds{{ID: "Auth-A", Account: "ACC-001", Amount: Amount(AED, "200.00"), State: HoldActive}}

	if got, want := l.Closing("ACC-001", 1, 1), Amount(AED, "250.00"); !got.Equal(want) {
		t.Errorf("a hold must not move the ledger balance: got %s, want %s", got, want)
	}
	if got, want := l.Available(holds, "ACC-001", 1), Amount(AED, "50.00"); !got.Equal(want) {
		t.Errorf("available = %s, want %s", got, want)
	}
}

// A declined authorization never opened, so it cannot be reducing anything.
func TestDeclinedHoldDoesNotReduceAvailable(t *testing.T) {
	l := fundedLedger(t, "250.00")
	holds := Holds{{ID: "Auth-B", Account: "ACC-001", Amount: Amount(AED, "90.00"), State: HoldDeclined}}

	if got, want := l.Available(holds, "ACC-001", 1), Amount(AED, "250.00"); !got.Equal(want) {
		t.Errorf("available = %s, want %s", got, want)
	}
}

func TestAuthorizeDecidesOnAvailableBalance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		balance string
		hold    string
		want    HoldState
	}{
		{"Auth-A: 250 covers a 200 hold", "250.00", "200.00", HoldActive},
		{"exactly zero after the hold is still approved", "200.00", "200.00", HoldActive},
		{"one fils short is declined", "199.99", "200.00", HoldDeclined},
		{"a negative balance declines everything", "-155.00", "90.00", HoldDeclined},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := fundedLedger(t, tc.balance)
			var holds Holds
			day := DayReport{Day: 1}
			authorize(l, &holds, Event{
				ID: "E", Type: AuthorizationEvent, BookedOn: 1, ValueDate: 1,
				Account: "ACC-001", Amount: Amount(AED, tc.hold), Auth: "Auth-X",
			}, &day)

			if got := holds.Find("Auth-X").State; got != tc.want {
				t.Fatalf("state = %s, want %s", got, tc.want)
			}
			if tc.want == HoldDeclined && len(day.Errors) != 1 {
				t.Errorf("a decline must be recorded as an error, got %d", len(day.Errors))
			}
		})
	}
}
