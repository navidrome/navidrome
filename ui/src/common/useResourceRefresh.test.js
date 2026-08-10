import { vi } from 'vitest'
import * as React from 'react'
import * as Redux from 'react-redux'
import * as RA from 'react-admin'
import { useResourceRefresh } from './useResourceRefresh'

vi.mock('react', async () => {
  const actual = await vi.importActual('react')
  return {
    ...actual,
    useState: vi.fn(),
  }
})

vi.mock('react-redux', async () => {
  const actual = await vi.importActual('react-redux')
  return {
    ...actual,
    useSelector: vi.fn(),
  }
})

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useRefresh: vi.fn(),
    useDataProvider: vi.fn(),
  }
})

describe('useResourceRefresh', () => {
  const setState = vi.fn()
  const useStateMock = (initState) => [initState, setState]
  const refresh = vi.fn()
  const useRefreshMock = () => refresh
  const getMany = vi.fn()
  const useDataProviderMock = () => ({ getMany })
  let lastTime

  // Applies each real selector against a constructed store, so both useSelector
  // calls in the hook (refresh payload + loaded records) resolve correctly.
  const mockStore = ({ refresh: refreshPayload, loaded = {} }) => {
    const state = {
      activity: { refresh: refreshPayload },
      admin: { resources: loaded },
    }
    vi.spyOn(Redux, 'useSelector').mockImplementation((selector) =>
      selector(state),
    )
  }

  // Turns id lists into the react-admin store shape: { album: { data: { 'al-1': {...} } } }
  const asStore = (byResource) =>
    Object.fromEntries(
      Object.entries(byResource).map(([r, ids]) => [
        r,
        { data: Object.fromEntries(ids.map((id) => [id, { id }])) },
      ]),
    )

  beforeEach(() => {
    vi.spyOn(React, 'useState').mockImplementation(useStateMock)
    vi.spyOn(RA, 'useRefresh').mockImplementation(useRefreshMock)
    vi.spyOn(RA, 'useDataProvider').mockImplementation(useDataProviderMock)
    lastTime = new Date(new Date().valueOf() + 1000)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('stores last time checked, to avoid redundant runs', () => {
    mockStore({ refresh: { lastReceived: lastTime } })

    useResourceRefresh()

    expect(setState).toHaveBeenCalledWith(lastTime)
  })

  it("does not run again if lastTime didn't change", () => {
    vi.spyOn(React, 'useState').mockImplementation(() => [lastTime, setState])
    mockStore({ refresh: { lastReceived: lastTime } })

    useResourceRefresh()

    expect(setState).not.toHaveBeenCalled()
  })

  describe('No visible resources specified', () => {
    it('triggers a UI refresh when received a "any" resource refresh', () => {
      mockStore({
        refresh: { lastReceived: lastTime, resources: { '*': '*' } },
      })

      useResourceRefresh()

      expect(refresh).toHaveBeenCalledTimes(1)
      expect(getMany).not.toHaveBeenCalled()
    })

    it('triggers a UI refresh when received an "any" id', () => {
      mockStore({
        refresh: { lastReceived: lastTime, resources: { album: ['*'] } },
      })

      useResourceRefresh()

      expect(refresh).toHaveBeenCalledTimes(1)
      expect(getMany).not.toHaveBeenCalled()
    })

    it('refetches only the received resources already loaded in the store', () => {
      mockStore({
        refresh: {
          lastReceived: lastTime,
          resources: { album: ['al-1', 'al-2'], song: ['sg-1', 'sg-2'] },
        },
        loaded: asStore({ album: ['al-1', 'al-2'], song: ['sg-1', 'sg-2'] }),
      })

      useResourceRefresh()

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledTimes(2)
      expect(getMany).toHaveBeenCalledWith('album', { ids: ['al-1', 'al-2'] })
      expect(getMany).toHaveBeenCalledWith('song', { ids: ['sg-1', 'sg-2'] })
    })

    it('skips ids that are not loaded in the store', () => {
      mockStore({
        refresh: {
          lastReceived: lastTime,
          resources: { album: ['al-1', 'al-2', 'al-3'] },
        },
        loaded: asStore({ album: ['al-2'] }),
      })

      useResourceRefresh()

      expect(getMany).toHaveBeenCalledTimes(1)
      expect(getMany).toHaveBeenCalledWith('album', { ids: ['al-2'] })
    })

    it('does not fetch when none of the received ids are loaded', () => {
      mockStore({
        refresh: {
          lastReceived: lastTime,
          resources: { artist: ['ar-1', 'ar-2'] },
        },
        loaded: asStore({ artist: ['ar-9'] }),
      })

      useResourceRefresh()

      expect(getMany).not.toHaveBeenCalled()
    })
  })

  describe('Visible resources specified', () => {
    it('triggers a UI refresh when received a "any" resource refresh', () => {
      mockStore({
        refresh: { lastReceived: lastTime, resources: { '*': '*' } },
      })

      useResourceRefresh('album')

      expect(refresh).toHaveBeenCalledTimes(1)
      expect(getMany).not.toHaveBeenCalled()
    })

    it('does not refresh the page when the wildcard is on a resource it does not watch', () => {
      // Guards other senders: a wildcard on an unwatched resource must not reload this list.
      mockStore({
        refresh: {
          lastReceived: lastTime,
          resources: { album: ['al-1', 'al-2'], song: ['*'] },
        },
        loaded: asStore({ album: ['al-1', 'al-2'] }),
      })

      useResourceRefresh('album')

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledWith('album', { ids: ['al-1', 'al-2'] })
    })

    it('refetches the loaded songs of a refreshed album', () => {
      // A track with no art of its own is served its album's, so its coverArt id moves with the
      // album's. The backend cannot know which tracks are loaded; the store can.
      mockStore({
        refresh: { lastReceived: lastTime, resources: { album: ['al-1'] } },
        loaded: {
          album: { data: { 'al-1': { id: 'al-1' } } },
          song: {
            data: {
              'sg-1': { id: 'sg-1', albumId: 'al-1' },
              'sg-2': { id: 'sg-2', albumId: 'al-2' },
            },
          },
        },
      })

      useResourceRefresh('song')

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledWith('song', { ids: ['sg-1'] })
    })

    it('fans an album refresh out to loaded playlist tracks, which have their own ids', () => {
      mockStore({
        refresh: { lastReceived: lastTime, resources: { album: ['al-1'] } },
        loaded: {
          playlistTrack: {
            data: {
              'pt-1': { id: 'pt-1', albumId: 'al-1' },
              'pt-2': { id: 'pt-2', albumId: 'al-2' },
            },
          },
        },
      })

      useResourceRefresh('playlistTrack', 'song', 'playlist')

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledWith('playlistTrack', { ids: ['pt-1'] })
    })

    it('does not fan out to track resources the component does not show', () => {
      mockStore({
        refresh: { lastReceived: lastTime, resources: { album: ['al-1'] } },
        loaded: {
          album: { data: { 'al-1': { id: 'al-1' } } },
          song: { data: { 'sg-1': { id: 'sg-1', albumId: 'al-1' } } },
        },
      })

      useResourceRefresh('album')

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledTimes(1)
      expect(getMany).toHaveBeenCalledWith('album', { ids: ['al-1'] })
    })

    it('does not refetch songs when no loaded song belongs to the refreshed album', () => {
      mockStore({
        refresh: { lastReceived: lastTime, resources: { album: ['al-9'] } },
        loaded: {
          song: { data: { 'sg-1': { id: 'sg-1', albumId: 'al-1' } } },
        },
      })

      useResourceRefresh('song')

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).not.toHaveBeenCalled()
    })

    it('refetches the received resources if they are visible and loaded', () => {
      mockStore({
        refresh: {
          lastReceived: lastTime,
          resources: { album: ['al-1', 'al-2'], song: ['sg-1', 'sg-2'] },
        },
        loaded: asStore({ album: ['al-1', 'al-2'], song: ['sg-1', 'sg-2'] }),
      })

      useResourceRefresh('song')

      expect(refresh).not.toHaveBeenCalled()
      expect(getMany).toHaveBeenCalledTimes(1)
      expect(getMany).toHaveBeenCalledWith('song', { ids: ['sg-1', 'sg-2'] })
    })
  })
})
