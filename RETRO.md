# RETRO — bartr-movies, 1.0.0

> Per [docs/EXPERIMENT.md](docs/EXPERIMENT.md): the honest one. What broke,
> what surprised me, what I'd do differently.

## Run summary

| | |
|---|---|
| Stack | Go 1.26 · chi v5 · `log/slog` · `flag`+env · prometheus/client_golang · k3s + Kustomize |
| Sessions | 10 (entries 1, 2, 3 [bundles 4], 5 [retroactive stub], 6, 7, 8, 9, 10) |
| Tags shipped | `0.1.0`, `0.2.0`, `0.3.0`, `0.5.0`, `0.6.0`, `0.7.0`, `0.8.0`, `0.9.0`, `1.0.0` |
| First commit → `1.0.0` tag | ~12 hours wall-clock (May 6 18:01 -0500 → May 7 ~01:00 -0500) |
| Sum of session focus minutes | ~5 hours (rest is breaks between sessions) |
| AI assistant | GitHub Copilot (Claude-class agent) |
| Outcome | All 7 §14 boxes green on a freshly-wiped local k3s cluster |

## Headline measurement

Single 500m-CPU pod on the inner-loop cluster, in-cluster load via `webv`:

- p95 list routes: **0.10 – 0.35 ms** (target < 50 ms)
- p95 detail routes: **~0.10 ms** (target < 10 ms)
- Sustained throughput: **585 RPS, 0% 5xx, 0% 4xx** (target ≥ 500 RPS, < 1% errors)

50–500× under the spec target. See [docs/PERFORMANCE.md](docs/PERFORMANCE.md).

## Where the methodology helped

1. **Frame-before-implement caught at least three premature implementations.**
   Sessions 6, 7, and 8 each had a moment where the agent (or I) wanted
   to grab an adjacent feature; the explicit "Out of scope" line in the
   frame was what stopped it. Session 8 (webv CLI) shipped clean *because*
   bench tuning was explicitly held back to 0.9.0.
2. **The fit check is the most useful single ritual.** Session 2 was an
   8-minute frame, and recording "this frame is too small" was honest
   evidence — session 3 then bundled sessions 3 and 4. Session 9 broke
   its 90–120 min budget; recording that was more valuable than
   pretending it fit.
3. **The git-tag-per-session rule was load-bearing.** When I caught a
   session-log timing error in session 10 (claimed 175 min, real ~42),
   the tags + commit timestamps were the source of truth that let me
   reconcile honestly.
4. **`.copilot-tracking/` Research artifacts were unexpectedly useful**
   even when I didn't re-read them. Writing the research file forced me
   to articulate the constraint surface before touching code.

## Where it got in the way

1. **Thread reuse vs. fresh thread.** Sessions 9 and 10 reused the
   session-8 chat thread because the cluster context was heavy
   (Prometheus selectors, Traefik entrypoints, webv CLI internals).
   Methodology default is one session per thread for clean signal; I
   logged the deviation both times. This is real friction the
   methodology should probably address — context-heavy late sessions
   can't afford a cold-start agent.
2. **Session 5 has no entry.** I bundled sessions 3+4 in one entry,
   then session 5 (OpenAPI / Swagger / robots.txt) shipped tag `0.5.0`
   without anyone writing the frame. I caught it during the session-10
   reconciliation and added a retroactive stub. The methodology says
   "tag every session" — it should also say "log every tag."
3. **Estimated durations decay fast.** I wrote "~175 min" for session 9
   from memory after the fact and it was off by a factor of four.
   Lesson: log Start time at frame, log End time at close ritual, do
   not estimate after the conversation ends.
4. **Inner-loop verification has a known race I keep hitting.** The
   first `make verify` after a fresh `make deploy` 502/503s during the
   rolling update; second one passes. I documented it in repo memory
   in session 6 and have hit it again in sessions 7, 8, 9, 10. The
   right fix is to make `make verify` retry once internally, not to
   keep re-learning the lesson.

## Surprises

1. **Go is genuinely fast.** I knew it would clear the 50 ms target;
   I did not expect to clear it by 500×. Detail routes hit ~95 µs p95
   on a 500m-CPU pod with no caching layer. This re-framed
   [docs/PERFORMANCE.md](docs/PERFORMANCE.md) into the headroom story.
2. **Default Prometheus histogram buckets lie about sub-ms services.**
   `prometheus.DefBuckets` starts at 5 ms; every request landed in the
   smallest bucket and `histogram_quantile` interpolated ~4.75 ms for
   every route. The dashboard showed a uniform-but-wrong story until
   session 9 fixed it. I would not have spotted this without writing
   the user-facing ask "are requests being cached?" — that question
   forced the right diagnosis.
3. **Kernel timer slice quantizes Go `time.Sleep` to ~1 ms.**
   Single-thread RPS on the load generator jumped in coarse steps
   (800 / 428 / 293) with no value near 500. Two threads with a 3 ms
   sleep was the cleanest landing. I assumed sub-ms sleep would Just
   Work; it does not.
4. **AI agents are very willing to over-document.** Twice the agent
   wanted to create new markdown files to summarize changes that were
   already in the git history. Caught both; the explicit "do not
   create markdown files to document changes unless requested"
   guardrail in my customization paid off.

## What I'd do differently

1. **Adopt a "log Start at frame, End at close ritual" rule, not a
   post-hoc estimate rule.** And cross-check against git timestamps
   in the close ritual itself, not at release time.
2. **Make `make verify` self-healing for the rolling-update race.**
   One built-in retry with a 5-second pause would erase a recurring
   inner-loop friction.
3. **Resist over-engineering the dashboard early.** I rebuilt panel
   layouts three times across sessions 7, 8, and 9. Most of that
   churn was because the histogram-bucket bug made the visuals look
   broken. Fix instrumentation before tuning visuals.
4. **Open the RETRO file at session 1, not session 10.** The strongest
   lessons happened mid-experiment; reconstructing them at the end
   is harder than appending one bullet per session as you go.

## One-liner for the submissions tracker

> Sessions + RPI got me to 1.0.0 in ~12 hours wall-clock / ~5 hours
> focus, hitting spec p95 targets by 50–500× and the 500 RPS goal
> by ~17%. The fit check + frame-before-implement caught three
> premature implementations; thread reuse on context-heavy late
> sessions was the main methodology friction.
