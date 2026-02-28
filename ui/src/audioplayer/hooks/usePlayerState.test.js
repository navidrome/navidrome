/* eslint-env node */

import { renderHook } from '@testing-library/react-hooks'
import { usePlayerState } from './usePlayerState'
import { useDispatch, useSelector } from 'react-redux'
import { describe, it, beforeEach, vi, expect } from 'vitest'

// Mock react-redux
vi.mock('react-redux', () => ({
  useDispatch: vi.fn(),
  useSelector: vi.fn(),
}))

// Mock actions
vi.mock('../../actions', () => ({
  clearQueue: vi.fn(() => ({ type: 'CLEAR_QUEUE' })),
  currentPlaying: vi.fn(() => ({ type: 'CURRENT_PLAYING' })),
  setPlayMode: vi.fn(() => ({ type: 'SET_PLAY_MODE' })),
  setVolume: vi.fn(() => ({ type: 'SET_VOLUME' })),
  syncQueue: vi.fn(() => ({ type: 'SYNC_QUEUE' })),
}))

// Import the mocked actions
import * as actions from '../../actions'

describe('usePlayerState', () => {
  const mockPlayerState = {
    queue: [],
    current: null,
    mode: 'single',
    volume: 0.8,
  }

  const mockDispatch = vi.fn()

  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    useDispatch.mockReturnValue(mockDispatch)
    useSelector.mockReturnValue(mockPlayerState)
  })

  it('should return player state and dispatch functions', () => {
    const { result } = renderHook(() => usePlayerState())

    expect(result.current.playerState).toEqual(mockPlayerState)
    expect(typeof result.current.dispatch).toBe('function')
    expect(typeof result.current.dispatchCurrentPlaying).toBe('function')
    expect(typeof result.current.dispatchSetPlayMode).toBe('function')
    expect(typeof result.current.dispatchSetVolume).toBe('function')
    expect(typeof result.current.dispatchSyncQueue).toBe('function')
    expect(typeof result.current.dispatchClearQueue).toBe('function')
  })

  it('should dispatch current playing action', () => {
    const { result } = renderHook(() => usePlayerState())
    const mockInfo = { trackId: 'track1' }

    result.current.dispatchCurrentPlaying(mockInfo)

    expect(mockDispatch).toHaveBeenCalledWith({ type: 'CURRENT_PLAYING' })
  })

  it('should dispatch set play mode action', () => {
    const { result } = renderHook(() => usePlayerState())

    result.current.dispatchSetPlayMode('loop')

    expect(mockDispatch).toHaveBeenCalledWith({ type: 'SET_PLAY_MODE' })
  })

  it('should dispatch set volume action with square root compensation', () => {
    const { result } = renderHook(() => usePlayerState())

    result.current.dispatchSetVolume(0.5)

    expect(mockDispatch).toHaveBeenCalledWith({ type: 'SET_VOLUME' })
    // Verify square root calculation
    expect(actions.setVolume).toHaveBeenCalledWith(Math.sqrt(0.5))
  })

  it('should dispatch sync queue action', () => {
    const { result } = renderHook(() => usePlayerState())
    const mockAudioInfo = { trackId: 'track1' }
    const mockAudioLists = [{ id: '1' }]

    result.current.dispatchSyncQueue(mockAudioInfo, mockAudioLists)

    expect(mockDispatch).toHaveBeenCalledWith({ type: 'SYNC_QUEUE' })
  })

  it('should dispatch clear queue action', () => {
    const { result } = renderHook(() => usePlayerState())

    result.current.dispatchClearQueue()

    expect(mockDispatch).toHaveBeenCalledWith({ type: 'CLEAR_QUEUE' })
  })

  it('should use correct Redux hooks', () => {
    renderHook(() => usePlayerState())

    expect(useDispatch).toHaveBeenCalled()
    expect(useSelector).toHaveBeenCalledWith(expect.any(Function))
  })
})
