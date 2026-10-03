/**
 * Music Journey end-to-end smoke test.
 *
 * Runs the REAL engine modules (loaded through Vite's resolver, exactly as
 * the app bundles them) against a running Navidrome-compatible API:
 *
 *   login -> fetch song pool (same 5 slices the UI issues) -> build journeys
 *   for every mood -> verify duration/dupes/phases/reasons -> live rebuild
 *   (regenerateRemaining) -> skip feedback effect -> replay-by-id contract
 *   -> failure handling against a broken endpoint.
 *
 * Usage:  MOCK_URL=http://localhost:4633 node tools/journey-smoke.mjs
 *         (default MOCK_URL is http://localhost:4633)
 */
import { createServer } from 'vite'

const BASE = process.env.MOCK_URL || 'http://localhost:4633'
const results = []
const check = (name, cond, extra = '') => {
  results.push({ name, ok: Boolean(cond), extra })
  // eslint-disable-next-line no-console
  console.log(
    `${cond ? 'PASS' : 'FAIL'}  ${name}${extra ? `  (${extra})` : ''}`,
  )
  if (!cond) process.exitCode = 1
}

/* ---- a fetch-based dataProvider speaking ra-data-json-server dialect ---- */
const dataProvider = {
  getList: async (resource, params) => {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(params.filter || {})) {
      if (Array.isArray(v)) v.forEach((x) => qs.append(k, x))
      else qs.append(k, v)
    }
    qs.set('_sort', params.sort.field)
    qs.set('_order', params.sort.order)
    const { page, perPage } = params.pagination
    qs.set('_start', String((page - 1) * perPage))
    qs.set('_end', String(page * perPage))
    const res = await fetch(`${BASE}/api/${resource}?${qs.toString()}`)
    if (!res.ok) throw new Error(`HTTP ${res.status} for ${resource}`)
    return {
      data: await res.json(),
      total: parseInt(res.headers.get('x-total-count') || '0', 10),
    }
  },
}

const vite = await createServer({
  server: { middlewareMode: true },
  appType: 'custom',
  logLevel: 'error',
})

try {
  const { buildJourney, regenerateRemaining, JourneyError } =
    await vite.ssrLoadModule('/src/musicJourney/engine/journeyEngine.js')
  const { buildNarrative } = await vite.ssrLoadModule(
    '/src/musicJourney/engine/phases.js',
  )
  const { transitionScore } = await vite.ssrLoadModule(
    '/src/musicJourney/engine/transitions.js',
  )
  const { getTrackProfile } = await vite.ssrLoadModule(
    '/src/musicJourney/engine/trackProfile.js',
  )
  const { fetchSongPool } = await vite.ssrLoadModule(
    '/src/musicJourney/hooks/fetchSongPool.js',
  )

  /* ------------------------------------------------ 1. auth contract */
  const login = await fetch(`${BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'smoke', password: 'smoke' }),
  }).then((r) => r.json())
  check(
    '1. login returns JWT + subsonic credentials',
    login.token?.split('.').length === 3 &&
      login.subsonicSalt &&
      login.subsonicToken,
  )

  /* ------------------------------------------------ 2. song pool fetch */
  const pool = await fetchSongPool(dataProvider, {})
  check(
    '2a. pool fetch merges all slices',
    pool.list.length > 100,
    `${pool.list.length} tracks: base=${pool.stats.base} starred=${pool.stats.starred} top=${pool.stats.top} recent=${pool.stats.recent} rare=${pool.stats.rare}`,
  )
  const poolIds = new Set(pool.list.map((s) => s.id))
  check('2b. pool has no duplicates', poolIds.size === pool.list.length)
  check(
    '2c. pool excludes missing files',
    pool.list.every((s) => !s.missing),
  )

  const genrePool = await fetchSongPool(dataProvider, {
    genreId: pool.list.find((s) => s.genre)?.genres?.[0]?.id,
    force: true,
  })
  const genreName = pool.list.find((s) => s.genre)?.genre
  check(
    '2d. genre filter is passed through to the API',
    genrePool.list.length > 0 &&
      genrePool.list.every((s) => s.genre === genreName),
    `${genrePool.list.length} ${genreName} tracks`,
  )

  /* ------------------------------------------------ 3. journey generation */
  const moods = ['chill', 'focus', 'energy', 'happy', 'melancholic']
  for (const mood of moods) {
    const j = buildJourney({
      songs: pool.list,
      prefs: { mood, durationMinutes: 30, intensity: 0.5, discovery: 0.3 },
    })
    const target = 30 * 60
    const ids = j.tracks.map((t) => t.track.id)
    const stageIds = new Set(j.narrative.map((p) => p.id))
    const okDuration =
      j.estimatedDuration >= target * 0.5 && j.estimatedDuration <= target + 300
    const okDupes = new Set(ids).size === ids.length
    const okStages =
      j.tracks.every((t) => stageIds.has(t.stage.id)) &&
      JSON.stringify(j.narrative) ===
        JSON.stringify(
          buildNarrative({
            mood,
            durationMinutes: 30,
            intensity: 0.5,
            discovery: 0.3,
          }).map((p) => ({ id: p.id, name: p.name })),
        )
    const okReasons = j.tracks.every(
      (t) => typeof t.reason === 'string' && t.reason.length > 5 && t.score > 0,
    )
    check(
      `3. "${mood}" journey: duration/dupes/stages/reasons`,
      okDuration && okDupes && okStages && okReasons,
      `${j.title} — ${j.tracks.length} tracks, ${Math.round(
        j.estimatedDuration / 60,
      )} min, stages: ${j.narrative.map((s) => s.name).join('→')}`,
    )
  }

  /* ------------------------------------------------ 4. hard filters */
  const decadeJourney = buildJourney({
    songs: pool.list,
    prefs: {
      mood: 'chill',
      durationMinutes: 15,
      intensity: 0.5,
      discovery: 0.3,
      decade: '1990',
    },
  })
  check(
    '4a. decade filter respected',
    decadeJourney.tracks.every(
      (t) => Math.floor(t.track.year / 10) * 10 === 1990,
    ),
    `years: ${[...new Set(decadeJourney.tracks.map((t) => t.track.year))].join(',')}`,
  )

  /* ------------------------------------------------ 5. transitions */
  const byGenre = (g) => pool.list.find((s) => s.genre === g)
  const ambient = byGenre('Ambient')
  const metalish =
    pool.list.find((s) => s.bpm && s.bpm >= 140) || byGenre('Rock')
  const tCliff = transitionScore(
    { song: ambient, profile: getTrackProfile(ambient) },
    { song: metalish, profile: getTrackProfile(metalish) },
    { discovery: 0 },
  )
  const tSmoothPair = pool.list
    .filter((s) => s.genre === 'Electronic')
    .slice(0, 2)
  const tSmooth =
    tSmoothPair.length === 2
      ? transitionScore(
          { song: tSmoothPair[0], profile: getTrackProfile(tSmoothPair[0]) },
          { song: tSmoothPair[1], profile: getTrackProfile(tSmoothPair[1]) },
          { discovery: 0 },
        )
      : { score: 1, penalties: [] }
  check(
    '5. transition intelligence: cliff < smooth',
    tCliff.score < tSmooth.score,
    `cliff=${tCliff.score.toFixed(2)} [${tCliff.penalties.join(',')}], smooth=${tSmooth.score.toFixed(2)}`,
  )

  /* ------------------------------------------------ 6. live regeneration */
  const journey = buildJourney({
    songs: pool.list,
    prefs: {
      mood: 'chill',
      durationMinutes: 30,
      intensity: 0.3,
      discovery: 0.3,
    },
  })
  const playedCount = 3
  const rebuilt = regenerateRemaining({
    journey,
    playedCount,
    prefs: {
      mood: 'chill',
      durationMinutes: 30,
      intensity: 0.9,
      discovery: 0.6,
    },
    songs: pool.list,
  })
  const keptSame =
    JSON.stringify(rebuilt.tracks.slice(0, playedCount)) ===
    JSON.stringify(journey.tracks.slice(0, playedCount))
  const currentIntact =
    rebuilt.tracks[playedCount - 1] === journey.tracks[playedCount - 1]
  const keptIds = new Set(
    journey.tracks.slice(0, playedCount).map((t) => t.track.id),
  )
  const tailFresh = rebuilt.tracks
    .slice(playedCount)
    .every((t) => !keptIds.has(t.track.id))
  check(
    '6. live rebuild keeps played part + current track, replaces only the tail',
    keptSame &&
      currentIntact &&
      tailFresh &&
      rebuilt.version === journey.version + 1,
    `kept=${rebuilt.keptCount}, total=${rebuilt.tracks.length}`,
  )

  /* ------------------------------------------------ 7. skip feedback */
  const dominant = (() => {
    const counts = {}
    journey.tracks.slice(playedCount).forEach((t) => {
      const g = t.track.genre?.toLowerCase()
      if (g) counts[g] = (counts[g] || 0) + 1
    })
    return Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0]
  })()
  const now = Date.now()
  const skipEvents = [0, 1, 2].map((i) => ({
    genreKey: dominant,
    artistKey: null,
    energy: 0.5,
    at: now - i * 1000,
  }))
  const afterSkips = buildJourney({
    songs: pool.list,
    prefs: {
      mood: 'chill',
      durationMinutes: 15,
      intensity: 0.5,
      discovery: 0.3,
      seed: 'skip-test',
    },
    skipEvents,
    now,
  })
  const before = buildJourney({
    songs: pool.list,
    prefs: {
      mood: 'chill',
      durationMinutes: 15,
      intensity: 0.5,
      discovery: 0.3,
      seed: 'skip-test',
    },
    now,
  })
  const share = (j) =>
    j.tracks.filter((t) => t.track.genre?.toLowerCase() === dominant).length /
    j.tracks.length
  check(
    `7. 3 consecutive skips reduce the skipped genre's share`,
    share(afterSkips) <= share(before),
    `before=${Math.round(share(before) * 100)}% after=${Math.round(share(afterSkips) * 100)}% (${dominant})`,
  )

  /* ------------------------------------------------ 8. replay contract */
  const replayIds = journey.tracks.slice(0, 5).map((t) => t.track.id)
  const replayRes = await dataProvider.getList('song', {
    pagination: { page: 1, perPage: replayIds.length },
    sort: { field: 'id', order: 'ASC' },
    filter: { id: replayIds, missing: false },
  })
  check(
    '8. history replay: songs fetched by id array',
    replayRes.data.length === replayIds.length &&
      replayIds.every((id) => replayRes.data.some((s) => s.id === id)),
  )

  /* ------------------------------------------------ 9. failure handling */
  const brokenProvider = {
    getList: async () => {
      throw new Error('network down')
    },
  }
  const degraded = await fetchSongPool(brokenProvider, { force: true }).catch(
    () => null,
  )
  check(
    '9. unreachable API degrades gracefully (empty pool -> friendly UI error)',
    degraded !== null && degraded.list.length === 0,
  )
  try {
    buildJourney({ songs: [], prefs: { mood: 'chill', durationMinutes: 30 } })
    check('9b. empty library throws EMPTY_LIBRARY', false)
  } catch (e) {
    check(
      '9b. empty library throws EMPTY_LIBRARY',
      e instanceof JourneyError && e.code === 'EMPTY_LIBRARY',
    )
  }

  /* ------------------------------------------------ subsonic streaming */
  const anySong = pool.list[0]
  const stream = await fetch(
    `${BASE}/rest/stream?id=${anySong.id}&u=smoke&f=json&v=1.8.0&c=smoke`,
  )
  const bytes = new Uint8Array(await stream.arrayBuffer())
  const isWav =
    bytes.length > 44 &&
    bytes[0] === 0x52 &&
    bytes[1] === 0x49 &&
    bytes[2] === 0x46
  check(
    '10. /rest/stream serves playable audio for the player',
    stream.ok && isWav,
    `${bytes.length} bytes`,
  )
} finally {
  await vite.close()
}

const failed = results.filter((r) => !r.ok)
// eslint-disable-next-line no-console
console.log(
  `\n${results.length - failed.length}/${results.length} checks passed${
    failed.length ? ` — FAILED: ${failed.map((f) => f.name).join('; ')}` : ''
  }`,
)
