import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import { createTheme } from '@material-ui/core/styles'

const simple = vi.hoisted(() => ({ on: false }))

vi.mock('./simpleMobile', () => ({
  shouldUseSimpleMobile: () => simple.on,
  applySimpleMobileDomHint: vi.fn(),
}))

vi.mock('react-admin', () => ({
  Layout: ({ children }) => <div data-testid="ra-layout">{children}</div>,
  toggleSidebar: () => ({ type: 'TOGGLE' }),
}))

vi.mock('react-hotkeys', () => ({
  HotKeys: ({ children }) => children,
}))

vi.mock('./Menu', () => ({ default: () => null }))
vi.mock('./AppBar', () => ({ default: () => null }))
vi.mock('./Notification', () => ({ default: () => null }))
vi.mock('./SimpleMobileLayout', () => ({
  default: () => <div data-testid="simple-mobile-layout" />,
}))
vi.mock('../themes/useCurrentTheme', () => ({
  default: () => createTheme(),
}))
vi.mock('../common', () => ({
  useSearchRefocus: () => {},
}))

import Layout from './Layout'

const renderLayout = () =>
  render(
    <Provider
      store={createStore(() => ({
        player: { queue: [] },
        admin: { ui: { sidebarOpen: true } },
      }))}
    >
      <Layout>
        <div>library</div>
      </Layout>
    </Provider>,
  )

describe('<Layout /> simple mobile routing', () => {
  beforeEach(() => {
    simple.on = false
  })

  it('renders the full React-Admin layout when simple mode is off', () => {
    renderLayout()
    expect(screen.getByTestId('ra-layout')).toBeInTheDocument()
    expect(screen.queryByTestId('simple-mobile-layout')).toBeNull()
  })

  it('renders SimpleMobileLayout instead of RA chrome when simple mode is on', () => {
    simple.on = true
    renderLayout()
    expect(screen.getByTestId('simple-mobile-layout')).toBeInTheDocument()
    expect(screen.queryByTestId('ra-layout')).toBeNull()
  })
})
