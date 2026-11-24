---
status: done
---
# The Training smoke check still fails on CI

## Description
Item 073 retries the Training attack press up to 5 times (`native/scripts/smoke-inner.sh` step 1b4), yet the Native workflow's `linux-windows` job failed on it on 2026-11-23 (9fe4964) and 2026-11-24 (2d63e9a) with "the Training match did not run", then passed on a plain rerun both times; the same smoke run passes locally. Each failure blocks the Native run until someone reruns it.

## Acceptance Criteria
- [x] The check waits on something the game reports (a log line, a state the game writes for the smoke) rather than comparing screenshots after fixed sleeps, or the cause of the missed presses on CI is found and fixed
- [x] 10 consecutive Native runs pass the Training step

## Done
The cause was the check itself, not missed presses: the attack misses the distant dummy and its animation ends in 5 ticks, so the capture taken after it only differed from the one before when a HUD animation happened to change phase (reproduced locally: identical captures with the attack in the log). The step now waits for the line the game logs when P1's attack plays sound 5,0 (missing from the fixture), retrying the press up to 5 times. Verified: 10 Native runs on 1696d74 (one push, nine dispatched) passed the `linux-windows` job; one run's macOS job was cancelled for want of a free macOS runner and was rerun. Locally, 10 smoke runs with every core saturated never failed the Training step, but other steps did under that artificial load ("CPU did not move", "the sample lifebar is not drawn"); neither has failed on CI.
