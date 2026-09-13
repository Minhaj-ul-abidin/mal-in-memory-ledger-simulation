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

type Report struct {
	Days []DayReport
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
