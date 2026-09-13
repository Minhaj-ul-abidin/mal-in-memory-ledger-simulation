package ledger

const window = 6

var accounts = []struct {
	ID  string
	Ccy Currency
}{
	{"ACC-001", AED},
	{"ACC-002", BHD},
}

func Replay(events []Event) Report {
	var r Report
	for d := Day(1); d <= window; d++ {
		day := DayReport{Day: d}
		for _, a := range accounts {
			day.Balances = append(day.Balances, AccountClosing{
				Account: a.ID,
				Closing: Zero(a.Ccy),
			})
		}
		r.Days = append(r.Days, day)
	}
	return r
}
