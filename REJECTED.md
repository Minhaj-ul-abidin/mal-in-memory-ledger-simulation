### The Upfront Criteria rejection evident from the questions example ledger. 
1. Criterian 8 "If the rounded daily interest accruals do not sum to the capitalized total, the remainder is discarded" 
Reason: This is a refusal to the rule 
"The rounded daily accruals must sum exactly to the capitalized total." Which can't be accepted as a secondary policy otherwise the code contradicts the rule it is implemented to must follow. 
2. "The three BHD instalments in E10 must each be BHD 3.334"
Reason: This is outrightly fabricating .002 BHD when it is not in ledger entries . The ledger mustn't report money under or more than that an event has moved.
3. "E7 causes exactly one overdraft fee to be assessed, on Day 2" 
Reason: This is not viable in any read of the leadger. 
- Under back valuation calculated after the funds are resetted on day 2 which comes as three fees Days 2, 4, 5 (before E9). (day 3 bails out with positive +5 and not +30 because of the fee incurred on negative day 2 which is considered for day 3 closing and pulls it down to +5)
- Under As-of day fee is day 5 fee not day 2.
4. "If Auth-B is approved, its hold reduces available balance but not ledger balance"
Reason: it is true but it is only restating the rule for available balance. This premise never happens. At E8 the ledger is negative at -155.00 and Auth-A is already released so there are no active holds, -155.00 - 90.00 = -245.00 which is below zero, so Auth-B is declined and never approved.
So this "Auth-B is never settled inside the window" reads like it was approved and just not presented however it was refused at the authorization itself.

### Approaches worked through and dropped
These were reasoned out from the rules and walked through against the event stream, then dropped before any code was written. None of them were abandoned by running them.
1. As-of day end. A closed day is final and never reopens. Dropped because the fee E7 triggers still stands after E9 takes E7 back, the customer keeps paying for an entry the bank itself reversed. See AMBIGUITIES 4.
2. Recalculate and rewrite. Reopen the day and correct the entries that are already sitting there. Dropped because it edits entries inside closed days, which append only does not allow. See AMBIGUITIES 4.
3. float64 for amounts. Dropped for int64 minor units. A float cannot hold 0.1 exactly so a float ledger cannot represent its own smallest unit, and the error turns up later as dust that no event ever moved.

### Approaches abandoned mid-build
These were built and committed, then replaced once a test showed them wrong.
1. Rounding each day's interest accrual on its own. Dropped because anything under half a minor unit is rounded away every day and never comes back, 5.00 earns 0.002 a day which rounds to 0.00, so a hundred days of it pays nothing instead of 0.20. The instalment split already refused to discard a residual and the interest was discarding one every day. Replaced by rounding the running total and booking each day as the increment of it, so the daily accruals still sum exactly to the capitalized credit. A literal read of "rounded daily accruals" rounds each day and gets 1.03, this gets 1.02 and closes at 466.02, exact interest is 1.018. See NUMBERS.md and WORKLOG 11:45 PM.
