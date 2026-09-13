package ledger

import (
	"strings"
	"testing"
)

// The restated grid is every day as it stands once E9 has landed. Day 5 is the
// one to read: it closed at -230.00 on the day, and stands at 465.00 at the end.
// Both are true, and a report showing only one of them hides the design.
func TestRestatedViewDiffersFromWhatEachDayClosedAt(t *testing.T) {
	r := Replay(Stream())

	want := []struct{ acc1, acc2 string }{
		{"AED 250.00", "BHD 0.000"},
		{"AED 250.00", "BHD 0.000"},
		{"AED 650.00", "BHD 0.000"},
		{"AED 465.00", "BHD 0.000"},
		{"AED 465.00", "BHD 10.000"},
		{"AED 466.02", "BHD 10.008"},
	}
	if len(r.Restated) != len(want) {
		t.Fatalf("restated %d days, want %d", len(r.Restated), len(want))
	}
	for i, w := range want {
		got := r.Restated[i]
		if got.Day != Day(i+1) {
			t.Errorf("row %d is Day %d", i, got.Day)
		}
		if s := got.Balances[0].Closing.Display(); s != w.acc1 {
			t.Errorf("Day %d ACC-001 restated %s, want %s", got.Day, s, w.acc1)
		}
		if s := got.Balances[1].Closing.Display(); s != w.acc2 {
			t.Errorf("Day %d ACC-002 restated %s, want %s", got.Day, s, w.acc2)
		}
	}

	// The same day, asked on the day rather than at the end.
	if got, want := r.Days[4].Balances[0].Closing.Display(), "AED -230.00"; got != want {
		t.Errorf("Day 5 closed at %s on the day, want %s", got, want)
	}
}

func TestPrintRendersBothViews(t *testing.T) {
	var b strings.Builder
	Replay(Stream()).Print(&b)
	out := b.String()

	for _, want := range []string{
		"Day 5",
		"AED -230.00",
		"E8: authorization Auth-B declined",
		"fee AED -25.00 (value date Day 2)",
		"Restated at close of Day 6",
		"AED 466.02",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q", want)
		}
	}
}
