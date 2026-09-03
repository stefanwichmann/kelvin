# Decisions

The decision log of kelvin. New entries at the top, format: `## <short title>`
with one-line **Decision** / **Why** / **Date**. `grep '^## '` is the index.

## Updater hardening stops at crash fixes

- **Decision:** The self-updater receives crash fixes only: a corrupt archive
  or unexpected release metadata must never terminate the daemon. No checksum
  or signature verification is added. One residual risk is accepted and
  recorded rather than mitigated: a compromised GitHub release account could
  ship a malicious same-major release to the installed v1 fleet.
- **Why:** The updater is scheduled for removal in v2.0.0 (see the campaign
  entry below), so every line invested in it is deleted within months. HTTPS
  already covers tampering in transit. An unsigned `checksums.txt` published
  by the same account adds no protection against account compromise — the
  attacker regenerates it — and offline-key signing, the only real mitigation,
  is not worth building into a dying subsystem. The updater's existing
  major-version guard keeps v1 installations from auto-updating into v2.0.0.
- **Date:** 2026-09-03

## Modernization campaign: kelvin becomes an event-driven CLIP v2 client

- **Decision:** Kelvin migrates from polling the Hue CLIP v1 API to consuming
  the CLIP v2 server-sent event stream, as one declared campaign with the
  following settled points. (1) The migration is the centerpiece; work on
  v1-only internals that it deletes is skipped. (2) Before migration work
  starts, four fixes land on the current code: web-interface security
  (loopback bind by default, authentication, masked bridge credential,
  origin checks), the gamut-clamp false manual-change detection, the
  configuration version constant, and updater crash safety. (3) v2.0.0 is a
  clean cut: it requires a v2-capable bridge, and the last v1.x release is
  final for v1-only setups. (4) Kelvin owns a lean CLIP v2 client built on
  the standard library plus a small SSE reader; the `go.hue` dependency
  retires. (5) Schedules bind to rooms and zones; a one-time migration maps
  stored device ids to rooms via the bridge's `id_v1` fields. Light state is
  still tracked per light internally. (6) The manual-override contract is
  unchanged but made reliable: kelvin clamps targets client-side using each
  light's reported gamut and mirek range and compares against the predicted
  readback, and a connectivity loss no longer counts as a power cycle.
  (7) The web interface carries over with embedded templates, adapted to
  rooms; no redesign. (8) The self-updater is removed in v2.0.0;
  distribution is GitHub releases and Docker. (9) v2.0.0 ships behavioral
  parity plus loud warnings for schedule entries that can never fire; every
  feature request lands after v2.0.0 on the new base. (10) Build order: a
  throwaway spike (event-stream client plus a clamp-readback experiment on a
  real bridge), then incremental work on master with the gates green at every
  commit, then at least a week of shadow mode — the new core computes and
  logs decisions next to the running v1 instance without sending — before
  cutover and tag.
- **Why:** The 2026-09-03 deep analysis traced roughly half of the open issue
  backlog to two roots the v1 polling architecture forces: out-of-gamut warm
  color temperatures misread as manual changes, and unsynchronized shared
  state between the polling loop and the web interface. The API research
  (`research/hue-api-v2.md`) shows the event stream replaces polling
  outright, the existing credential carries over as the v2 application key,
  `id_v1` gives a mechanical id migration, v2 exposes per-light gamut and
  mirek ranges that make clamp prediction exact, and no maintained Go
  library covers the event stream — so a lean own client is the smallest
  ownership burden. Rejected alternatives: modernizing in place on v1
  (invests in code the migration deletes), a full parallel rewrite (two
  codebases for a daily-use service), a dual v1/v2 backend (doubles the test
  surface to serve bridges unsupported since 2020), adopting a generated
  REST client (covers only the half kelvin needs least), and shipping
  schedule-semantics changes inside the same release as the platform swap.
- **Date:** 2026-09-03
