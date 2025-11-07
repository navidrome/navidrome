import { useSelector, useDispatch } from 'react-redux'
import {
  clearQueue,
  currentPlaying,
  setPlayMode,
  setVolume,
  syncQueue,
} from '../../actions'

/**
 * Custom hook for managing player state and actions via Redux.
 * Centralizes access to player-related state and dispatch functions.
 *
 * @returns {Object} Player state and action dispatchers.
 * @returns {Object} playerState - Current player state from Redux store.
 * @returns {Function} dispatch - Redux dispatch function.
 * @returns {Function} dispatchCurrentPlaying - Dispatches current playing action.
 * @returns {Function} dispatchSetPlayMode - Dispatches set play mode action.
 * @returns {Function} dispatchSetVolume - Dispatches set volume action.
 * @returns {Function} dispatchSyncQueue - Dispatches sync queue action.
 * @returns {Function} dispatchClearQueue - Dispatches clear queue action.
 *
 * @example
 * const { playerState, dispatchCurrentPlaying } = usePlayerState();
 */
export const usePlayerState = () => {
  const playerState = useSelector((state) => state.player)
  const dispatch = useDispatch()

  /**
   * Dispatches the current playing action.
   *
   * @param {Object} info - Audio information.
   */
  const dispatchCurrentPlaying = (info) => {
    dispatch(currentPlaying(info))
  }

  /**
   * Dispatches the set play mode action.
   *
   * @param {string} mode - Play mode (e.g., 'single', 'loop', 'shuffle').
   */
  const dispatchSetPlayMode = (mode) => {
    dispatch(setPlayMode(mode))
  }

  /**
   * Dispatches the set volume action with square root compensation.
   *
   * @param {number} volume - Volume level (0-1).
   */
  const dispatchSetVolume = (volume) => {
    // sqrt to compensate for the logarithmic volume
    dispatch(setVolume(Math.sqrt(volume)))
  }

  /**
   * Dispatches the sync queue action.
   *
   * @param {Object} audioInfo - Audio information.
   * @param {Array} audioLists - List of audio tracks.
   */
  const dispatchSyncQueue = (audioInfo, audioLists) => {
    dispatch(syncQueue(audioInfo, audioLists))
  }

  /**
   * Dispatches the clear queue action.
   */
  const dispatchClearQueue = () => {
    dispatch(clearQueue())
  }

  return {
    playerState,
    dispatch,
    dispatchCurrentPlaying,
    dispatchSetPlayMode,
    dispatchSetVolume,
    dispatchSyncQueue,
    dispatchClearQueue,
  }
}
