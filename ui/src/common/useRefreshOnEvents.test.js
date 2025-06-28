import { vi } from 'vitest'
import * as React from 'react'
import * as Redux from 'react-redux'
import { useRefreshOnEvents } from './useRefreshOnEvents'

vi.mock('react', async () => {
  const actual = await vi.importActual('react')
  return {
    ...actual,
    useState: vi.fn(),
    useEffect: vi.fn(),
  }
})

vi.mock('react-redux', async () => {
  const actual = await vi.importActual('react-redux')
  return {
    ...actual,
    useSelector: vi.fn(),
  }
})

describe('useRefreshOnEvents', () => {
  const setState = vi.fn()
  const useStateMock = (initState) => [initState, setState]
  const onRefresh = vi.fn().mockResolvedValue()
  let lastTime
  let mockUseEffect

  beforeEach(() => {
    vi.spyOn(React, 'useState').mockImplementation(useStateMock)
    mockUseEffect = vi.spyOn(React, 'useEffect')
    lastTime = new Date(new Date().valueOf() + 1000)
    onRefresh.mockClear()
    setState.mockClear()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('stores last time checked, to avoid redundant runs', () => {
    const useSelectorMock = () => ({
      lastReceived: lastTime,
      resources: { library: ['lib-1'] }, // Need some resources to trigger the update
    })
    vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

    // Mock useEffect to immediately call the effect callback
    mockUseEffect.mockImplementation((callback) => callback())

    useRefreshOnEvents({
      events: ['library'],
      onRefresh,
    })

    expect(setState).toHaveBeenCalledWith(lastTime)
  })

  it("does not run again if lastTime didn't change", () => {
    vi.spyOn(React, 'useState').mockImplementation(() => [lastTime, setState])
    const useSelectorMock = () => ({ lastReceived: lastTime })
    vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

    // Mock useEffect to immediately call the effect callback
    mockUseEffect.mockImplementation((callback) => callback())

    useRefreshOnEvents({
      events: ['library'],
      onRefresh,
    })

    expect(setState).not.toHaveBeenCalled()
    expect(onRefresh).not.toHaveBeenCalled()
  })

  describe('Event listening and refresh triggering', () => {
    beforeEach(() => {
      // Mock useEffect to immediately call the effect callback
      mockUseEffect.mockImplementation((callback) => callback())
    })

    it('triggers refresh when a watched event occurs', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { library: ['lib-1', 'lib-2'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useRefreshOnEvents({
        events: ['library'],
        onRefresh,
      })

      expect(onRefresh).toHaveBeenCalledTimes(1)
      expect(setState).toHaveBeenCalledWith(lastTime)
    })

    it('triggers refresh when multiple watched events occur', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: {
          library: ['lib-1'],
          user: ['user-1'],
          album: ['album-1'], // This shouldn't trigger since it's not watched
        },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useRefreshOnEvents({
        events: ['library', 'user'],
        onRefresh,
      })

      expect(onRefresh).toHaveBeenCalledTimes(1)
      expect(setState).toHaveBeenCalledWith(lastTime)
    })

    it('does not trigger refresh when unwatched events occur', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { album: ['album-1'], song: ['song-1'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useRefreshOnEvents({
        events: ['library', 'user'],
        onRefresh,
      })

      expect(onRefresh).not.toHaveBeenCalled()
      expect(setState).not.toHaveBeenCalled()
    })

    it('triggers refresh on global refresh event', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { '*': '*' },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useRefreshOnEvents({
        events: ['library'],
        onRefresh,
      })

      expect(onRefresh).toHaveBeenCalledTimes(1)
      expect(setState).toHaveBeenCalledWith(lastTime)
    })

    it('triggers refresh when listening to all events with "*"', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { song: ['song-1'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useRefreshOnEvents({
        events: ['*'],
        onRefresh,
      })

      expect(onRefresh).toHaveBeenCalledTimes(1)
      expect(setState).toHaveBeenCalledWith(lastTime)
    })

    it('handles empty events array gracefully', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { library: ['lib-1'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useRefreshOnEvents({
        events: [],
        onRefresh,
      })

      expect(onRefresh).not.toHaveBeenCalled()
      expect(setState).not.toHaveBeenCalled()
    })

    it('handles missing onRefresh function gracefully', () => {
      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { library: ['lib-1'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      expect(() => {
        useRefreshOnEvents({
          events: ['library'],
          // onRefresh is undefined
        })
      }).not.toThrow()

      expect(setState).toHaveBeenCalledWith(lastTime)
    })

    it('handles onRefresh errors gracefully', async () => {
      const consoleWarnSpy = vi
        .spyOn(console, 'warn')
        .mockImplementation(() => {})
      const failingRefresh = vi
        .fn()
        .mockRejectedValue(new Error('Refresh failed'))

      const useSelectorMock = () => ({
        lastReceived: lastTime,
        resources: { library: ['lib-1'] },
      })
      vi.spyOn(Redux, 'useSelector').mockImplementation(useSelectorMock)

      useRefreshOnEvents({
        events: ['library'],
        onRefresh: failingRefresh,
      })

      expect(failingRefresh).toHaveBeenCalledTimes(1)
      expect(setState).toHaveBeenCalledWith(lastTime)

      // Wait for the promise to be rejected and handled
      await new Promise((resolve) => setTimeout(resolve, 10))

      expect(consoleWarnSpy).toHaveBeenCalledWith(
        'Error in useRefreshOnEvents onRefresh callback:',
        expect.any(Error),
      )

      consoleWarnSpy.mockRestore()
    })
  })
})
