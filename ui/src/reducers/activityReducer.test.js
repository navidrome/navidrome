import { activityReducer } from './activityReducer'
import {
  EVENT_SCAN_STATUS,
  EVENT_SERVER_START,
  EVENT_NOW_PLAYING_COUNT,
  EVENT_STREAM_RECONNECTED,
} from '../actions'
import config from '../config'

describe('activityReducer', () => {
  const initialState = {
    scanStatus: {
      scanning: false,
      folderCount: 0,
      count: 0,
      error: '',
      elapsedTime: 0,
    },
    serverStart: { version: config.version },
    nowPlayingCount: 0,
    streamReconnected: 0,
  }

  it('returns the initial state when no action is specified', () => {
    expect(activityReducer(undefined, {})).toEqual(initialState)
  })

  it('handles EVENT_SCAN_STATUS action with elapsedTime field', () => {
    const elapsedTime = 123456789 // nanoseconds
    const action = {
      type: EVENT_SCAN_STATUS,
      data: {
        scanning: true,
        folderCount: 5,
        count: 100,
        error: '',
        elapsedTime: elapsedTime,
      },
    }

    const newState = activityReducer(initialState, action)
    expect(newState.scanStatus).toEqual({
      scanning: true,
      folderCount: 5,
      count: 100,
      error: '',
      elapsedTime: elapsedTime,
    })
  })

  it('handles EVENT_SCAN_STATUS action with string elapsedTime', () => {
    const action = {
      type: EVENT_SCAN_STATUS,
      data: {
        scanning: true,
        folderCount: 5,
        count: 100,
        error: '',
        elapsedTime: '123456789',
      },
    }

    const newState = activityReducer(initialState, action)
    expect(newState.scanStatus.elapsedTime).toEqual(123456789)
  })

  it('handles EVENT_SCAN_STATUS with error field', () => {
    const action = {
      type: EVENT_SCAN_STATUS,
      data: {
        scanning: false,
        folderCount: 0,
        count: 0,
        error: 'Test error message',
        elapsedTime: 0,
      },
    }

    const newState = activityReducer(initialState, action)
    expect(newState.scanStatus.error).toEqual('Test error message')
  })

  it('handles EVENT_SERVER_START action', () => {
    const action = {
      type: EVENT_SERVER_START,
      data: {
        version: '1.0.0',
        startTime: '2023-01-01T00:00:00Z',
      },
    }

    const newState = activityReducer(initialState, action)
    expect(newState.serverStart).toEqual({
      version: '1.0.0',
      startTime: Date.parse('2023-01-01T00:00:00Z'),
    })
  })

  it('preserves the scanStatus when handling EVENT_SERVER_START', () => {
    const currentState = {
      scanStatus: {
        scanning: true,
        folderCount: 5,
        count: 100,
        error: 'Previous error',
        elapsedTime: 12345,
      },
      serverStart: { version: config.version },
    }

    const action = {
      type: EVENT_SERVER_START,
      data: {
        version: '1.0.0',
        startTime: '2023-01-01T00:00:00Z',
      },
    }

    const newState = activityReducer(currentState, action)
    expect(newState.scanStatus).toEqual(currentState.scanStatus)
    expect(newState.serverStart).toEqual({
      version: '1.0.0',
      startTime: Date.parse('2023-01-01T00:00:00Z'),
    })
  })

  it('handles EVENT_NOW_PLAYING_COUNT', () => {
    const action = {
      type: EVENT_NOW_PLAYING_COUNT,
      data: { count: 5 },
    }
    const newState = activityReducer(initialState, action)
    expect(newState.nowPlayingCount).toEqual(5)
  })

  it('handles EVENT_STREAM_RECONNECTED', () => {
    const action = {
      type: EVENT_STREAM_RECONNECTED,
      data: {},
    }
    const beforeTimestamp = Date.now()
    const newState = activityReducer(initialState, action)
    const afterTimestamp = Date.now()

    expect(newState.streamReconnected).toBeGreaterThanOrEqual(beforeTimestamp)
    expect(newState.streamReconnected).toBeLessThanOrEqual(afterTimestamp)
  })
})
