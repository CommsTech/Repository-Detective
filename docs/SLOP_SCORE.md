# Slop score

A deterministic quality signal for patterns that generated code tends to leave
behind: placeholders, commented-out code, assertions that cannot fail, swallowed
errors, debug output, emoji in source, and marketing filler in prose.

**What it is:** a heuristic cleanliness score from 0 to 100 for a set of source
files, plus the individual tells that produced it. Higher is cleaner.

**What it is not:** a security verdict, a correctness proof, or a substitute for
tests and review. A score of 100 means no known pattern matched.

## Scoring

Each tell carries a weight by severity (high 8, medium 4, low 1.5, info 0.5).
The penalty is normalised per 1,000 lines so a large repository is not punished
for size alone, and subtracted from 100. Grades: A >= 90, B >= 80, C >= 70,
D >= 60, otherwise F.

## Bounds and fail-closed behaviour

The analyser refuses to run past its budgets: 4,000 files, 1 MiB per file, 2,000
findings, 20 seconds. Exceeding a budget returns an explicit incomplete result.
It never reports a truncated analysis as clean.

## Current status

Implemented as the `slopscore` package with unit tests and wired into the scan
pipeline behind an opt-in switch:

```dotenv
REPOSITORY_DETECTIVE_SLOP_SCORE_ENABLED=true
```

When enabled, each scan emits `slopscore` findings (auditor type `slopscore`)
and logs a line with the score, grade and top tells. It is **off by default** so
it cannot flood an existing findings queue, and it does not feed policy gates or
issue filing on its own. Score display in the UI is not implemented yet.

The tell taxonomy is adapted from the Codecraft Pro craft guidance (slop tells),
reimplemented here rather than vendored.
