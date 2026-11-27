import { describe, it, expect } from 'vitest'
import { settingsReducer } from './settingsReducer'
import {
  SET_SIDEBAR_PLAYLISTS_FAVOURITES,
  setSidebarPlaylistsOnlyFavourites,
} from '../actions'

describe('settingsReducer', () => {
  it('defaults sidebarPlaylistsOnlyFavourites to false', () => {
    const state = settingsReducer(undefined, { type: 'UNKNOWN' })
    expect(state.sidebarPlaylistsOnlyFavourites).toBe(false)
  })

  it('enables the flag via the action creator', () => {
    const state = settingsReducer(
      undefined,
      setSidebarPlaylistsOnlyFavourites(true),
    )
    expect(state.sidebarPlaylistsOnlyFavourites).toBe(true)
  })

  it('disables the flag and preserves other settings', () => {
    const initial = settingsReducer(undefined, { type: 'UNKNOWN' })
    const on = settingsReducer(initial, {
      type: SET_SIDEBAR_PLAYLISTS_FAVOURITES,
      data: true,
    })
    const off = settingsReducer(on, {
      type: SET_SIDEBAR_PLAYLISTS_FAVOURITES,
      data: false,
    })
    expect(off.sidebarPlaylistsOnlyFavourites).toBe(false)
    expect(off.notifications).toEqual(initial.notifications)
    expect(off.toggleableFields).toEqual(initial.toggleableFields)
  })
})
