/**
 * useJourneyWatcher — keeps the journey state in sync with the real player.
 *
 * Mounted once near the app root. It translates raw player events into
 * journey semantics:
 *
 *  - PLAYER_TRACK_ABANDONED while a journey track was playing:
 *      played < 60% (and > 3s)  -> skip feedback for the engine
 *      played >= 90%            -> treated as a completed track
 *  - track advanced without an abandon event -> natural completion
 *  - queue no longer contains the journey      -> journey interrupted
 *  - last journey track ended                  -> journey completed
 *
 * The listener is intentionally defensive: any of these computations
 * failing must never break normal playback.
 */
import { useEffect, useMemo, useRef } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  journeyEnded,
  journeySkipped,
  journeyTrackCompleted,
} from '../state/journeyActions'

const SKIP_RATIO = 0.6
const COMPLETED_RATIO = 0.9
const MIN_SKIP_MS = 3000

export const useJourneyWatcher = () => {
  const dispatch = useDispatch()
  const active = useSelector((state) => state.journey?.active)
  const lastAbandoned = useSelector((state) => state.journey?.lastAbandoned)
  const queue = useSelector((state) => state.player?.queue)
  const current = useSelector((state) => state.player?.current)

  const prevCurrentRef = useRef(null)
  const processedAbandonRef = useRef(null)

  const journeyTracks = active?.journey?.tracks
  const journeyIds = useMemo(
    () =>
      journeyTracks ? new Set(journeyTracks.map((t) => t.track.id)) : null,
    [journeyTracks],
  )

  // 1) Skip feedback / late-stage abandon.
  useEffect(() => {
    if (!active || !lastAbandoned || !journeyIds?.has(lastAbandoned.trackId)) {
      return
    }
    if (processedAbandonRef.current === lastAbandoned) return
    processedAbandonRef.current = lastAbandoned

    const jt = journeyTracks.find((t) => t.track.id === lastAbandoned.trackId)
    if (!jt) return
    const durationMs = jt.track.duration * 1000
    const ratio = durationMs > 0 ? lastAbandoned.positionMs / durationMs : 0
    if (ratio >= COMPLETED_RATIO) {
      dispatch(journeyTrackCompleted())
    } else if (ratio < SKIP_RATIO && lastAbandoned.positionMs >= MIN_SKIP_MS) {
      dispatch(journeySkipped(jt.track))
    }
  }, [active, journeyIds, journeyTracks, lastAbandoned, dispatch])

  // 2) Natural progression + end detection.
  useEffect(() => {
    const prev = prevCurrentRef.current
    prevCurrentRef.current = current

    if (!active || !prev?.trackId || prev.uuid === current?.uuid) return

    const prevWasJourney = journeyIds?.has(prev.trackId)

    // Queue lost the journey entirely (cleared or replaced).
    const journeyStillQueued = queue?.some((item) =>
      journeyIds?.has(item.trackId),
    )
    if (!journeyStillQueued) {
      // If the previous track was the journey's last one, it completed.
      const lastTrack = journeyTracks[journeyTracks.length - 1]
      const completed = prevWasJourney && lastTrack?.track.id === prev.trackId
      dispatch(journeyEnded(completed ? 'completed' : 'interrupted'))
      return
    }

    if (!prevWasJourney) return

    const abandonedAlready =
      processedAbandonRef.current?.trackId === prev.trackId
    if (!abandonedAlready) {
      dispatch(journeyTrackCompleted())
    }

    // Did we just pass the final journey track?
    let lastJourneyIdx = -1
    queue.forEach((item, i) => {
      if (journeyIds.has(item.trackId)) lastJourneyIdx = i
    })
    const currentIdx = queue.findIndex((item) => item.uuid === current?.uuid)
    const pastEnd =
      current?.ended || (currentIdx !== -1 && currentIdx > lastJourneyIdx)
    const prevWasLast =
      journeyTracks[journeyTracks.length - 1]?.track.id === prev.trackId
    if (pastEnd && prevWasLast) {
      dispatch(journeyEnded('completed'))
    } else if (currentIdx !== -1 && currentIdx > lastJourneyIdx) {
      dispatch(journeyEnded('interrupted'))
    }
  }, [active, current, queue, journeyIds, journeyTracks, dispatch])
}

export default useJourneyWatcher
