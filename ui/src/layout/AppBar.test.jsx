import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, beforeEach, expect, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { activityReducer } from '../reducers'
import AppBar from './AppBar'
import config from '../config'

const mockSetPref = vi.hoisted(() => vi.fn())

vi.mock('./simpleMobile', () => ({
  setSimpleMobilePref: (...args) => mockSetPref(...args),
}))

let store

vi.mock('react-admin', () => ({
  AppBar: ({ userMenu, children }) => (
    <div data-testid="appbar">
      {children}
      {userMenu}
    </div>
  ),
  useTranslate: () => (x) => x,
  usePermissions: () => ({ permissions: 'admin' }),
  getResources: () => [],
  useDataProvider: () => ({ getList: vi.fn() }),
  useNotify: () => vi.fn(),
  Button: ({ children }) => <button>{children}</button>,
}))

vi.mock('./NowPlayingPanel', () => ({
  default: () => <div data-testid="now-playing-panel" />,
}))
vi.mock('./ActivityPanel', () => ({
  default: () => <div data-testid="activity-panel" />,
}))
vi.mock('./PersonalMenu', () => ({
  default: () => <div />,
}))
vi.mock('./UserMenu', () => ({
  default: ({ children }) => <div>{children}</div>,
}))
vi.mock('../dialogs/Dialogs', () => ({
  Dialogs: () => <div />,
}))
vi.mock('../dialogs', () => ({
  AboutDialog: () => <div />,
}))

const renderAppBar = () =>
  render(
    <Provider store={store}>
      <ThemeProvider theme={createTheme()}>
        <AppBar />
      </ThemeProvider>
    </Provider>,
  )

describe('<AppBar />', () => {
  beforeEach(() => {
    mockSetPref.mockClear()
    config.devActivityPanel = true
    config.enableNowPlaying = true
    store = createStore(combineReducers({ activity: activityReducer }), {
      activity: { nowPlayingCount: 0 },
    })
  })

  it('renders NowPlayingPanel when enabled', () => {
    renderAppBar()
    expect(screen.getByTestId('now-playing-panel')).toBeInTheDocument()
  })

  it('hides NowPlayingPanel when disabled', () => {
    config.enableNowPlaying = false
    renderAppBar()
    expect(screen.queryByTestId('now-playing-panel')).toBeNull()
  })

  it('mounts the shuffle control next to react-admin-title', () => {
    renderAppBar()
    expect(document.getElementById('react-admin-title')).toBeInTheDocument()
    expect(screen.getByTestId('title-shuffle-button')).toBeInTheDocument()
    expect(screen.getByLabelText('menu.playRandom')).toBeInTheDocument()
  })

  it('shows a simple-mode menu item that opts in via setSimpleMobilePref', () => {
    renderAppBar()
    const item = screen.getByTestId('simple-mode-menu-item')
    expect(item).toHaveTextContent('menu.simpleMode')
    fireEvent.click(item)
    expect(mockSetPref).toHaveBeenCalledWith(true)
  })
})
