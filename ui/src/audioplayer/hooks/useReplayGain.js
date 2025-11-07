import { useEffect, useState } from 'react'
import { calculateGain } from '../../utils/calculateReplayGain'

/**
 * Custom hook for managing replay gain functionality using Web Audio API.
 * Adjusts audio gain based on track or album replay gain metadata.
 *
 * @param {Object} audioInstance - The HTML audio element instance.
 * @param {Object} playerState - The current player state from Redux store.
 * @param {Object} gainInfo - Replay gain configuration from Redux store.
 * @returns {Object} Replay gain-related state.
 * @returns {AudioContext|null} context - Web Audio API context.
 * @returns {GainNode|null} gainNode - Gain node for audio manipulation.
 *
 * @example
 * const { context, gainNode } = useReplayGain(audioInstance, playerState, gainInfo);
 */
export const useReplayGain = (audioInstance, playerState, gainInfo) => {
  const [context, setContext] = useState(null)
  const [gainNode, setGainNode] = useState(null)

  useEffect(() => {
    if (
      context === null &&
      audioInstance &&
      'AudioContext' in window &&
      (gainInfo.gainMode === 'album' || gainInfo.gainMode === 'track')
    ) {
      try {
        const ctx = new AudioContext()
        // Support radios in Firefox
        if (audioInstance) {
          audioInstance.crossOrigin = 'anonymous'
        }
        const source = ctx.createMediaElementSource(audioInstance)
        const gain = ctx.createGain()

        source.connect(gain)
        gain.connect(ctx.destination)

        setContext(ctx)
        setGainNode(gain)
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error(
          'Error initializing Web Audio API for replay gain:',
          error,
        )
        // Fallback: continue without replay gain
      }
    }
  }, [audioInstance, context, gainInfo.gainMode])

  useEffect(() => {
    if (gainNode && context) {
      try {
        const current = playerState.current || {}
        const song = current.song || {}

        const numericGain = calculateGain(gainInfo, song)
        gainNode.gain.setValueAtTime(numericGain, context.currentTime)
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Error applying replay gain:', error)
        // Continue playback without gain adjustment
      }
    }
  }, [audioInstance, context, gainNode, playerState, gainInfo])

  return {
    context,
    gainNode,
  }
}
