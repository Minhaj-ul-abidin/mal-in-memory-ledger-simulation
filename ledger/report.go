package ledger

import (
	"fmt"
	"io"
)

type AccountClosing struct {
	Account string
	Closing Money
}

type DayReport struct {
	Day      Day
	Balances []AccountClosing
	Fees     []string
	Interest []string
	Auths    []string
	Errors   []string
}

// RestatedDay is a day's closing as known at the end of the window, which is a
// different figure from what that day closed at when it closed. Days reopened by
// a backdated entry only show their final value here.
type RestatedDay struct {
	Day      Day
	Balances []AccountClosing
}

type Report struct {
	Days     []DayReport
	Restated []RestatedDay
}

func (r Report) Print(w io.Writer) {
	for _, d := range r.Days {
		fmt.Fprintf(w, "Day %d\n", d.Day)
		for _, b := range d.Balances {
			fmt.Fprintf(w, "  %-9s closing  %s\n", b.Account, b.Closing.Display())
		}
		lines(w, "fees", d.Fees)
		lines(w, "interest", d.Interest)
		lines(w, "auths", d.Auths)
		lines(w, "errors", d.Errors)
		fmt.Fprintln(w)
	}
	r.printRestated(w)
}

func (r Report) printRestated(w io.Writer) {
	if len(r.Restated) == 0 {
		return
	}
	fmt.Fprintf(w, "Restated at close of Day %d\n", r.Restated[len(r.Restated)-1].Day)
	for _, d := range r.Restated {
		fmt.Fprintf(w, "  Day %d  ", d.Day)
		for _, b := range d.Balances {
			fmt.Fprintf(w, "  %-9s %-14s", b.Account, b.Closing.Display())
		}
		fmt.Fprintln(w)
	}
}

func lines(w io.Writer, label string, ls []string) {
	if len(ls) == 0 {
		fmt.Fprintf(w, "  %-9s none\n", label)
		return
	}
	for _, l := range ls {
		fmt.Fprintf(w, "  %-9s %s\n", label, l)
		label = ""
	}
}
