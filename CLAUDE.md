# CLAUDE.md

Kelvin is a Go daemon that adjusts Philips Hue lights to schedules derived
from local sunrise and sunset times. Single binary, one `main` package.

## Gates

Run all five before every commit; all must pass clean:

- `go build ./...`
- `go vet ./...`
- `gofmt -l .` — must print nothing
- `go test ./...` — the suite runs offline; no test may need the bridge or the network
- `go mod tidy` — must leave no diff

Additionally, `scripts/docker-smoke.sh` (needs Docker) must pass for commits
touching the Dockerfile, `.goreleaser.yaml`, the web interface, or
configuration I/O — and always before tagging a release. CI runs it on every
push.

## Conventions

- Commit subjects follow conventional commits: `type(scope): imperative description`,
  at most ~65 characters. The scope is the issue number where one exists
  (`fix(#126): …`), an area otherwise.
- One intent per commit: fixes, documentation, and version bumps land separately.
- Open work lives in GitHub issues, never in markdown checklists.
- Recorded decisions live in `DECISIONS.md`. Research notes live in `research/`
  as cited Markdown files.
- Commit locally on green gates; push and tag only on explicit instruction.

## Domain pitfalls

- The value `-1` for color temperature or brightness means "do not manage this
  property". This sentinel flows unlabeled from the configuration through
  `Schedule`, `Interval`, and `LightState`. Handle it in every calculation.
- The Hue bridge clamps out-of-gamut colors and out-of-range values, then
  reports the clamped state. Compare against the achievable value, never the
  requested one — a mismatch is misread as a manual user change and disables
  automation (see issue #126).
- Color temperatures below roughly 1700K convert to xy coordinates outside
  every Hue gamut.
- The web interface loads templates from the working-directory-relative path
  `gui/`; the binary must start in its install directory.
- `config.json` carries a `version` field; migrations run on read.
  `latestConfigurationVersion` must equal the newest migration's output version,
  or fresh configurations get migrated and silently rewritten.
- The auto-updater refuses major-version jumps unless started with
  `-forceUpdate`. This guard is what holds the v1 fleet back at v1.3.x once
  a v2.0.0 release exists.

## Current campaign

Kelvin is being modernized into an event-driven Hue CLIP v2 client. The scope,
sequence, and rejected alternatives are recorded in `DECISIONS.md` under
"Modernization campaign". The work items live in GitHub issues. Facts about
the CLIP v2 API are in `research/hue-api-v2.md` — read it before touching
bridge communication.
