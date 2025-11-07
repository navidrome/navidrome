import { useCallback, useEffect, useState } from 'react'
import subsonic from '../../subsonic'

/**
 * Custom hook for managing the audio instance and related effects.
 * Handles audio element setup and mobile volume adjustments.
 *
 * @param {boolean} isMobilePlayer - Whether the player is running on a mobile device.
 * @returns {Object} Audio instance-related state and handlers.
 * @returns {HTMLAudioElement|null} audioInstance - The audio element instance.
 * @returns {Function} setAudioInstance - Setter for the audio instance.
 * @returns {Function} onAudioPlay - Handler for audio play events.
 *
 * @example
 * const { audioInstance, setAudioInstance, onAudioPlay } = useAudioInstance(isMobilePlayer);
 */
export const useAudioInstance = (isMobilePlayer) => {
  const [audioInstance, setAudioInstance] = useState(null)

  /**
   * Handles audio play events, resuming context if needed and updating document title.
   *
   * @param {AudioContext|null} audioContext - Web Audio API context from replay gain hook.
   * @param {Object} info - Audio play information.
   * @param {Object} info.song - Song metadata.
   * @param {number} info.duration - Track duration.
   * @param {boolean} info.isRadio - Whether it's a radio stream.
   * @param {string} info.trackId - Track identifier.
   * @param {number} info.currentTime - Current playback time.
   * @param {Function} dispatchCurrentPlaying - Function to dispatch current playing action.
   * @param {boolean} showNotifications - Whether to show notifications.
   * @param {Function} sendNotification - Function to send notifications.
   * @param {number|null} startTime - Start time for scrobbling.
   * @param {Function} setStartTime - Setter for start time.
   * @param {Function} resetPreloading - Function to reset preloading.
   * @param {Object} config - Application configuration.
   * @param {Object} ReactGA - Google Analytics instance.
   */
  const onAudioPlay = useCallback(
    (
      audioContext,
      info,
      dispatchCurrentPlaying,
      showNotifications,
      sendNotification,
      startTime,
      setStartTime,
      resetPreloading,
      config,
      ReactGA,
    ) => {
      // Resume audio context if suspended
      if (audioContext && audioContext.state !== 'running') {
        try {
          audioContext.resume()
        } catch (error) {
          // eslint-disable-next-line no-console
          console.error('Error resuming audio context:', error)
        }
      }

      dispatchCurrentPlaying(info)

      if (startTime === null) {
        setStartTime(Date.now())
      }

      if (info.duration) {
        const song = info.song
        document.title = `${song.title} - ${song.artist} - Navidrome`

        if (!info.isRadio) {
          const pos = startTime === null ? null : Math.floor(info.currentTime)
          try {
            subsonic.nowPlaying(info.trackId, pos)
          } catch (error) {
            // eslint-disable-next-line no-console
            console.error('Error updating now playing:', error)
          }
        }

        resetPreloading()

        if (config.gaTrackingId) {
          try {
            ReactGA.event({
              category: 'Player',
              action: 'Play song',
              label: `${song.title} - ${song.artist}`,
            })
          } catch (error) {
            // eslint-disable-next-line no-console
            console.error('Google Analytics error:', error)
          }
        }

        if (showNotifications) {
          try {
            sendNotification(
              song.title,
              `${song.artist} - ${song.album}`,
              info.cover,
            )
          } catch (error) {
            // eslint-disable-next-line no-console
            console.error('Notification error:', error)
          }
        }
      }
    },
    [],
  )

  // Mobile volume adjustment effect
  useEffect(() => {
    if (isMobilePlayer && audioInstance) {
      try {
        audioInstance.volume = 1
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Error setting mobile volume:', error)
      }
    }
  }, [isMobilePlayer, audioInstance])

  return {
    audioInstance,
    setAudioInstance,
    onAudioPlay,
  }
}
