---
status: todo
---
# The Training smoke check still fails on CI

## Description
Item 073 retries the Training attack press up to 5 times (`native/scripts/smoke-inner.sh` step 1b4), yet the Native workflow's `linux-windows` job failed on it on 2026-11-23 (9fe4964) and 2026-11-24 (2d63e9a) with "the Training match did not run", then passed on a plain rerun both times; the same smoke run passes locally. Each failure blocks the Native run until someone reruns it.

## Acceptance Criteria
- [ ] The check waits on something the game reports (a log line, a state the game writes for the smoke) rather than comparing screenshots after fixed sleeps, or the cause of the missed presses on CI is found and fixed
- [ ] 10 consecutive Native runs pass the Training step
