import { renderHook, act } from '@testing-library/react-hooks'
import { useReplayGain } from './useReplayGain'
import { describe, it, beforeEach, afterEach, vi, expect } from 'vitest'

// Mock calculateGain utility
vi.mock('../../utils/calculateReplayGain', () => ({
  calculateGain: vi.fn(),
}))

// Import the mocked module
import * as calculateReplayGain from '../../utils/calculateReplayGain'

describe('useReplayGain', () => {
  const mockCalculateGain = calculateReplayGain.calculateGain

  beforeEach(() => {
    vi.clearAllMocks()
    // Mock Web Audio API
    global.AudioContext = vi.fn().mockImplementation(function () {
      this.createMediaElementSource = vi.fn(() => ({
        connect: vi.fn(),
      }))
      this.createGain = vi.fn(() => ({
        gain: {
          setValueAtTime: vi.fn(),
        },
        connect: vi.fn(),
      }))
      this.currentTime = 0
    })
  })

  afterEach(() => {
    delete global.AudioContext
  })

  it('should initialize with null context and gainNode', () => {
    const { result } = renderHook(() =>
      useReplayGain(null, { current: {} }, { gainMode: 'track' }),
    )

    expect(result.current.context).toBeNull()
    expect(result.current.gainNode).toBeNull()
  })

  it('should create audio context when conditions are met', () => {
    const mockAudioInstance = { crossOrigin: '' }
    const mockPlayerState = {
      current: { song: { title: 'Test Song' } },
    }
    const mockGainInfo = { gainMode: 'track' }

    const { result } = renderHook(() =>
      useReplayGain(mockAudioInstance, mockPlayerState, mockGainInfo),
    )

    expect(global.AudioContext).toHaveBeenCalled()
    expect(result.current.context).toBeInstanceOf(AudioContext)
  })

  it('should apply gain when gainNode exists', () => {
    const mockAudioInstance = { crossOrigin: '' }
    const mockPlayerState = {
      current: { song: { title: 'Test Song' } },
    }
    const mockGainInfo = { gainMode: 'track' }

    mockCalculateGain.mockReturnValue(0.8)

    const { result } = renderHook(() =>
      useReplayGain(mockAudioInstance, mockPlayerState, mockGainInfo),
    )

    expect(mockCalculateGain).toHaveBeenCalledWith(
      mockGainInfo,
      mockPlayerState.current.song,
    )
    expect(result.current.gainNode.gain.setValueAtTime).toHaveBeenCalledWith(
      0.8,
      0,
    )
  })

  it('should handle Web Audio API errors gracefully', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    // Mock AudioContext to throw error
    global.AudioContext = vi.fn().mockImplementation(function () {
      throw new Error('Web Audio API not supported')
    })

    const mockAudioInstance = {}
    const mockPlayerState = { current: {} }
    const mockGainInfo = { gainMode: 'track' }

    const { result } = renderHook(() =>
      useReplayGain(mockAudioInstance, mockPlayerState, mockGainInfo),
    )

    expect(consoleSpy).toHaveBeenCalledWith(
      'Error initializing Web Audio API for replay gain:',
      expect.any(Error),
    )
    expect(result.current.context).toBeNull()

    consoleSpy.mockRestore()
  })

  it('should handle gain application errors gracefully', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const mockAudioInstance = { crossOrigin: '' }
    const mockPlayerState = {
      current: { song: { title: 'Test Song' } },
    }
    const mockGainInfo = { gainMode: 'track' }

    // Mock gain.setValueAtTime to throw error
    const mockGainNode = {
      gain: {
        setValueAtTime: vi.fn(() => {
          throw new Error('Gain application failed')
        }),
      },
      connect: vi.fn(),
    }

    global.AudioContext = vi.fn().mockImplementation(function () {
      this.createMediaElementSource = vi.fn(() => ({
        connect: vi.fn(),
      }))
      this.createGain = vi.fn(() => mockGainNode)
      this.currentTime = 0
    })

    const { result } = renderHook(() =>
      useReplayGain(mockAudioInstance, mockPlayerState, mockGainInfo),
    )

    expect(consoleSpy).toHaveBeenCalledWith(
      'Error applying replay gain:',
      expect.any(Error),
    )

    consoleSpy.mockRestore()
  })

  it('should not initialize when gainMode is not album or track', () => {
    const mockAudioInstance = {}
    const mockPlayerState = { current: {} }
    const mockGainInfo = { gainMode: 'off' }

    const { result } = renderHook(() =>
      useReplayGain(mockAudioInstance, mockPlayerState, mockGainInfo),
    )

    expect(global.AudioContext).not.toHaveBeenCalled()
    expect(result.current.context).toBeNull()
  })
})
