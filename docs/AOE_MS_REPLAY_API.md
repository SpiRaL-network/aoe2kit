# Microsoft AoE2 DE Web API — map

*Reverse-engineered 2026-07-21 official-site network capture + ~25 polite live probes +
web research (labeled DRAFT where unverified). Microsoft's `api.ageofempires.com` is the semi-public
successor to the defunct aoe2.net API — it powers the official ageofempires.com stats pages and the
community (aoe2insights, aoe2companion) has used it for years. **Respect posture:** read-only GET/POST,
low volume, cache responses, honor limits, no bulk redistribution.*

## Auth & headers (gateway-enforced)
- **No API key / no bearer token** — read-only stats are open.
- **`Origin: https://www.ageofempires.com` + `Referer: https://www.ageofempires.com/` REQUIRED** on the
  `api.ageofempires.com` JSON endpoints (a CORS preflight `OPTIONS` fires; the gateway checks Origin to
  deter cross-domain scraping). `GetMatchReplay` (the download) does not need it; the JSON POSTs do.
- `Content-Type: application/json`, browser-like `User-Agent`.
- **Rate limit (gem-DRAFT, unverified): ~60 req/min → 429; cache ≥120s.** Treat as the ceiling; stay well under.

## VERIFIED WORKING (live-confirmed 2026-07-21)

### 1. Replay download — the corpus workhorse
```
GET https://api.ageofempires.com/api/GameStats/AgeII/GetMatchReplay/?gameId=<G>&profileId=<P>&matchId=<G>
```
- All three params required; `matchId`==`gameId`; **`profileId` must be a participant** (else 404/16B).
- `200` → `application/zip`, `filename=AgeIIDE_Replay_<gameId>.zip` (the community filename origin).
- **Per-POV**: each participant's profileId returns THAT player's own recording (verified: same game, two
  pids → two files, camera tail=1 vs tail=8). Fetch all N pids → all N attention streams of one game.
- `aoe.ms/replay/?gameId=<G>&profileId=<P>` = 301 wrapper that appends matchId and forwards.
- Retention: ≥ weeks.

### 2. Player career stats — `GetFullStats`
```
POST https://api.ageofempires.com/api/GameStats/AgeII/GetFullStats
BODY {"profileId":"<pid>","gamertag":"","playerNumber":0,"gameId":0,"matchType":3}   (+Origin headers)
```
- **`profileId` as a STRING is the working key** (playerNumber/gamertag-only returned 204). `matchType` int.
- `200` → rich JSON:
  - `user`: profileId, userName, **elo**, **playerStanding** (percentile, e.g. 0.01 = top), avatarUrl
    (Steam), isHuman, matchReplayAvailable.
  - `careerStats`: **lifetime** totalGames/Wins, **unitsKilled, unitsLost, buildingsRaised, buildingsLost,
    wondersBuilt, castlesBuilt, trebsBuilt, farmsBuilt**, highScore{Total,Military,Economy,Technology}.
  - `mpStatList`: totalMatches/Wins/currentWinStreak for the matchType.
  - `mpMatches.matchList`: recent matches — **often empty now** (see hollow section).
- This is the player-profile + fingerprint-baseline source (elo/standing/lifetime aggregates + avatar).

## EXISTS BUT HOLLOW / DEPRECATED (200/204 with no usable data)
- `POST v2/AgeII/GetMPFull` — the leaderboard/index endpoint aoe2insights historically used. **Always 204**
  for every body shape tried (game/matchType/region/teamSize/page/count). Likely deprecated or moved.
- `POST v2/AgeII/GetMPMatchList {profileId,page,count}` — `200` but `{"matchList":[],"totalMatches":0}` even
  for an active top-ladder player. Endpoint alive, data hollow.
- `POST v2/AgeII/GetMPMatchDetail {matchId}` — `"Match not found"` (matchId ≠ gameId; no valid id source
  since the list endpoints are empty).

## 404 (do not exist under these paths)
`v2/AgeII/GetMPRankInfo`, `v2/AgeII/GetMPLeaderboard` (POST), `GameStats/AgeII/GetMatchStats`,
`GameStats/AgeII/GetRecentMatches`. (Gem claimed a GET `GetMPLeaderboard?board=3` fallback — unverified.)

## THE STRATEGIC PUNCHLINE
**Microsoft's API gives you (a) replay downloads and (b) career-aggregate player stats — but NOT per-match
stat breakdowns.** The per-match endpoints are hollow/dead. That is precisely why aoe2insights, aoe2companion,
and everyone else **parse the replays** for their Military/Economy/eAPM/civ breakdowns — the API won't hand
them over. **This independently validates AoE2Kit's entire approach:** parsing replays isn't the hard way,
it's the *only* way to per-match truth. And it triple-confirms the outcome-layer finding — per-match
kills/razes are in neither the replay's postgame block NOR the stats API; they're recoverable only by
parsing the command+checksum stream (our sync-matrix state deltas) or by XS-instrumenting owned scenarios.

## How the two community sites actually get their data
- **aoe2insights** (post-game): server-side crawler over this API (career stats via GetFullStats-family) +
  **downloads & parses replays** for per-match analysis. Maintains its own match DB/index (why it has match
  pages the hollow API can't produce). Its match pages are our practical gameId+profileId index (Cloudflare
  → reach via the shared-Chromium/CDP path, not curl).
- **aoe2lobby** (live/pre-game): NOT this HTTP API. A backend bot speaks the game's **Steam/Relic online
  lobby protocol** (relic-link successor), subscribes to the live open-lobby list, and rebroadcasts it to
  browsers over its own `/ws` WebSocket (message types `lobby_match_all/update/remove`, `lobby_elo_update`,
  `lobby_observers_update`, `spectate`). Complements insights: *what's starting now* vs *what happened*.

## AoE2Kit integration
- `kit replay fetch <gameId> --profile <pid> [--out dir]` — implemented. Stores
  files as `AgeIIDE_Replay_<gameId>_p<profileId>.zip` so same-match POVs do not
  overwrite each other. Existing files are cache hits unless `--force` is used.
- `kit replay fetch <gameId> --all-povs --profile <seedPid>` or
  `--from-roster <replay>` — implemented as a guarded first slice. It fetches a
  seed POV or reads an existing replay, parses positive `profileId` values from
  the roster, then fetches those POVs. If the replay roster parser exposes no
  profile IDs for that replay mode, it reports a warning instead of guessing.
- `kit player stats <profileId>` — implemented. Wraps `GetFullStats` with
  Origin/Referer headers and a 120-second default JSON cache. The verification
  label is explicit: `api_reported_career_aggregate_not_per_match_truth`.
- Future index bootstrap: gameIds/profileIds from (a) aoe2insights match pages
  via CDP, or (b) participant rosters inside already-fetched replays. One seed
  replay can snowball into all POVs plus every player's career stats when the
  roster parser exposes the profile IDs.

## Ethics (ratified)
Public-space rule: multiplayer replays are published performances; in-game fingerprinting is fair. **Doxxing
wall stands** — never bridge game identity to real-world identity. Downloads for analysis/fixtures, not bulk
redistribution. Stay far under rate limits; cache aggressively.
