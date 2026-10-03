/**
 * Dev-only Navidrome compatibility harness.
 *
 * Implements the same HTTP contract the real Navidrome server exposes for
 * the endpoints this UI uses (native /api REST, /auth, /rest Subsonic and
 * the /api/events SSE stream), backed by an in-memory demo library with
 * real streaming WAV audio and generated cover art.
 *
 * It exists ONLY so the UI can be developed/demoed without a Navidrome
 * instance (e.g. CI sandboxes). It is never used in production: the real
 * app always talks to a real Navidrome server through the same endpoints.
 *
 * Usage:  node tools/dev-navidrome-mock.mjs   (listens on :4633 by default)
 */
import http from 'node:http'
import { URL } from 'node:url'
import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import zlib from 'node:zlib'

const PORT = parseInt(process.env.MOCK_PORT || '4633', 10)
const AUDIO_CACHE_DIR = path.join(os.tmpdir(), 'navidrome-mock-audio')
fs.mkdirSync(AUDIO_CACHE_DIR, { recursive: true })

/* ------------------------------------------------------------------ */
/* Demo library                                                        */
/* ------------------------------------------------------------------ */

const GENRES = [
  {
    name: 'Electronic',
    bpm: [118, 138],
    artists: [
      'Neon Circuit',
      'Velvet Static',
      'Aurora Signal',
      'Pulse Meridian',
    ],
  },
  {
    name: 'Rock',
    bpm: [110, 150],
    artists: [
      'Granite Echo',
      'The Hollow Pines',
      'Scarlet Mile',
      'Iron Orchard',
    ],
  },
  {
    name: 'Jazz',
    bpm: [90, 130],
    artists: ['Blue Note Collective', 'Mira Solano Trio', 'The Late Set'],
  },
  {
    name: 'Ambient',
    bpm: [60, 85],
    artists: ['Still Water', 'North of Nowhere', 'Glass Horizon'],
  },
  {
    name: 'Pop',
    bpm: [100, 125],
    artists: ['Cassette Youth', 'Lumen', 'Paper Satellites', 'June Ave'],
  },
  {
    name: 'Hip-Hop',
    bpm: [85, 105],
    artists: ['MC Ledger', 'Night Shift Poets', 'Concrete Bloom'],
  },
]

const TITLES_A = [
  'Midnight',
  'Golden',
  'Silent',
  'Electric',
  'Fading',
  'Neon',
  'Slow',
  'Distant',
  'Burning',
  'Quiet',
  'Wild',
  'Frozen',
  'Velvet',
  'Broken',
  'Silver',
  'Hidden',
]
const TITLES_B = [
  'Drive',
  'Horizon',
  'Rain',
  'Signal',
  'Echoes',
  'Skyline',
  'Garden',
  'Tides',
  'Motion',
  'Lights',
  'Streets',
  'Mirrors',
  'Currents',
  'Skies',
  'Embers',
  'Pulse',
]

// Deterministic pseudo-random so the demo library is stable between runs.
let seed = 20260203
const rnd = () => {
  seed = (seed * 1103515245 + 12345) % 2147483648
  return seed / 2147483648
}
const pick = (arr) => arr[Math.floor(rnd() * arr.length)]

const albums = []
const songs = []
let albumSeq = 0
for (const genre of GENRES) {
  const genreAlbumCount = 12
  for (let a = 0; a < genreAlbumCount; a++) {
    albumSeq++
    const artist = pick(genre.artists)
    const year = 1975 + Math.floor(rnd() * 50)
    const album = {
      id: `al-${albumSeq}`,
      name: `${pick(TITLES_A)} ${pick(TITLES_B)}`,
      albumArtist: artist,
      artist,
      genre: genre.name,
      year,
      songCount: 0,
      duration: 0,
      playCount: 0,
      rating: 0,
      starred: rnd() < 0.15,
      compilation: false,
      createdAt: new Date(Date.UTC(year, 0, 1)).toISOString(),
      updatedAt: new Date(Date.UTC(year + 1, 5, 1)).toISOString(),
    }
    albums.push(album)
    const trackCount = 4 + Math.floor(rnd() * 4)
    for (let t = 0; t < trackCount; t++) {
      const duration = Math.round(8 + rnd() * 12) // seconds (short demo audio)
      const bpm = Math.round(
        genre.bpm[0] + rnd() * (genre.bpm[1] - genre.bpm[0]),
      )
      const playCount = rnd() < 0.45 ? Math.floor(rnd() * 40) : 0
      const daysAgo = Math.floor(rnd() * 400)
      songs.push({
        id: `sg-${albumSeq}-${t}`,
        title: `${pick(TITLES_A)} ${pick(TITLES_B)}`,
        album: album.name,
        albumId: album.id,
        artist,
        artistId: `ar-${genre.name}-${artist.replace(/\s+/g, '-')}`,
        albumArtist: artist,
        albumArtistId: `ar-${genre.name}-${artist.replace(/\s+/g, '-')}`,
        genre: genre.name,
        genres: [{ id: `g-${genre.name}`, name: genre.name }],
        year,
        originalYear: year,
        releaseYear: year,
        bpm: rnd() < 0.8 ? bpm : null,
        duration,
        trackNumber: t + 1,
        discNumber: 1,
        playCount,
        playDate: playCount
          ? new Date(Date.now() - daysAgo * 86400000).toISOString()
          : null,
        rating: rnd() < 0.2 ? 1 + Math.floor(rnd() * 5) : 0,
        starred: rnd() < 0.18,
        starredAt: null,
        bitRate: 320,
        sampleRate: 44100,
        channels: 2,
        suffix: 'wav',
        codec: 'WAV',
        size: duration * 16000,
        path: `/music/${genre.name}/${album.name}/${t + 1}.wav`,
        compilation: false,
        hasCoverArt: true,
        libraryId: 1,
        libraryName: 'Demo Music',
        missing: false,
        comment: '',
        lyrics: '',
        createdAt: new Date(Date.now() - 600 * 86400000).toISOString(),
        updatedAt: new Date(Date.now() - 30 * 86400000).toISOString(),
        rgTrackGain: null,
        rgTrackPeak: null,
      })
      album.songCount++
      album.duration += duration
    }
  }
}
// Some listening concentration: make a few artists clearly "loved".
for (const s of songs) {
  if (s.artist === 'Neon Circuit' || s.artist === 'Still Water') {
    s.playCount += 15
    s.starred = true
    s.rating = Math.max(s.rating, 4)
    s.playDate = new Date(
      Date.now() - Math.floor(rnd() * 60) * 86400000,
    ).toISOString()
  }
}
const artists = [
  ...new Map(GENRES.flatMap((g) => g.artists).map((a) => [a, a])).values(),
].map((name, i) => ({
  id: `ar-${i}`,
  name,
  albumCount: albums.filter((al) => al.albumArtist === name).length,
  songCount: songs.filter((s) => s.artist === name).length,
  playCount: songs
    .filter((s) => s.artist === name)
    .reduce((x, s) => x + s.playCount, 0),
  rating: 0,
  starred: false,
  biography: '',
  imageUrl: '',
  fullText: name,
}))
const genreRecords = GENRES.map((g) => ({
  id: `g-${g.name}`,
  name: g.name,
  songCount: songs.filter((s) => s.genre === g.name).length,
}))

/* ------------------------------------------------------------------ */
/* Audio + artwork generation                                          */
/* ------------------------------------------------------------------ */

const wavCache = new Map()
const wavFor = (song) => {
  if (wavCache.has(song.id)) return wavCache.get(song.id)
  const file = path.join(AUDIO_CACHE_DIR, `${song.id}.wav`)
  if (!fs.existsSync(file)) {
    const sampleRate = 8000
    const frames = sampleRate * song.duration
    const data = Buffer.alloc(frames * 2)
    const freq =
      220 + (crypto.createHash('md5').update(song.id).digest()[0] % 8) * 55
    for (let i = 0; i < frames; i++) {
      const t = i / sampleRate
      const env = Math.min(1, t * 2, (song.duration - t) * 2)
      const v =
        Math.sin(2 * Math.PI * freq * t) * 0.35 +
        Math.sin(2 * Math.PI * freq * 1.5 * t) * 0.15 * Math.sin(t * 2)
      data.writeInt16LE(Math.round(v * env * 32767 * 0.5), i * 2)
    }
    const header = Buffer.alloc(44)
    header.write('RIFF', 0)
    header.writeUInt32LE(36 + data.length, 4)
    header.write('WAVE', 8)
    header.write('fmt ', 12)
    header.writeUInt32LE(16, 16)
    header.writeUInt16LE(1, 20) // PCM
    header.writeUInt16LE(1, 22) // mono
    header.writeUInt32LE(sampleRate, 24)
    header.writeUInt32LE(sampleRate * 2, 28)
    header.writeUInt16LE(2, 32)
    header.writeUInt16LE(16, 34)
    header.write('data', 36)
    header.writeUInt32LE(data.length, 40)
    fs.writeFileSync(file, Buffer.concat([header, data]))
  }
  wavCache.set(song.id, file)
  return file
}

const pngCache = new Map()
const pngFor = (id) => {
  if (pngCache.has(id)) return pngCache.get(id)
  const hash = crypto.createHash('md5').update(id).digest()
  const r = hash[0],
    g = hash[1],
    b = hash[2]
  // Minimal 1x1 PNG, colour derived from the id.
  const sig = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10])
  const chunk = (type, data) => {
    const len = Buffer.alloc(4)
    len.writeUInt32BE(data.length)
    const td = Buffer.concat([Buffer.from(type), data])
    const crcTable = []
    for (let n = 0; n < 256; n++) {
      let c = n
      for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
      crcTable[n] = c >>> 0
    }
    let crc = 0xffffffff
    for (const byte of td) crc = crcTable[(crc ^ byte) & 0xff] ^ (crc >>> 8)
    const crcBuf = Buffer.alloc(4)
    crcBuf.writeUInt32BE((crc ^ 0xffffffff) >>> 0)
    return Buffer.concat([len, td, crcBuf])
  }
  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(1, 0)
  ihdr.writeUInt32BE(1, 4)
  ihdr[8] = 8
  ihdr[9] = 2
  const raw = Buffer.from([0, r, g, b])
  const png = Buffer.concat([
    sig,
    chunk('IHDR', ihdr),
    chunk('IDAT', zlib.deflateSync(raw)),
    chunk('IEND', Buffer.alloc(0)),
  ])
  pngCache.set(id, png)
  return png
}

/* ------------------------------------------------------------------ */
/* Auth                                                                */
/* ------------------------------------------------------------------ */

const b64url = (obj) => Buffer.from(JSON.stringify(obj)).toString('base64url')
const makeJwt = (username) =>
  `${b64url({ alg: 'HS256', typ: 'JWT' })}.${b64url({
    uid: 'u-demo',
    username,
    role: 'admin',
    exp: Math.floor(Date.now() / 1000) + 7 * 86400,
  })}.mock-signature`

const loginResponse = (username) => ({
  id: 'u-demo',
  isAdmin: true,
  name: 'Demo Listener',
  username,
  token: makeJwt(username),
  subsonicSalt: crypto.randomBytes(8).toString('hex'),
  subsonicToken: crypto.randomBytes(16).toString('hex'),
})

/* ------------------------------------------------------------------ */
/* deluan/rest-ish query handling                                      */
/* ------------------------------------------------------------------ */

const parseListParams = (searchParams) => {
  const p = {
    sort: searchParams.get('_sort') || 'id',
    order: (searchParams.get('_order') || 'ASC').toUpperCase(),
    start: parseInt(searchParams.get('_start') || '0', 10),
    end:
      searchParams.get('_end') != null
        ? parseInt(searchParams.get('_end'), 10)
        : null,
    filters: {},
  }
  for (const key of searchParams.keys()) {
    if (key.startsWith('_')) continue
    p.filters[key] = searchParams.getAll(key)
  }
  return p
}

const cmpValues = (a, b, desc) => {
  const aNull = a === null || a === undefined || a === ''
  const bNull = b === null || b === undefined || b === ''
  // SQLite semantics: NULLs sort first on ASC, last on DESC.
  if (aNull && bNull) return 0
  if (aNull) return desc ? 1 : -1
  if (bNull) return desc ? -1 : 1
  if (a < b) return -1
  if (a > b) return 1
  return 0
}

const applyListParams = (rows, params) => {
  let out = [...rows]
  for (const [key, values] of Object.entries(params.filters)) {
    if (!values.length) continue
    out = out.filter((row) => {
      // genre_id matches any of the song's genre ids (mirrors Navidrome's
      // persistence.genreFilter), e.g. genre_id=g-Rock
      if (key === 'genre_id') {
        const ids = (row.genres || []).map((g) => g.id)
        return values.some((expected) => ids.includes(expected))
      }
      const v = row[key]
      return values.some((expected) => {
        if (expected === 'true') return v === true || v === 'true'
        if (expected === 'false')
          return v === false || v === 'false' || v == null
        if (Array.isArray(v))
          return v.some((x) => String(x.id || x) === expected)
        return String(v) === expected
      })
    })
  }
  const desc = params.order === 'DESC'
  if (params.sort === 'random') {
    for (let i = out.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[out[i], out[j]] = [out[j], out[i]]
    }
  } else {
    out.sort(
      (a, b) =>
        cmpValues(a[params.sort], b[params.sort], desc) * (desc ? -1 : 1),
    )
  }
  const total = out.length
  const start = params.start
  const end = params.end == null ? out.length : Math.max(params.end, start)
  return {
    rows: out.slice(start, end),
    total,
    start,
    end: Math.min(end, total),
  }
}

/* ------------------------------------------------------------------ */
/* Server                                                              */
/* ------------------------------------------------------------------ */

const json = (res, status, body, headers = {}) => {
  const data = JSON.stringify(body)
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(data),
    'Access-Control-Expose-Headers': 'X-Total-Count, Content-Range',
    ...headers,
  })
  res.end(data)
}

const sseClients = new Set()

const server = http.createServer((req, res) => {
  const u = new URL(req.url, `http://${req.headers.host}`)
  const p = u.pathname

  /* ----- auth ----- */
  if (p === '/auth/login' || p === '/auth/createAdmin') {
    let body = ''
    req.on('data', (d) => (body += d))
    req.on('end', () => {
      const { username } = JSON.parse(body || '{}')
      json(res, 200, loginResponse(username || 'demo'))
    })
    return
  }
  if (p === '/auth/avatar') {
    res.writeHead(200, { 'Content-Type': 'image/png' })
    return res.end(pngFor('avatar'))
  }

  /* ----- SSE events ----- */
  if (p === '/api/events') {
    res.writeHead(200, {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    })
    res.write(
      `event: scanStatus\ndata: ${JSON.stringify({ scanning: false, folderCount: 1, count: songs.length })}\n\n`,
    )
    sseClients.add(res)
    req.on('close', () => sseClients.delete(res))
    return
  }

  /* ----- native API lists ----- */
  if (p.startsWith('/api/')) {
    const params = parseListParams(u.searchParams)
    const list = (rows) => {
      const { rows: out, total } = applyListParams(rows, params)
      json(res, 200, out, {
        'X-Total-Count': String(total),
        'Content-Range': `${p.split('/')[2]} ${params.start}-${total ? total - 1 : 0}/${total}`,
      })
    }

    if (p === '/api/song') return list(songs)
    if (p === '/api/album') return list(albums)
    if (p === '/api/artist') return list(artists)
    if (p === '/api/genre') return list(genreRecords)
    if (p === '/api/playlist') return list([])
    if (p === '/api/share') return list([])
    if (p === '/api/radio') return list([])
    if (p === '/api/user')
      return list([
        {
          id: 'u-demo',
          userName: 'demo',
          name: 'Demo Listener',
          isAdmin: true,
          email: '',
        },
      ])
    if (p === '/api/library')
      return list([
        { id: 1, name: 'Demo Music', path: '/music', songCount: songs.length },
      ])
    if (p === '/api/insights') return list([])
    if (p === '/api/transcoding') return list([])
    if (p === '/api/tag') return list([])
    if (p === '/api/scrobble') return list([])

    // getOne-style routes
    let m
    if ((m = p.match(/^\/api\/song\/([^/]+)$/))) {
      const song = songs.find((s) => s.id === m[1])
      return song
        ? json(res, 200, song)
        : json(res, 404, { error: 'not found' })
    }
    if ((m = p.match(/^\/api\/album\/([^/]+)$/))) {
      const album = albums.find((a) => a.id === m[1])
      return album
        ? json(res, 200, album)
        : json(res, 404, { error: 'not found' })
    }
    if ((m = p.match(/^\/api\/artist\/([^/]+)$/))) {
      const artist = artists.find((a) => a.id === m[1] || a.name === m[1])
      return artist
        ? json(res, 200, artist)
        : json(res, 404, { error: 'not found' })
    }
    if (/^\/api\/keepalive\//.test(p)) return json(res, 200, { id: 'ok' })
    if (p === '/api/config/config')
      return json(res, 200, {
        version: '0.0.0-mock',
        metricsEnabled: false,
        lastScan: new Date().toISOString(),
        count: songs.length,
      })
    if (p === '/api/translation/en')
      return json(res, 200, { id: 'en', data: '{}' })

    return json(res, 404, { error: `mock harness: no route for ${p}` })
  }

  /* ----- Subsonic ----- */
  if (p.startsWith('/rest/')) {
    const command = p.slice('/rest/'.length).replace('.view', '')
    const ok = (extra = {}) =>
      json(res, 200, {
        'subsonic-response': {
          status: 'ok',
          version: '1.16.1',
          type: 'navidrome-mock',
          ...extra,
        },
      })

    if (command === 'ping') return ok()
    if (command === 'getScanStatus')
      return ok({
        scanStatus: { scanning: false, count: songs.length, folderCount: 1 },
      })
    if (command === 'getNowPlaying') return ok({ nowPlaying: { entry: [] } })
    if (command === 'star' || command === 'unstar') {
      const id = u.searchParams.get('id')
      const song = songs.find((s) => s.id === id)
      if (song) {
        song.starred = command === 'star'
        song.starredAt = song.starred ? new Date().toISOString() : null
        const album = albums.find((a) => a.id === song.albumId)
        if (album) album.starred = song.starred
      }
      return ok()
    }
    if (command === 'setRating') {
      const id = u.searchParams.get('id')
      const song = songs.find((s) => s.id === id)
      if (song) song.rating = parseInt(u.searchParams.get('rating') || '0', 10)
      return ok()
    }
    if (command === 'reportPlayback') {
      const id = u.searchParams.get('mediaId')
      const state = u.searchParams.get('state')
      const song = songs.find((s) => s.id === id)
      if (song && state === 'stopped') {
        song.playCount += 1
        song.playDate = new Date().toISOString()
        const album = albums.find((a) => a.id === song.albumId)
        if (album) album.playCount += 1
      }
      return ok()
    }
    if (command === 'getTranscodeDecision') {
      // No transcoding in the harness: the client falls back to raw streaming.
      return json(res, 200, {
        'subsonic-response': {
          status: 'failed',
          version: '1.16.1',
          error: {
            code: 0,
            message: 'transcoding not available in dev harness',
          },
        },
      })
    }
    if (command === 'stream') {
      const song = songs.find((s) => s.id === u.searchParams.get('id'))
      if (!song) {
        res.writeHead(404)
        return res.end()
      }
      const file = wavFor(song)
      const stat = fs.statSync(file)
      res.writeHead(200, {
        'Content-Type': 'audio/wav',
        'Content-Length': stat.size,
        'Accept-Ranges': 'bytes',
      })
      return fs.createReadStream(file).pipe(res)
    }
    if (command === 'getCoverArt') {
      const png = pngFor(u.searchParams.get('id') || 'cover')
      res.writeHead(200, {
        'Content-Type': 'image/png',
        'Content-Length': png.length,
        'Cache-Control': 'max-age=600',
      })
      return res.end(png)
    }
    if (command === 'getArtistInfo' || command === 'getAlbumInfo')
      return ok(
        command === 'getArtistInfo'
          ? { artistInfo: { similarArtist: [] } }
          : { albumInfo: {} },
      )
    if (command === 'getTopSongs') return ok({ topSongs: { song: [] } })
    if (command === 'getSimilarSongs2')
      return ok({ similarSongs2: { song: [] } })

    return ok()
  }

  if (p === '/backgrounds/' || p.startsWith('/backgrounds')) {
    const png = pngFor('bg')
    res.writeHead(200, { 'Content-Type': 'image/png' })
    return res.end(png)
  }

  json(res, 404, { error: 'not found' })
})

const keepAliveTimer = setInterval(() => {
  for (const client of sseClients) {
    try {
      client.write(
        `event: keepAlive\ndata: ${JSON.stringify({ ts: Date.now() })}\n\n`,
      )
    } catch {
      sseClients.delete(client)
    }
  }
}, 5000)
server.on('close', () => clearInterval(keepAliveTimer))

server.listen(PORT, () => {
  // eslint-disable-next-line no-console
  console.log(
    `[dev-navidrome-mock] listening on :${PORT} (${songs.length} songs, ${albums.length} albums)`,
  )
})
