import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { activityReducer } from '../reducers'
import AppBar from './AppBar'
import config from '../config'

let store

const mocks = vi.hoisted(() => ({ resources: [] }))

vi.mock('react-admin', () => ({
  AppBar: ({ userMenu }) => <div data-testid="appbar">{userMenu}</div>,
  MenuItemLink: ({ primaryText }) => <div>{primaryText}</div>,
  useTranslate: () => (x) => x,
  usePermissions: () => ({ permissions: 'admin' }),
  getResources: () => mocks.resources,
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
  QuickConnectDialog: () => <div />,
}))

describe('<AppBar />', () => {
  beforeEach(() => {
    config.devActivityPanel = true
    config.enableNowPlaying = true
    config.enableQuickConnect = false
    mocks.resources = []
    store = createStore(combineReducers({ activity: activityReducer }), {
      activity: { nowPlayingCount: 0 },
    })
  })

  it('renders NowPlayingPanel when enabled', () => {
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.getByTestId('now-playing-panel')).toBeInTheDocument()
  })

  it('hides NowPlayingPanel when disabled', () => {
    config.enableNowPlaying = false
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.queryByTestId('now-playing-panel')).toBeNull()
  })

  it('shows the Quick Connect menu item when enabled', () => {
    config.enableQuickConnect = true
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.queryAllByText('menu.quickConnect.name')).not.toHaveLength(0)
  })

  it('hides the Quick Connect menu item when disabled', () => {
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.queryAllByText('menu.quickConnect.name')).toHaveLength(0)
    expect(screen.queryAllByText('menu.about')).not.toHaveLength(0)
  })

  it('uses the resource label for settings items when set', () => {
    mocks.resources = [
      {
        name: 'player',
        hasList: true,
        options: { subMenu: 'settings', label: 'resources.player.menuName' },
      },
      { name: 'transcoding', hasList: true, options: { subMenu: 'settings' } },
    ]
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.getByText('resources.player.menuName')).toBeInTheDocument()
    expect(screen.getByText('resources.transcoding.name')).toBeInTheDocument()
  })
})
