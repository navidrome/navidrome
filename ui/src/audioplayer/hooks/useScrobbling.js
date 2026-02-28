import { useCallback, useState } from 'react'
import subsonic from '../../subsonic'

/**
 * Custom hook for managing scrobbling functionality in the audio player.
 * Handles scrobbling state and logic for tracking played songs to external services.
 *
 * @param {Object} playerState - The current player state from Redux store.
 * @param {Function} dispatch - Redux dispatch function.
 * @param {Object} dataProvider - Data provider for API calls.
 * @returns {Object} Scrobbling-related state and handlers.
 * @returns {number|null} startTime - Timestamp when playback started.
 * @returns {boolean} scrobbled - Whether the current track has been scrobbled.
 * @returns {Function} onAudioProgress - Handler for audio progress events.
 * @returns {Function} onAudioPlayTrackChange - Handler for track change events.
 * @returns {Function} onAudioEnded - Handler for audio ended events.
 * @returns {Function} resetScrobbling - Function to reset scrobbling state.
 *
 * @example
 * const { startTime, scrobbled, onAudioProgress, onAudioEnded } = useScrobbling(playerState, dispatch, dataProvider);
 */
export const useScrobbling = (playerState, dispatch, dataProvider) => {
  const [startTime, setStartTime] = useState(null)
  const [scrobbled, setScrobbled] = useState(false)

  /**
   * Handles audio progress events for scrobbling logic.
   * Scrobbles the track if it has been played for more than 50% or 4 minutes.
   *
   * @param {Object} info - Audio progress information.
   * @param {number} info.currentTime - Current playback time.
   * @param {number} info.duration - Total duration of the track.
   * @param {boolean} info.isRadio - Whether the current track is a radio stream.
   * @param {string} info.trackId - Unique identifier of the track.
   */
  const onAudioProgress = useCallback(
    (info) => {
      if (info.ended) {
        document.title = 'Navidrome'
      }

      const progress = (info.currentTime / info.duration) * 100
      if (isNaN(info.duration) || (progress < 50 && info.currentTime < 240)) {
        return
      }

      if (info.isRadio) {
        return
      }

      if (!scrobbled) {
        try {
          if (info.trackId) {
            subsonic.scrobble(info.trackId, startTime)
          }
          setScrobbled(true)
        } catch (error) {
          // eslint-disable-next-line no-console
          console.error('Scrobbling error:', error)
          // Continue without failing the player
        }
      }
    },
    [startTime, scrobbled],
  )

  /**
   * Handles track change events by resetting scrobbling state.
   */
  const onAudioPlayTrackChange = useCallback(() => {
    if (scrobbled) {
      setScrobbled(false)
    }
    if (startTime !== null) {
      setStartTime(null)
    }
  }, [scrobbled, startTime])

  /**
   * Handles audio ended events, resetting state and performing keepalive.
   *
   * @param {string} currentPlayId - ID of the current playing track.
   * @param {Array} audioLists - List of audio tracks.
   * @param {Object} info - Audio information.
   */
  const onAudioEnded = useCallback(
    (currentPlayId, audioLists, info) => {
      setScrobbled(false)
      setStartTime(null)
      try {
        dataProvider
          .getOne('keepalive', { id: info.trackId })
          // eslint-disable-next-line no-console
          .catch((e) => console.log('Keepalive error:', e))
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Keepalive error:', error)
      }
    },
    [dataProvider],
  )

  /**
   * Resets the scrobbling state. Useful for manual resets or testing.
   */
  const resetScrobbling = useCallback(() => {
    setScrobbled(false)
    setStartTime(null)
  }, [])

  return {
    startTime,
    setStartTime,
    scrobbled,
    onAudioProgress,
    onAudioPlayTrackChange,
    onAudioEnded,
    resetScrobbling,
  }
}
