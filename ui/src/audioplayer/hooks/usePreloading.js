import { useCallback, useState } from 'react'

/**
 * Custom hook for managing audio preloading functionality.
 * Preloads the next song in the queue to improve playback continuity.
 *
 * @param {Object} playerState - The current player state from Redux store.
 * @returns {Object} Preloading-related state and handlers.
 * @returns {boolean} preloaded - Whether the next song has been preloaded.
 * @returns {Function} preloadNextSong - Function to preload the next song.
 * @returns {Function} resetPreloading - Function to reset preloading state.
 *
 * @example
 * const { preloaded, preloadNextSong } = usePreloading(playerState);
 */
export const usePreloading = (playerState) => {
  const [preloaded, setPreloaded] = useState(false)

  /**
   * Finds the next song in the queue.
   *
   * @returns {Object|null} The next song object or null if not found.
   */
  const nextSong = useCallback(() => {
    const idx = playerState.queue.findIndex(
      (item) => item.uuid === playerState.current?.uuid,
    )
    return idx !== -1 ? playerState.queue[idx + 1] : null
  }, [playerState])

  /**
   * Preloads the next song by creating an Audio element.
   * This helps reduce buffering delays during playback.
   */
  const preloadNextSong = useCallback(() => {
    if (!preloaded) {
      const next = nextSong()
      if (next != null) {
        try {
          const audio = new Audio()
          audio.src = next.musicSrc
          // Optional: Add load event listeners for better control
          audio.addEventListener('canplaythrough', () => {
            // Preload complete
          })
          audio.addEventListener('error', (error) => {
            // eslint-disable-next-line no-console
            console.error('Preloading error:', error)
          })
          setPreloaded(true)
        } catch (error) {
          // eslint-disable-next-line no-console
          console.error('Error during preloading:', error)
          // Continue without preloading
        }
      }
    }
  }, [preloaded, nextSong])

  /**
   * Resets the preloading state. Useful for track changes or manual resets.
   */
  const resetPreloading = useCallback(() => {
    setPreloaded(false)
  }, [])

  return {
    preloaded,
    preloadNextSong,
    resetPreloading,
  }
}
