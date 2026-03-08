import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Must mock before importing the module under test
vi.mock('./fetchDecision', () => ({
  fetchTranscodeDecision: vi.fn(),
}))

import { decisionService, detectBrowserProfile } from './index'
import { fetchTranscodeDecision } from './fetchDecision'

describe('transcode module index', () => {
  beforeEach(() => {
    localStorage.setItem('username', 'testuser')
    localStorage.setItem('subsonic-token', 'testtoken')
    localStorage.setItem('subsonic-salt', 'testsalt')
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('exports decisionService with expected methods', () => {
    expect(typeof decisionService.getDecision).toBe('function')
    expect(typeof decisionService.getCachedDecision).toBe('function')
    expect(typeof decisionService.prefetchDecisions).toBe('function')
    expect(typeof decisionService.invalidateAll).toBe('function')
    expect(typeof decisionService.buildStreamUrl).toBe('function')
  })

  it('exports detectBrowserProfile', () => {
    expect(typeof detectBrowserProfile).toBe('function')
  })
})
