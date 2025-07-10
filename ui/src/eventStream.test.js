import { describe, it, beforeEach, vi, expect } from 'vitest'
import { startEventStream } from './eventStream'
import { serverDown } from './actions'
import config from './config'

class MockEventSource {
  constructor(url) {
    this.url = url
    this.readyState = 1
    this.listeners = {}
    this.onerror = null
  }
  addEventListener(type, handler) {
    this.listeners[type] = handler
  }
  close() {
    this.readyState = 2
  }
}

describe('startEventStream', () => {
  vi.useFakeTimers()
  let dispatch
  let instance

  beforeEach(() => {
    dispatch = vi.fn()
    global.EventSource = vi.fn((url) => {
      instance = new MockEventSource(url)
      return instance
    })
    localStorage.setItem('is-authenticated', 'true')
    localStorage.setItem('token', 'abc')
    config.devNewEventStream = true
    // Mock console.log to suppress output during tests
    vi.spyOn(console, 'log').mockImplementation(() => {})
  })

  afterEach(() => {
    config.devNewEventStream = false
  })

  it('reconnects after an error', async () => {
    await startEventStream(dispatch)
    expect(global.EventSource).toHaveBeenCalledTimes(1)
    instance.onerror(new Event('error'))
    expect(dispatch).toHaveBeenCalledWith(serverDown())
    vi.advanceTimersByTime(5000)
    expect(global.EventSource).toHaveBeenCalledTimes(2)
  })
})
