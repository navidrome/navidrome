import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import { createTheme } from '@material-ui/core/styles'

const simple = vi.hoisted(() => ({ throwOnDetect: false }))

vi.mock('./simpleMobile', () => ({
  applySimpleMobileDomHint: () => {
    if (simple.throwOnDetect) {
      throw new Error('detect failed')
    }
  },
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
vi.mock('../themes/useCurrentTheme', () => ({
  default: () => createTheme(),
}))
vi.mock('../common', () => ({
  useSearchRefocus: () => {},
}))

import Layout from './Layout'

const renderLayout = (storeState) =>
  render(
    <Provider
      store={createStore(() => ({
        player: { queue: [] },
        admin: { ui: { sidebarOpen: true } },
        ...storeState,
      }))}
    >
      <Layout>
        <div>library</div>
      </Layout>
    </Provider>,
  )

describe('<Layout />', () => {
  beforeEach(() => {
    simple.throwOnDetect = false
  })

  it('always renders the full React-Admin layout (simple chrome is static HTML)', () => {
    renderLayout()
    expect(screen.getByTestId('ra-layout')).toBeInTheDocument()
    expect(screen.getByText('library')).toBeInTheDocument()
  })

  it('still renders RA layout if the simple-mode DOM hint throws', () => {
    simple.throwOnDetect = true
    renderLayout()
    expect(screen.getByTestId('ra-layout')).toBeInTheDocument()
  })
})
