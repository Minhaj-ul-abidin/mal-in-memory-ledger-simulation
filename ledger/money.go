package ledger

import (
	"fmt"
	"strconv"
	"strings"
)

// Currency fixes how many decimal places an amount is stored and rounded to.
// Precision belongs to the currency, never to the arithmetic that touches it.
type Currency struct {
	Code     string
	Decimals int
}

var (
	AED = Currency{Code: "AED", Decimals: 2}
	BHD = Currency{Code: "BHD", Decimals: 3}
)

func (c Currency) scale() int64 {
	s := int64(1)
	for i := 0; i < c.Decimals; i++ {
		s *= 10
	}
	return s
}

// Money is a signed amount in the currency's minor units: 1/100 AED, 1/1000 BHD.
//
// Integers rather than float64. A float64 cannot hold 0.1 exactly, so a float
// ledger cannot represent its own smallest unit and will eventually disagree
// with itself by amounts no one can see but every reconciliation can.
type Money struct {
	units int64
	ccy   Currency
}

func Minor(ccy Currency, units int64) Money { return Money{units: units, ccy: ccy} }

func Zero(ccy Currency) Money { return Money{ccy: ccy} }

// Amount builds Money from a decimal string ("1200.00"). It rejects any input
// carrying more precision than the currency has, rather than silently rounding
// it away — an amount the ledger cannot store is a caller bug, not an input.
func Amount(ccy Currency, s string) Money {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(strings.ReplaceAll(s, ",", ""), ".")

	if len(frac) > ccy.Decimals {
		panic(fmt.Sprintf("ledger: %s.%s has more precision than %s holds (%d dp)", whole, frac, ccy.Code, ccy.Decimals))
	}
	frac += strings.Repeat("0", ccy.Decimals-len(frac))

	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("ledger: bad amount %q: %v", s, err))
	}
	var f int64
	if frac != "" {
		if f, err = strconv.ParseInt(frac, 10, 64); err != nil {
			panic(fmt.Sprintf("ledger: bad amount %q: %v", s, err))
		}
	}

	units := w*ccy.scale() + f
	if neg {
		units = -units
	}
	return Money{units: units, ccy: ccy}
}

func (m Money) Currency() Currency { return m.ccy }
func (m Money) Units() int64       { return m.units }
func (m Money) IsZero() bool       { return m.units == 0 }
func (m Money) IsNegative() bool   { return m.units < 0 }
func (m Money) IsPositive() bool   { return m.units > 0 }

func (m Money) assertSame(o Money) {
	if m.ccy.Code != o.ccy.Code {
		panic(fmt.Sprintf("ledger: currency mismatch %s vs %s", m.ccy.Code, o.ccy.Code))
	}
}

func (m Money) Add(o Money) Money { m.assertSame(o); return Money{m.units + o.units, m.ccy} }
func (m Money) Sub(o Money) Money { m.assertSame(o); return Money{m.units - o.units, m.ccy} }
func (m Money) Neg() Money        { return Money{-m.units, m.ccy} }

func (m Money) Equal(o Money) bool { m.assertSame(o); return m.units == o.units }

// String renders the amount at the currency's own precision, always.
// "10.000" and "10.00" are different claims about what is known.
func (m Money) String() string {
	sign, u := "", m.units
	if u < 0 {
		sign, u = "-", -u
	}
	s := m.ccy.scale()
	if m.ccy.Decimals == 0 {
		return fmt.Sprintf("%s%d", sign, u)
	}
	return fmt.Sprintf("%s%d.%0*d", sign, u/s, m.ccy.Decimals, u%s)
}

func (m Money) Display() string { return m.ccy.Code + " " + m.String() }

// Rate applies num/den to an amount, rounding half-up away from zero.
//
// Half-up, not half-even: this is a customer-facing interest figure, and the
// tie always resolving toward the customer is easier to defend to a regulator
// than one that alternates by parity. See NUMBERS.md.
func (m Money) Rate(num, den int64) Money {
	n := m.units * num
	if n < 0 {
		return Money{-((-n + den/2) / den), m.ccy}
	}
	return Money{(n + den/2) / den, m.ccy}
}

// Split divides an amount into n parts that sum to exactly the original.
//
// The indivisible remainder lands on the LAST part. Rounding each part
// independently is how 10.000 BHD becomes three instalments of 3.334 and the
// ledger reports 0.002 BHD that no event ever moved.
func Split(m Money, n int) []Money {
	if n <= 0 {
		panic("ledger: split into non-positive parts")
	}
	base, rem := m.units/int64(n), m.units%int64(n)
	parts := make([]Money, n)
	for i := range parts {
		parts[i] = Money{base, m.ccy}
	}
	parts[n-1] = Money{base + rem, m.ccy}
	return parts
}

// Sum totals amounts of one currency. An empty sum needs the currency stated,
// since there is no zero that belongs to every currency at once.
func Sum(ccy Currency, ms ...Money) Money {
	total := Zero(ccy)
	for _, m := range ms {
		total = total.Add(m)
	}
	return total
}
