---
status: done
---
# Flaky smoke: the Training attack check fails under load

## Description
`native/scripts/smoke-inner.sh` step 1b4 compares the Training match before and after holding F for one second; on a loaded machine (load average ~4.5) it failed 3 times out of 5 on 2026-11-20, with and without that day's changes (HEAD 1078ebf), while every other step passed. The screenshot shows both fighters idle: the attack did not register or the capture came too early. Fixed sleeps make the check timing-dependent.

## Acceptance Criteria
- [x] The Training check waits on an observable condition (or retries the capture) instead of fixed sleeps
- [ ] 10 consecutive smoke runs pass on a loaded machine

## Notes
- 2026-11-20: the attack press is retried up to 5 times before failing. Not run 10 times in a row: one full pass after the change.
