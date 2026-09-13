# Constants

Every constant in the ledger, where it came from, and why that value.

## Given by the spec

These are not choices. They are stated in the brief and are reproduced here only
so the chosen values below can be read against them.

| Constant | Value | Used for |
|---|---|---|
| Overdraft fee, AED | `25.00` | assessed once per day per account on a negative closing |
| Daily interest rate | `0.04%` = `4/10000` | positive closing balances only |
| AED precision | 2 decimal places | |
| BHD precision | 3 decimal places | |
| Replay window | 6 days | |

The rate is held as the integer pair `4 / 10000` rather than a decimal fraction,
so the interest calculation never leaves integer arithmetic.

## Chosen

### Overdraft fee, BHD — `2.560`

The spec denominates the fee in AED. ACC-002 is a BHD account and the brief never
says what it is charged. See `AMBIGUITIES.md` entry 2.

Both currencies are pegged to the US dollar at administratively fixed rates:
AED 3.6725/USD and BHD 0.376/USD.

```
25.00 AED ÷ 3.6725 = 6.80736 USD
6.80736 USD × 0.376 = 2.55957 BHD
                    → 2.560 at BHD's three places
```

**Why not half it.** The fee is a penalty for the same behaviour on both
accounts. Halving it means a BHD customer pays half of what an AED customer pays
for an identical overdraft, which is a difference you would have to justify to a
regulator and could not.

**Why not `25.000` BHD.** One BHD is roughly 9.77 AED, so keeping the numeral and
changing the currency makes the penalty about ten times larger in real terms.
The numeral is not the constant; the economic value is.

**Why a fixed constant rather than a rate feed.** Both pegs are set
administratively, not by a market. A constant here records a fact that does not
move, rather than caching one that does. A floating pair would need a feed, a
staleness policy, and an answer to which rate date applies to a back-valued fee.

**Never exercised.** ACC-002 never closes negative in this stream, so this
constant is never reached. A test asserts that, so it cannot drift into silent
use without the suite noticing.

### Rounding mode — half-up, away from zero

`0.005` rounds to `0.01`, and `-0.005` rounds to `-0.01`.

The spec says amounts are stored and rounded to their own precision. It does not
say how a tie resolves. See `AMBIGUITIES.md` entry 7.


### Interest rounding — the running total, not each day

A day's accrual is the increment of the rounded running total, not the rounded
figure for that day taken on its own.

**Why not round each day.** Anything smaller than half a minor unit rounds to
nothing and is gone for good. AED 5.00 earns 0.002 a day, which rounds to 0.00,
so the account earns nothing for as long as the money sits there — a hundred days
of it still pays zero.

**It is the rule the split already follows.** Residuals are allocated, never
discarded, and that has to hold in both directions. Rounding per day enforced it
on the instalments, where the risk is inventing 0.002 nobody moved, and broke it
on the interest, where the risk is confiscating the same amount every day.

**How.** Interest accumulates as integer numerators, `balance × 4`, and is
divided and rounded once over the whole run. A day posts the difference between
the total through today and the total through yesterday, so the daily figures
still sum to the capitalized credit exactly.

**What it costs.** The window total is `1.02` rather than `1.03`, and ACC-001
closes at `466.02`. The exact interest over the six days is `1.018`; rounding
each day separately reached `1.03` by rounding `0.186` up on three days.

### Money representation — `int64` minor units

`AED 1200.00` is stored as `120000`. `BHD 10.000` is stored as `10000`.

**Why not `float64`.** A float cannot hold `0.1` exactly, so a float ledger
cannot represent its own smallest unit. The error is invisible per entry and
shows up at reconciliation as amounts no event moved.

**Why not a decimal library.** Standard library only, and `int64` is sufficient
here by a wide margin: the maximum is roughly `9.2 × 10^18` minor units, which at
BHD's three places is about `9.2 × 10^15` BHD.

**Why minor units rather than a major unit plus a fraction.** One integer, one
comparison, and no normalisation step that can be forgotten.

Precision is carried by `Currency`, never by the arithmetic. No function needs to
know how many places it is working to; it asks the amount.

### Assessment key — `(account, day, assessment type)`

The fee is assessed once per day per account, so `(account, day)` identifies it.
The third segment is what restatement diffs against when it works out the delta
to post.

**Why the type segment is load-bearing.** Without it, a fee correction of
`-25.00` and an interest correction of `+0.10` for the same account and day
collapse into a single `-24.90`, and both are lost. They are different statement
lines with different regulatory treatment and must never net into one figure.
