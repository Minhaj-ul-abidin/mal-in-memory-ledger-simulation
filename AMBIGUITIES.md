1. Fee trigger rule : According to the rules specified fees is trigerred for negative and interest is accrues on positve zero balance is neither. 
  *Assumption*:  all accounts are zero balance allowed accounts and fees is only trigerred for the day with the negative closing balance. 
2. Overdraft Fees is specified as AED 25.00, this opens the question for fx rate fo BHD is not specified. 
  *Assumption*: We will consider a BHD fee conversion with constant fx rate for now but in PROD fee should be describes for all allowed account currencies. See Constants for the mathc.
3. Split ambiguity: the extra value from the splits doesn't have a clear policy of when it is charged. 
  *Assumption*: Taking it the standard way of charging the last installement with any remaining diff added. 
  As a rule Residuals are allocated, never discarded.
4. Backdating policy : every entry carries a value date but the spec never says what happens when an entry lands on a day that is already closed. E7 is booked Day 5 with value date Day 2, so Day 2 either stays closed or it opens again and there is no rule given for which.
  *Assumption*: Back valuation with restatement. An entry landing on a closed day reopens that day and every day after it, and each reopened day is rechecked again against what we know now.
  The day is rechecked, never rewritten. We work out what should have been booked, diff it against what was booked, and append that difference as a new entry. No existing entry is touched so append only still holds.
  Rejected as-of day end (a closed day is final) because the fee E7 triggers would still stand after E9 takes E7 back, the customer pays 25.00 for an entry the bank itself reversed. Rejected recalculate and rewrite because it edits entries inside closed days.
  What this allows us to have : the ledger resolves to the same value whatever order the events arrive in. Two ledgers given the same facts agree even if one of them heard about E7 late.
  ALthough reopening every day back to the earliest value date is expensive, and in PROD this needs a bound on how far back a value date is allowed to go. For a six day window it is fine.
  Criteria 2 and 6 in REJECTED.md both are decided according to this choice. 
5. Fee trigger balance : the fee is booked with value date equal to the day assessed and closing counts every entry with value date less than or equal to that day, so the fee is inside the balance that decides it.
  *Assumption*: the day's own fee is left out of its own trigger. Fees from earlier days stay in, they are ordinary debits.
  So the balance we test and the balance we report differ by 25.00 on a fee day. Day 4 is -180.00 not -155.00 for this reason.
6. What restatement covers : entry 4 says a reopened day is rechecked, taken literally that rechecks the authorizations too. Auth-A was approved on Day 2 at 250.00, after E7 that day is -370.00, so a literal recheck declines it and unwinds E5.
  *Assumption*: restatement covers fee and interest only. An auth is decided on the balance at the moment it arrives and that decision stands.
  As a rule Assessments restate, decisions do not.
7. Rounding mode : amounts round to their own precision but the spec never says how a tie resolves.
  *Assumption*: half up away from zero, a tie going to the customer is easier to defend. No tie happens in the example so it will be pinned with a test built to tie.
8. Does interest compound : interest accrues daily but capitalizes as one credit at the end of Day 6, nothing says if a day's accrual joins the next day's balance.
  *Assumption*: it does not. Accruals are memo entries, recorded but they do not move the balance.
9. Hold release on a partial settlement : Auth-A holds 200.00 and settles 185.00, the spec does not say what happens to the remaining 15.00.
  *Assumption*: the remainder releases and the auth closes, the final amount was charged by the merchant.
10. How an auth ends when nothing settles it : no expiry horizon is given, so an approved hold that is never settled has no defined end.
  *Assumption*: we model the full set anyway since Deliverable 2 asks for it. Settled, partially settled, declined, expired, released or voided, over settled, and settlement with no auth.
  Nothing here expires, Auth-B is declined so it never opens. Over settlement does not occur, we reject it like an unknown auth rather than guess.
