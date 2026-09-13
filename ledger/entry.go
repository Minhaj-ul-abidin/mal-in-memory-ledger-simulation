package ledger

type Day int

type EntryKind int

const (
	Credit EntryKind = iota
	Debit
	Reversal
	Fee
	FeeReversal
	Accrual
	AccrualAdjustment
	Capitalization
)

var kindNames = [...]string{
	Credit:            "CREDIT",
	Debit:             "DEBIT",
	Reversal:          "REVERSAL",
	Fee:               "FEE",
	FeeReversal:       "FEE_REVERSAL",
	Accrual:           "ACCRUAL",
	AccrualAdjustment: "ACCRUAL_ADJ",
	Capitalization:    "CAPITALIZATION",
}

func (k EntryKind) String() string { return kindNames[k] }

// Accruals are memo entries: the balance only moves when their summed total
// capitalizes on the last day.
func (k EntryKind) AffectsBalance() bool {
	return k != Accrual && k != AccrualAdjustment
}

// An Entry is never mutated or deleted. A correction is a new entry.
type Entry struct {
	Seq     int
	Source  string
	Account string
	Kind    EntryKind
	Amount  Money // signed, so a balance is a plain sum

	BookedOn  Day // the day the ledger learned about it
	ValueDate Day // the day it belongs to

	Ref int // Seq of the entry this one reverses, 0 if none
}

// An Assessment is what a day close produces. Fee and interest corrections are
// keyed apart so a fee delta can never net against an interest delta.
type Assessment int

const (
	FeeAssessment Assessment = iota
	InterestAssessment
)

func (k EntryKind) assessment() (Assessment, bool) {
	switch k {
	case Fee, FeeReversal:
		return FeeAssessment, true
	case Accrual, AccrualAdjustment:
		return InterestAssessment, true
	}
	return 0, false
}
