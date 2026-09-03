# Hue bridge local API since 2017: CLIP v2 and what kelvin's modernization needs to know

Date: 2026-09-03

Question: How has the Philips Hue bridge local API changed since 2017, and what
does a modernization of kelvin need to know?

Sourcing note: the CLIP v2 API reference and design-guidance pages on
developers.meethue.com sit behind a free developer login. Every fetch of
`developers.meethue.com/develop/...` returns the login page. Where the primary
page is gated, this document says so and cites the best available secondary
source (community mirrors of the official OpenAPI spec, Home Assistant's
`aiohue`, openHAB's binding) — marked "secondary". Several claims are verified
empirically against the maintainer's own bridge (BSB002, swversion 1978074000,
apiversion 1.78.0, bridge id 001788FFFE412EFC) — marked "empirical".

## 1. CLIP v2 API: resource model, HTTPS, auth, v1 deprecation

CLIP v2 replaces v1's numeric ids with a typed resource model under
`/clip/v2/resource/{type}`. Resource types include `device`, `light`, `room`,
`zone`, `grouped_light`, and `scene`; every resource carries a UUID `id`.
Primary reference is gated; the community OpenHue spec mirrors it.
- https://developers.meethue.com/develop/hue-api-v2/api-reference/ (gated)
- https://www.openhue.io/api/openhue-api-1/light.md (secondary)

Each v2 resource carries an `id_v1` field mapping back to its v1 path (e.g.
`"id_v1": "/lights/25"`). Empirical: confirmed on the local bridge; this gives
kelvin a migration path for its stored numeric light ids.

v2 requires HTTPS. Signify announced in June 2025 that new firmware drops HTTP
entirely as of August 1 (2025), and the Bridge Pro ships HTTPS-only.
- https://developers.meethue.com/taxonomy/term/8 (primary news listing, item of 2025-06-04)
- https://github.com/ebaauw/homebridge-hue/issues/1219 (secondary, Bridge Pro HTTPS-only)

Empirical caveat: on 2026-09-03 the maintainer's BSB002 (firmware 1978074000,
July 2026) still answers full v1 API requests over plain HTTP with 200. The
announced HTTP shutdown has not reached this bridge yet. Do not rely on HTTP
surviving the next firmware.

Authentication: v2 sends the application key in a `hue-application-key` HTTP
header instead of embedding the v1 whitelist username in the URL path. An
existing v1 username works as the v2 application key. Empirical: kelvin's
stored v1 username authenticates `/clip/v2/resource/light` unchanged. Key
creation still uses the link-button flow (POST `/api`).
- https://github.com/home-assistant-libs/aiohue/blob/master/aiohue/v2/__init__.py (secondary; sets the header, HTTPS-only base URL)

v1 deprecation status: Signify's public "New Hue API" page states that HTTPS
replaced HTTP in November 2018, the v1 OAuth2 endpoint is deprecated since
July 2020, the Philips Hue SDK is deprecated since July 2019, UPnP discovery
was scheduled to be disabled in Q2 2022, new features (dynamic scenes,
gradient control, events) ship on v2 only, and "in the long term API v1 will
eventually be removed". No sunset date for local CLIP v1 has ever been
announced, and v1 still works on current BSB002 firmware (empirical,
2026-09-03).
- https://developers.meethue.com/new-hue-api/ (primary)

## 2. The event stream at /eventstream/clip/v2

The bridge exposes server-sent events at `https://<bridge>/eventstream/clip/v2`.
Subscribe with `Accept: text/event-stream` plus the `hue-application-key`
header, over TLS. Empirical: connecting this way to the local bridge succeeds;
the bridge greets with the SSE comment `: hi` and then stays silent until a
resource changes.
- https://developers.meethue.com/develop/hue-api-v2/core-concepts/ (gated primary)
- https://github.com/home-assistant-libs/aiohue/blob/master/aiohue/v2/controllers/events.py (secondary)

Event payload: each SSE message carries a JSON array of event containers with
`id` (UUID), `creationtime`, `type` (`add` | `update` | `delete` | `error`),
and `data` — an array of partial resource objects (the changed fields plus
`id`, `id_v1`, `type`). A light state change arrives as an `update` container
whose `data` entry carries the new `on`/`dimming`/`color`/`color_temperature`.
- https://github.com/home-assistant-libs/aiohue/blob/master/aiohue/v2/controllers/events.py (secondary)

Coalescing: the official docs (gated) state a 1-second rate limit on event
containers — if the same property changes twice within that window only the
last state is delivered, and changes to multiple resources within the window
arrive grouped in one container. This wording is reproduced independently in
several integrator forums.
- https://ccforum.userecho.com/communities/4/topics/10569-philips-hue-api-v2-events-server-side-events (secondary, quoting the gated docs)

Keepalive and reconnect: the bridge sends no periodic keepalive after the
initial `: hi`, so a dead connection is indistinguishable from a quiet one.
`aiohue` works around this by provoking a bridge event every 60 s (renaming a
geofence client) and treating 90 s of read silence as a dead connection, then
reconnecting with exponential backoff (2 s × attempts, capped at 600 s). It
sends `last-event-id` on reconnect but does not trust replay: after more than
60 s of disconnection it refetches the full state.
- https://github.com/home-assistant-libs/aiohue/blob/master/aiohue/v2/controllers/events.py (secondary)
- https://github.com/home-assistant-libs/aiohue/blob/master/aiohue/v2/__init__.py (secondary; full-state refetch rule)

Polling verdict: an initial GET of the full resource state is always required,
and a full refetch after long disconnects is prudent. Beyond that the event
stream replaces per-second polling entirely — Home Assistant's Hue integration
runs on events alone.

## 3. Rate limits and request patterns

The official v2 guidance (gated, "Hue System Performance" and core concepts)
recommends at most ~10 commands per second to individual `light` resources and
at most ~1 command per second to `grouped_light`. The numbers reproduce
consistently across independent integrations; an Elgato developer confirms the
bridge allows only ~1 change per second when the target is a room or zone.
- https://developers.meethue.com/develop/application-design-guidance/hue-system-performance/ (gated primary; kelvin's code already links it)
- https://github.com/homebridge/homebridge/issues/42 (secondary, 10/s lights, 1/s groups)
- https://github.com/elgatosf/streamdeck-philipshue-legacy/discussions/61 (secondary, 1/s room/zone)

An overloaded bridge answers 429 or 503. `aiohue` caps concurrent requests at
3 per host and retries 429/503 with linear backoff (0.25 s × attempt, max 25
retries) rather than enforcing per-resource-type rates.
- https://github.com/home-assistant-libs/aiohue/blob/master/aiohue/v2/__init__.py (secondary)

## 4. Color handling in v2

The `light` resource reports color temperature as `color_temperature.mirek`
plus `mirek_valid`, and per-light capability as
`color_temperature.mirek_schema.mirek_minimum`/`mirek_maximum`. Color lights
report `color.xy`, plus their own `color.gamut` (red/green/blue corner points)
and `gamut_type` (`A`, `B`, `C`, or `other`). Dimming reports `brightness`
(percent, 0–100, never 0) and `min_dim_level` (percent of maximum lumen at
minimum brightness). Empirical: the local bridge reports e.g.
`mirek_schema {200, 454}` and `gamut_type "A"` with explicit gamut corners per
light.
- https://www.openhue.io/api/openhue-api-1/light.md (secondary mirror of gated reference)
- Empirical: GET /clip/v2/resource/light on bridge 001788FFFE412EFC, 2026-09-03

`mirek_valid` is new: `mirek` is null/invalid whenever the light's current
color sits outside the CT spectrum — v2 distinguishes "what you asked for"
from "what the light can express" at least for CT.
- https://www.openhue.io/api/openhue-api-1/light.md (secondary)

Clamping: v1 documents that out-of-range xy values are mapped — "the closest
color to the coordinates will be chosen" — which is exactly why kelvin's
manual-change detection misfires: the bridge reports the clamped value, not
the one kelvin set.
- https://github.com/peter-murray/node-hue-api/blob/master/docs/lightState.md (secondary mirror of v1 doc wording)

Whether v2 reports the achievable-after-clamp value or echoes the requested
value is NOT documented in any accessible source (see Gaps). What v2 does
provide is the data to make the question moot: with per-light `gamut` and
`mirek_schema`, a client can clamp client-side before sending and predict the
achievable value exactly. Note also that v2 reports xy at 4 decimals and
brightness at 2 decimals (empirical), so state comparison needs tolerance.

## 5. Bridge discovery and TLS today

discovery.meethue.com (N-UPnP) is alive and returns the bridge list with a
`port` field. Empirical: on 2026-09-03 it answers 200 with
`[{"id":"001788fffe412efc","internalipaddress":"192.168.0.52","port":443}]`.
The endpoint replaced www.meethue.com/api/nupnp on July 1, 2018.
- https://discovery.meethue.com/ (primary, empirical)
- https://developers.meethue.com/news/ (primary, 2018 endpoint change)

mDNS is the recommended local mechanism: the bridge advertises
`_hue._tcp.local.`. Empirical: the local bridge announces itself as
"Hue Bridge - 412EFC" via `dns-sd -B _hue._tcp local.`. UPnP discovery is
deprecated and was scheduled for disablement in Q2 2022; the Bridge Pro ships
without UPnP or `description.xml` at all.
- https://developers.meethue.com/new-hue-api/ (primary, UPnP deprecation)
- https://github.com/ebaauw/homebridge-hue/issues/1219 (secondary, Bridge Pro)
- https://github.com/openhue/openhue-go (secondary; discovers via mDNS, falls back to discovery.meethue.com)

TLS: the bridge serves an ECDSA P-256 certificate with subject
`C=NL, O=Philips Hue, CN=<bridge id>` issued by
`C=NL, O=Philips Hue, CN=root-bridge`, valid to 2038-01-19. Empirical:
confirmed on the local bridge (CN=001788FFFE412EFC, uppercase). Validate by
trusting the Signify/Hue root CA (downloadable from the gated developer docs)
and checking that the certificate CN equals the expected bridge id; hostname
verification against the IP fails by design. Very old firmware used
self-signed per-bridge certificates; Signify states it has replaced these with
CA-issued certificates and advises dropping per-bridge certificate pinning in
favor of CA validation.
- Empirical: openssl s_client against 192.168.0.52:443, 2026-09-03
- https://developers.meethue.com/develop/application-design-guidance/using-https/ (gated primary)
- https://iotech.blog/posts/philips-https/ (secondary; CA from gated docs, CN=bridge id check)
- https://github.com/ebaauw/homebridge-hue/issues/1219 (secondary; Signify's pinning advice)

Separate cloud note: Signify announced (2024-10-10) that its cloud TLS
certificates move from DigiCert to Google Trust Services in 2025 — relevant to
discovery.meethue.com/remote API callers that pin roots, not to the local
bridge CA.
- https://developers.meethue.com/taxonomy/term/8 (primary)

## 6. Hue Bridge Pro and the classic bridge

Signify announced the Hue Bridge Pro on September 4, 2025: up to 150 lights
and 50+ accessories (about triple the BSB002), a "Hue Chip Pro" (~5× CPU,
~15× memory), Wi-Fi in addition to Ethernet, and MotionAware — motion sensing
derived from Zigbee RF disturbance between three or more lights.
- https://www.signify.com/global/our-company/news/press-releases/2025/20250904-far-more-than-intelligent-lighting-philips-hue-reimagines-smart-home (primary)

For API clients the Bridge Pro is a CLIP v2 bridge: model id BSB003, MAC
prefix C42996, HTTPS only, no UPnP/description.xml (mDNS is primary
discovery), CA-issued certificate, launch firmware swversion 2071096000 with
the same apiversion line (1.73.0 at launch) as the BSB002. MotionAware motion
areas are exposed through official v2 API resources exclusive to the Bridge
Pro (announced 2025-09-04).
- https://github.com/ebaauw/homebridge-hue/issues/1219 (secondary, model/MAC/HTTPS/discovery)
- https://developers.meethue.com/taxonomy/term/8 (primary, MotionAware API news)

The classic BSB002 still receives firmware and API updates: latest firmware
1.79.1978293000 on August 27, 2026, with feature releases (new device support)
as recently as July 2026, and platform-wide API changes (e.g. `device_mode`
deprecated in favor of `switch_mode`, June 6, 2026) continuing on both
bridges. Multi-bridge migration to a Bridge Pro shipped December 2025.
- https://www.philips-hue.com/en-us/support/release-notes/bridge (primary)
- https://www.philips-hue.com/en-us/support/release-notes/bridge-pro (primary)
- https://developers.meethue.com/taxonomy/term/8 (primary)

## 7. Go client libraries for CLIP v2

openhue/openhue-go is the most credible v2 library: Apache-2.0, generated
from the community-maintained OpenHue OpenAPI spec via oapi-codegen, mDNS
discovery with discovery.meethue.com fallback, link-button auth flow, typed
CRUD for lights/rooms/zones/scenes/devices. Latest release v0.4.0 on
April 16, 2025; ~136 stars; 9 known importers; requires Go 1.23+. Its public
API exposes no event-stream/SSE support as of v0.4.0.
- https://github.com/openhue/openhue-go (secondary/primary for itself)
- https://pkg.go.dev/github.com/openhue/openhue-go (primary for versions)

amimof/huego is v1-only and last saw meaningful activity in mid-2023.
kelvin's current dependency, stefanwichmann/go.hue (own fork), is v1-only
with its last commit in 2022. scttfrdmn/foxfire advertises full v2 coverage
including the event stream, TLS handling, and client-side rate limiting, but
has 0 stars, 0 forks, and no adoption — unproven.
- https://github.com/amimof/huego
- https://github.com/stefanwichmann/go.hue
- https://github.com/scttfrdmn/foxfire

Verdict: no mature Go library covers CLIP v2 including the event stream.
Realistic options: openhue-go for typed REST plus a hand-rolled SSE reader
(the protocol is a single long-lived GET; ~100 lines with net/http), or a
fully hand-rolled v2 client. The wire format is small: kelvin uses only
light, grouped_light, and scene-adjacent state.

## Gaps and absences

- The entire CLIP v2 reference, core concepts, migration guide, HTTPS
  guidance, discovery guidance, and system-performance pages on
  developers.meethue.com are login-gated. All claims sourced from them above
  rest on consistent secondary mirrors. The Wayback Machine is unreachable
  from this environment, so archived copies could not be checked.
- No primary or secondary source states whether a v2 light resource reports
  the achievable (clamped) value or echoes the requested value after an
  out-of-gamut `color.xy` or out-of-schema `mirek` write. This must be
  verified empirically on the maintainer's bridge before porting kelvin's
  manual-change detection (a one-off test: PUT an out-of-gamut xy, read back,
  compare). Absence of documentation is the finding.
- No announced sunset date exists for local CLIP v1 — only "will eventually
  be removed" (primary). Empirically v1 still works, even over HTTP, on
  BSB002 firmware from July 2026.
- Whether the Bridge Pro (BSB003) serves CLIP v1 at all is unverified; the
  homebridge-hue thread discusses v2 only. Treat v1-on-Bridge-Pro as
  unsupported.
- The exact official wording of the 10/s-per-light and 1/s-per-grouped_light
  limits could not be read from the primary page (gated); the numbers are
  corroborated by multiple independent integrators.
- Whether the bridge honors `last-event-id` replay of missed events is
  undocumented; aiohue sends it but refetches full state after >60 s gaps,
  implying replay is not to be trusted.

## Implications for kelvin

- Replace the 1 s polling loop with the v2 event stream plus one full-state
  GET at startup and after reconnects. The stream is push-based, coalesced to
  1 s, and covers everything kelvin polls for today.
- Keep the existing bridge username — it is the `hue-application-key`.
  Migrate stored numeric light ids via the `id_v1` field.
- Move to HTTPS now: trust the Hue `root-bridge` CA (or fall back to
  InsecureSkipVerify + CN==bridge-id check), and expect HTTP to disappear.
- Rebuild manual-change detection on v2 data: clamp targets client-side using
  each light's `gamut` and `mirek_schema` before sending, compare with
  rounding tolerance, and verify the clamp-echo behavior empirically once.
- Respect ~10/s per light and ~1/s per grouped_light; back off on 429/503.
- Discover via mDNS `_hue._tcp` first, discovery.meethue.com second; drop
  UPnP and the lanscan dependency.
- Library choice: openhue-go (typed, maintained, no SSE) plus a small own SSE
  reader, or a lean hand-rolled v2 client. Retire stefanwichmann/go.hue.
- BSB002 remains supported and updated; a v2 client gains Bridge Pro
  compatibility for free, a v1 client may not run there at all.
