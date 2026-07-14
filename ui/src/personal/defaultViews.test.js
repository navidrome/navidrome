import {
  getDefaultViewChoices,
  getStoredDefaultView,
  isResourceDefaultView,
  resourceDefaultViews,
} from './defaultViews'
import albumLists, { defaultAlbumList } from '../album/albumLists'

describe('defaultViews', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('includes album lists and top-level resource lists as choices', () => {
    const choices = getDefaultViewChoices((key, options) =>
      options?.smart_count ? `${key}:${options.smart_count}` : key,
    )

    expect(choices.map((choice) => choice.id)).toEqual([
      ...Object.keys(albumLists),
      ...resourceDefaultViews,
    ])
    expect(choices).toEqual(
      expect.arrayContaining([
        { id: 'artist', name: 'resources.artist.name:2' },
        { id: 'song', name: 'resources.song.name:2' },
        { id: 'playlist', name: 'resources.playlist.name:2' },
      ]),
    )
  })

  it('identifies resource-backed default views', () => {
    expect(isResourceDefaultView('artist')).toBe(true)
    expect(isResourceDefaultView('song')).toBe(true)
    expect(isResourceDefaultView('playlist')).toBe(true)
    expect(isResourceDefaultView('recentlyAdded')).toBe(false)
  })

  it('falls back to the default album list when no default view is stored', () => {
    expect(getStoredDefaultView()).toBe(defaultAlbumList)
  })

  it('returns the stored default view', () => {
    localStorage.setItem('defaultView', 'playlist')

    expect(getStoredDefaultView()).toBe('playlist')
  })
})
