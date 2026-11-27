import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createHashHistory } from './hashHistory'

// Resolves after `count` popstate events. Register it before navigating, so
// events queued while handling an earlier one (e.g. a revert) are not missed.
const nextPop = (count = 1) =>
  new Promise((resolve) => {
    let seen = 0
    const onPop = () => {
      if (++seen < count) return
      window.removeEventListener('popstate', onPop)
      setTimeout(resolve, 0)
    }
    window.addEventListener('popstate', onPop)
  })

describe('createHashHistory', () => {
  let history

  beforeEach(() => {
    window.history.replaceState(null, '', '/app/#/album/all')
  })

  afterEach(() => {
    history?.destroy()
    history = undefined
    vi.restoreAllMocks()
  })

  it('reads the initial location from the hash', () => {
    window.history.replaceState(null, '', '/app/#/song?filter=%7B%7D')
    history = createHashHistory()
    expect(history.location.pathname).toBe('/song')
    expect(history.location.search).toBe('?filter=%7B%7D')
    expect(history.action).toBe('POP')
  })

  it('normalizes a missing hash to #/ without adding an entry', () => {
    window.history.replaceState(null, '', '/app/')
    const length = window.history.length
    history = createHashHistory()
    expect(window.location.hash).toBe('#/')
    expect(window.history.length).toBe(length)
    expect(history.location.pathname).toBe('/')
  })

  it('creates hash hrefs', () => {
    history = createHashHistory()
    expect(history.createHref({ pathname: '/song', search: '?a=1' })).toBe(
      '#/song?a=1',
    )
  })

  describe('push', () => {
    // Assigning location.hash is a fragment navigation, which Safari turns into a full reload
    // on insecure (plain HTTP, non-localhost) origins. See #6363.
    it('updates the URL with pushState, not a fragment navigation', () => {
      history = createHashHistory()
      const pushState = vi.spyOn(window.history, 'pushState')
      const onHashChange = vi.fn()
      window.addEventListener('hashchange', onHashChange)

      history.push('/song?filter=%7B%22title%22%3A%22x%22%7D')

      window.removeEventListener('hashchange', onHashChange)
      expect(pushState).toHaveBeenCalledTimes(1)
      expect(window.location.pathname).toBe('/app/')
      expect(window.location.hash).toBe(
        '#/song?filter=%7B%22title%22%3A%22x%22%7D',
      )
      expect(onHashChange).not.toHaveBeenCalled()
    })

    it('notifies listeners and updates location and action', () => {
      history = createHashHistory()
      const listener = vi.fn()
      history.listen(listener)

      history.push('/artist', { from: 'menu' })

      expect(listener).toHaveBeenCalledTimes(1)
      const [location, action] = listener.mock.calls[0]
      expect(action).toBe('PUSH')
      expect(location.pathname).toBe('/artist')
      expect(location.state).toEqual({ from: 'menu' })
      expect(history.location).toBe(location)
      expect(history.action).toBe('PUSH')
    })

    it('accepts location objects and relative paths', () => {
      history = createHashHistory()
      history.push({ pathname: '/album/1/show', search: '?x=1' })
      expect(window.location.hash).toBe('#/album/1/show?x=1')
      history.push('?x=2')
      expect(history.location.pathname).toBe('/album/1/show')
      expect(history.location.search).toBe('?x=2')
    })

    it('does not add an entry when pushing the current path', () => {
      history = createHashHistory()
      const length = window.history.length
      history.push('/album/all')
      expect(window.history.length).toBe(length)
    })
  })

  describe('replace', () => {
    it('updates the URL with replaceState and notifies listeners', () => {
      history = createHashHistory()
      const replaceState = vi.spyOn(window.history, 'replaceState')
      const listener = vi.fn()
      history.listen(listener)
      const length = window.history.length

      history.replace('/album/recentlyAdded')

      expect(replaceState).toHaveBeenCalledTimes(1)
      expect(window.history.length).toBe(length)
      expect(window.location.hash).toBe('#/album/recentlyAdded')
      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({ pathname: '/album/recentlyAdded' }),
        'REPLACE',
      )
    })
  })

  describe('pop', () => {
    it('notifies listeners once when going back', async () => {
      history = createHashHistory()
      history.push('/song')
      const listener = vi.fn()
      history.listen(listener)

      const popped = nextPop()
      history.goBack()
      await popped

      expect(listener).toHaveBeenCalledTimes(1)
      expect(listener.mock.calls[0][1]).toBe('POP')
      expect(history.location.pathname).toBe('/album/all')
    })

    it('restores state when going back', async () => {
      history = createHashHistory()
      history.push('/song', { a: 1 })
      history.push('/artist')

      const popped = nextPop()
      history.go(-1)
      await popped

      expect(history.location.pathname).toBe('/song')
      expect(history.location.state).toEqual({ a: 1 })
    })

    it('handles native fragment navigation (e.g. editing the address bar)', async () => {
      history = createHashHistory()
      const listener = vi.fn()
      history.listen(listener)

      const changed = new Promise((resolve) =>
        window.addEventListener('hashchange', () => setTimeout(resolve, 0), {
          once: true,
        }),
      )
      window.location.hash = '#/playlist'
      await changed

      expect(listener).toHaveBeenCalledTimes(1)
      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({ pathname: '/playlist' }),
        'POP',
      )
    })
  })

  describe('listen', () => {
    it('returns an unlisten function', () => {
      history = createHashHistory()
      const listener = vi.fn()
      const unlisten = history.listen(listener)
      unlisten()
      history.push('/song')
      expect(listener).not.toHaveBeenCalled()
    })
  })

  describe('block', () => {
    it('asks for confirmation and cancels the transition when denied', () => {
      const getUserConfirmation = vi.fn((message, callback) => callback(false))
      history = createHashHistory({ getUserConfirmation })
      history.block('Discard changes?')

      history.push('/song')

      expect(getUserConfirmation).toHaveBeenCalledWith(
        'Discard changes?',
        expect.any(Function),
      )
      expect(window.location.hash).toBe('#/album/all')
      expect(history.location.pathname).toBe('/album/all')
    })

    it('lets the transition through when confirmed', () => {
      const getUserConfirmation = vi.fn((message, callback) => callback(true))
      history = createHashHistory({ getUserConfirmation })
      history.block(() => 'Discard changes?')
      history.push('/song')
      expect(history.location.pathname).toBe('/song')
    })

    it('stops blocking after unblock', () => {
      const getUserConfirmation = vi.fn()
      history = createHashHistory({ getUserConfirmation })
      const unblock = history.block('Discard changes?')
      unblock()
      history.push('/song')
      expect(getUserConfirmation).not.toHaveBeenCalled()
      expect(history.location.pathname).toBe('/song')
    })

    it('reverts a blocked back navigation', async () => {
      const getUserConfirmation = vi.fn((message, callback) => callback(false))
      history = createHashHistory({ getUserConfirmation })
      history.push('/song')
      history.block('Discard changes?')
      const listener = vi.fn()
      history.listen(listener)

      // the blocked back navigation, then the revert forward to /song
      const popped = nextPop(2)
      history.goBack()
      await popped

      expect(getUserConfirmation).toHaveBeenCalledTimes(1)
      expect(listener).not.toHaveBeenCalled()
      expect(window.location.hash).toBe('#/song')
      expect(history.location.pathname).toBe('/song')
    })

    it('handles the next back navigation normally after reverting one', async () => {
      const getUserConfirmation = vi.fn((message, callback) => callback(false))
      history = createHashHistory({ getUserConfirmation })
      history.push('/song')
      const unblock = history.block('Discard changes?')

      const reverted = nextPop(2)
      history.goBack()
      await reverted
      unblock()

      const listener = vi.fn()
      history.listen(listener)
      const popped = nextPop()
      history.goBack()
      await popped

      expect(listener).toHaveBeenCalledTimes(1)
      expect(history.location.pathname).toBe('/album/all')
    })
  })
})
