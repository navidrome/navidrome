import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import { createMuiTheme } from '@material-ui/core/styles'
import Login from './Login'

const mocks = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
}))

vi.mock('react-admin', () => ({
  createMuiTheme: (theme) => theme,
  useCheckAuth: () => mocks.checkAuth,
  useLogin: () => vi.fn(() => Promise.resolve()),
  useNotify: () => vi.fn(),
  useTranslate: () => (x) => x,
  useVersion: () => 1,
}))

vi.mock('react-router-dom', () => ({
  useHistory: () => ({ push: mocks.push, replace: mocks.replace }),
}))

vi.mock('../themes/useCurrentTheme', () => ({
  default: () => createMuiTheme({}),
}))

vi.mock('./Notification', () => ({
  default: () => <div />,
}))

const renderLogin = () =>
  render(
    <Provider store={createStore(() => ({}))}>
      <Login location={{}} />
    </Provider>,
  )

describe('<Login />', () => {
  beforeEach(() => {
    mocks.checkAuth.mockReset()
    mocks.push.mockReset()
    mocks.replace.mockReset()
  })

  it('replaces the login page with the home page when already signed in', async () => {
    mocks.checkAuth.mockResolvedValue()

    renderLogin()

    await waitFor(() => expect(mocks.replace).toHaveBeenCalledWith('/'))
    expect(mocks.checkAuth).toHaveBeenCalledWith({}, false)
    expect(mocks.push).not.toHaveBeenCalled()
  })

  it('shows the login form when the user is not signed in', async () => {
    mocks.checkAuth.mockRejectedValue()

    renderLogin()

    await waitFor(() => expect(mocks.checkAuth).toHaveBeenCalledWith({}, false))
    expect(screen.getByText('ra.auth.sign_in')).toBeInTheDocument()
    expect(mocks.replace).not.toHaveBeenCalled()
    expect(mocks.push).not.toHaveBeenCalled()
  })
})
