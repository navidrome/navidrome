import React from 'react'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { settingsReducer, activityReducer } from '../reducers'
import { processEvent, EVENT_REFRESH_RESOURCE } from '../actions'
import PlaylistsSubMenu from './PlaylistsSubMenu'

const mockUseQueryWithStore = vi.fn()

vi.mock('../config', () => ({
  // losslessFormats is read at module-load time by common/QualityInfo.jsx,
  // pulled in transitively via the '../common' barrel file
  default: {
    enableFavourites: true,
    maxSidebarPlaylists: 100,
    losslessFormats: '',
  },
}))

vi.mock('react-dnd', () => ({
  useDrop: () => [{}, () => {}],
}))

vi.mock('react-router-dom', () => ({
  useHistory: () => ({ push: vi.fn() }),
}))

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useTranslate: () => (x) => x,
    useDataProvider: () => ({ addToPlaylist: vi.fn() }),
    useNotify: () => vi.fn(),
    useQueryWithStore: (query) => mockUseQueryWithStore(query),
    MenuItemLink: ({ primaryText }) => <div>{primaryText}</div>,
  }
})

const playlists = {
  'pl-1': { id: 'pl-1', name: 'Mine', ownerId: 'user-1' },
  'pl-2': { id: 'pl-2', name: 'Theirs', ownerId: 'user-2' },
}

const renderMenu = (preloadedSettings = {}) => {
  const store = createStore(
    combineReducers({
      settings: settingsReducer,
      activity: activityReducer,
    }),
    { settings: preloadedSettings, activity: {} },
  )
  const theme = createTheme()
  render(
    <Provider store={store}>
      <ThemeProvider theme={theme}>
        <PlaylistsSubMenu
          state={{ menuPlaylists: true, menuSharedPlaylists: true }}
          setState={vi.fn()}
          sidebarIsOpen={true}
          dense={false}
        />
      </ThemeProvider>
    </Provider>,
  )
  return store
}

const lastQuery = () =>
  mockUseQueryWithStore.mock.calls[
    mockUseQueryWithStore.mock.calls.length - 1
  ][0]

describe('<PlaylistsSubMenu />', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('userId', 'user-1')
    mockUseQueryWithStore.mockReturnValue({ data: playlists, loaded: true })
    // SubMenu uses MUI's useMediaQuery, which needs window.matchMedia in jsdom
    window.matchMedia = (query) => ({
      matches: false,
      media: query,
      addListener: () => {},
      removeListener: () => {},
    })
    // OverflowTooltip (via MenuItemLink) needs ResizeObserver, unavailable in jsdom
    window.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  it('queries without a starred filter by default', () => {
    renderMenu()
    expect(lastQuery().payload.filter).toBeUndefined()
    expect(screen.getByText('Mine')).not.toBeNull()
    expect(screen.getByText('Theirs')).not.toBeNull()
  })

  it('adds the starred filter when favourites-only is enabled', () => {
    renderMenu({ sidebarPlaylistsOnlyFavourites: true })
    expect(lastQuery().payload.filter).toEqual({ starred: true })
  })

  it('toggles the setting when the heart action is clicked', () => {
    const store = renderMenu()
    fireEvent.click(screen.getByTitle('menu.onlyFavourites'))
    expect(store.getState().settings.sidebarPlaylistsOnlyFavourites).toBe(true)
    expect(lastQuery().payload.filter).toEqual({ starred: true })
  })

  it('refetches when a playlist refresh event arrives', async () => {
    const store = renderMenu()
    const before = lastQuery().payload.refresh
    // useRefreshOnEvents compares Date.now() timestamps; make sure it advances
    await act(() => new Promise((resolve) => setTimeout(resolve, 5)))
    act(() => {
      store.dispatch(
        processEvent(EVENT_REFRESH_RESOURCE, { playlist: ['pl-1'] }),
      )
    })
    expect(lastQuery().payload.refresh).toBe(before + 1)
  })
})
