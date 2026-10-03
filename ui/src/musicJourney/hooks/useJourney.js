/**
 * useJourney — orchestration between the engine, the song pool, Redux and
 * the existing audio player queue.
 *
 * The hook never creates a second player: starting a journey simply fills
 * the existing queue via the standard playTracks action, and live
 * adjustments rebuild the queue tail with PLAYER_REBUILD_QUEUE, which keeps
 * the currently playing item (same uuid) so playback is not interrupted.
 */
import { useCallback, useMemo, useRef } from 'react'
import { useDispatch, useSelector, useStore } from 'react-redux'
import { useDataProvider } from 'react-admin'
import { playTracks } from '../../actions'
import { rebuildQueueAfter } from '../../actions'
import { clearQueue } from '../../actions'
import {
  buildJourney,
  JourneyError,
  regenerateRemaining,
} from '../engine/journeyEngine'
import { fetchSongPool } from './fetchSongPool'
import {
  journeyEnded,
  journeyGenerateFailure,
  journeyGenerateStart,
  journeyGenerateSuccess,
  journeyRegenerated,
  setJourneyPreferences,
} from '../state/journeyActions'

const clamp01 = (v) => Math.min(1, Math.max(0, v))

/** Preference deltas applied by the live controls ("More energetic"...). */
export const JOURNEY_ADJUSTMENTS = {
  'more-energy': (p) => ({ ...p, intensity: clamp01(p.intensity + 0.2) }),
  'more-relaxed': (p) => ({ ...p, intensity: clamp01(p.intensity - 0.2) }),
  'more-familiar': (p) => ({ ...p, discovery: clamp01(p.discovery - 0.3) }),
  surprise: (p) => ({ ...p, discovery: clamp01(p.discovery + 0.35) }),
}

/** How many already-played journey tracks to keep during a live rebuild. */
export const computeKeptCount = (queue, currentUuid, journey) => {
  if (!currentUuid) return 0
  const idx = queue.findIndex((item) => item.uuid === currentUuid)
  if (idx < 0) return 0
  const journeyIds = new Set(journey.tracks.map((t) => t.track.id))
  let count = 0
  for (let i = 0; i <= idx; i++) {
    if (journeyIds.has(queue[i]?.trackId)) count++
  }
  return Math.min(count, journey.tracks.length)
}

const songsToPlayData = (tracks, rawSongs) => {
  const data = {}
  const ids = []
  for (const t of tracks) {
    const raw = rawSongs[t.track.id]
    if (!raw) continue
    data[t.track.id] = raw
    ids.push(t.track.id)
  }
  return { data, ids }
}

export const useJourney = () => {
  const dispatch = useDispatch()
  const store = useStore()
  const dataProvider = useDataProvider()
  const journeyState = useSelector((state) => state.journey)
  const playerState = useSelector((state) => state.player)

  // Monotonic guard against races between concurrent generations.
  const seqRef = useRef(0)
  const busyRef = useRef(false)

  const generate = useCallback(
    async (preferences) => {
      if (busyRef.current) return null
      busyRef.current = true
      const seq = ++seqRef.current
      dispatch(journeyGenerateStart(preferences))
      try {
        const pool = await fetchSongPool(dataProvider, {
          genreId: preferences.genreId,
        })
        if (seq !== seqRef.current) return null

        const journey = buildJourney({
          songs: pool.list,
          prefs: preferences,
          skipEvents: store.getState().journey?.feedback.events || [],
        })
        const { data, ids } = songsToPlayData(journey.tracks, pool.songs)
        if (!ids.length) {
          throw new JourneyError('NOT_ENOUGH_TRACKS', { available: 0 })
        }
        // Fill the queue FIRST: these dispatches happen outside a React event
        // handler (after awaits), so each one renders separately. If the
        // journey went active before its tracks were queued, the watcher
        // would briefly see an active journey with an empty queue and end it.
        dispatch(playTracks(data, ids))
        dispatch(journeyGenerateSuccess(journey))
        return journey
      } catch (error) {
        if (seq === seqRef.current) {
          dispatch(
            journeyGenerateFailure(
              error instanceof JourneyError
                ? { code: error.code, details: error.details }
                : { code: 'UNKNOWN', details: {} },
            ),
          )
        }
        return null
      } finally {
        busyRef.current = false
      }
    },
    [dataProvider, dispatch, store],
  )

  /**
   * Live adjustment: rebuilds only the future part of the active journey.
   * `kind` is one of JOURNEY_ADJUSTMENTS keys or {type:'less-genre'}.
   */
  const adjust = useCallback(
    async (kind) => {
      const active = journeyState.active
      if (!active || busyRef.current) return null
      busyRef.current = true
      const seq = ++seqRef.current
      try {
        const prefs = { ...active.journey.preferences }
        let nextPrefs
        if (kind === 'less-genre') {
          // Drop the genre that dominates the remaining (unplayed) part.
          const kept = computeKeptCount(
            playerState.queue,
            playerState.current?.uuid,
            active.journey,
          )
          const counts = new Map()
          active.journey.tracks.slice(kept).forEach((t) => {
            const g = String(t.track.genre || '')
              .trim()
              .toLowerCase()
            if (g) counts.set(g, (counts.get(g) || 0) + 1)
          })
          const dominant = [...counts.entries()].sort((a, b) => b[1] - a[1])[0]
          nextPrefs = dominant
            ? {
                ...prefs,
                excludeGenres: [...(prefs.excludeGenres || []), dominant[0]],
              }
            : prefs
        } else {
          nextPrefs = JOURNEY_ADJUSTMENTS[kind](prefs)
        }

        const pool = await fetchSongPool(dataProvider, {
          genreId: prefs.genreId,
        })
        if (seq !== seqRef.current) return null

        // Re-read the store: playback may have advanced while the pool was
        // fetched, and the split point must reflect the *current* position —
        // otherwise a track playing right now could be rebuilt into the tail.
        const latest = store.getState()
        const latestActive = latest.journey?.active
        const latestPlayer = latest.player
        if (!latestActive || latestActive.journey.id !== active.journey.id) {
          return null
        }

        const keptCount = computeKeptCount(
          latestPlayer.queue,
          latestPlayer.current?.uuid,
          latestActive.journey,
        )
        const rebuilt = regenerateRemaining({
          journey: latestActive.journey,
          playedCount: Math.max(keptCount, 1),
          prefs: nextPrefs,
          songs: pool.list,
          skipEvents: latest.journey.feedback.events,
        })

        const fresh = rebuilt.tracks.slice(rebuilt.keptCount)
        const { data, ids } = songsToPlayData(fresh, pool.songs)

        const anchorUuid = latestPlayer.current?.uuid
        if (
          anchorUuid &&
          latestPlayer.queue.some((i) => i.uuid === anchorUuid)
        ) {
          dispatch(journeyRegenerated(rebuilt))
          dispatch(rebuildQueueAfter(anchorUuid, data, ids))
        } else {
          // Nothing is anchored (player idle): restart from the kept head.
          const all = songsToPlayData(rebuilt.tracks, pool.songs)
          dispatch(playTracks(all.data, all.ids))
          dispatch(journeyRegenerated(rebuilt))
        }
        dispatch(setJourneyPreferences(rebuilt.preferences))
        return rebuilt
      } catch (error) {
        if (seq === seqRef.current) {
          dispatch(
            journeyGenerateFailure(
              error instanceof JourneyError
                ? { code: error.code, details: error.details }
                : { code: 'UNKNOWN', details: {} },
            ),
          )
        }
        return null
      } finally {
        busyRef.current = false
      }
    },
    [dataProvider, dispatch, journeyState, playerState, store],
  )

  const stop = useCallback(() => {
    seqRef.current++
    dispatch(clearQueue())
    dispatch(journeyEnded('stopped'))
  }, [dispatch])

  /** Replay a journey from history: refetch its tracks by id and play them. */
  const replay = useCallback(
    async (entry) => {
      if (!entry?.trackIds?.length) return false
      try {
        const res = await dataProvider.getList('song', {
          pagination: { page: 1, perPage: entry.trackIds.length },
          sort: { field: 'id', order: 'ASC' },
          filter: { id: entry.trackIds, missing: false },
        })
        const byId = {}
        ;(res.data || []).forEach((song) => {
          byId[song.id] = song
        })
        // Preserve the original journey order.
        const ids = entry.trackIds.filter((id) => byId[id])
        if (!ids.length) return false
        const data = {}
        ids.forEach((id) => (data[id] = byId[id]))
        dispatch(playTracks(data, ids))
        return true
      } catch {
        return false
      }
    },
    [dataProvider, dispatch],
  )

  return useMemo(
    () => ({ generate, adjust, stop, replay }),
    [generate, adjust, stop, replay],
  )
}
