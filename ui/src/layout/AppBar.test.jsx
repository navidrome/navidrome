import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { activityReducer } from '../reducers'
import AppBar from './AppBar'
import config from '../config'

let store
let mockPermissions = 'admin'

vi.mock('react-admin', () => ({
  AppBar: ({ userMenu }) => <div data-testid="appbar">{userMenu}</div>,
  useTranslate: () => (x) => x,
  usePermissions: () => ({ permissions: mockPermissions }),
  getResources: () => [],
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

describe('<AppBar />', () => {
  beforeEach(() => {
    config.devActivityPanel = true
    config.enableNowPlaying = true
    config.nowPlayingAdminOnly = true
    mockPermissions = 'admin'
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

  it('shows NowPlayingPanel to all users when adminOnly is false', () => {
    config.nowPlayingAdminOnly = false
    render(
      <Provider store={store}>
        <AppBar />
      </Provider>,
    )
    expect(screen.getByTestId('now-playing-panel')).toBeInTheDocument()
  })

  describe('admin-only mode', () => {
    beforeEach(() => {
      config.nowPlayingAdminOnly = true
    })

    it('shows NowPlayingPanel to admin users', () => {
      mockPermissions = 'admin'
      render(
        <Provider store={store}>
          <AppBar />
        </Provider>,
      )
      expect(screen.getByTestId('now-playing-panel')).toBeInTheDocument()
    })

    it('hides NowPlayingPanel from non-admin users', () => {
      mockPermissions = 'user'
      render(
        <Provider store={store}>
          <AppBar />
        </Provider>,
      )
      expect(screen.queryByTestId('now-playing-panel')).toBeNull()
    })
  })

  describe('non-admin users', () => {
    beforeEach(() => {
      mockPermissions = 'user'
    })

    it('cannot see NowPlayingPanel when adminOnly is true', () => {
      config.nowPlayingAdminOnly = true
      render(
        <Provider store={store}>
          <AppBar />
        </Provider>,
      )
      expect(screen.queryByTestId('now-playing-panel')).toBeNull()
    })

    it('can see NowPlayingPanel when adminOnly is false', () => {
      config.nowPlayingAdminOnly = false
      render(
        <Provider store={store}>
          <AppBar />
        </Provider>,
      )
      expect(screen.getByTestId('now-playing-panel')).toBeInTheDocument()
    })

    it('cannot see NowPlayingPanel when feature is disabled', () => {
      config.enableNowPlaying = false
      config.nowPlayingAdminOnly = false
      render(
        <Provider store={store}>
          <AppBar />
        </Provider>,
      )
      expect(screen.queryByTestId('now-playing-panel')).toBeNull()
    })
  })
})
