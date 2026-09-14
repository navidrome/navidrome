import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import SimpleMobileLayout from './SimpleMobileLayout'

const mockSetPref = vi.fn()

vi.mock('react-admin', () => ({
  useTranslate: () => (x) => x,
  Notification: () => <div data-testid="notification" />,
}))

vi.mock('../themes/useCurrentTheme', () => ({
  default: () => createTheme(),
}))

vi.mock('../common/ShuffleAllButton', () => ({
  ShuffleAllButton: ({ variant }) => (
    <button data-testid="shuffle-all-hero" data-variant={variant}>
      menu.playRandom
    </button>
  ),
}))

vi.mock('./Notification', () => ({
  default: () => <div data-testid="notification" />,
}))

vi.mock('./simpleMobile', () => ({
  setSimpleMobilePref: (...args) => mockSetPref(...args),
}))

const renderLayout = (storeState = { player: { queue: [] } }) =>
  render(
    <Provider store={createStore(() => storeState)}>
      <ThemeProvider theme={createTheme()}>
        <SimpleMobileLayout />
      </ThemeProvider>
    </Provider>,
  )

describe('<SimpleMobileLayout />', () => {
  beforeEach(() => {
    mockSetPref.mockClear()
  })

  it('shows shuffle and open-full-version actions', () => {
    renderLayout()
    expect(screen.getByTestId('simple-mobile-layout')).toBeInTheDocument()
    expect(screen.getByTestId('shuffle-all-hero')).toHaveAttribute(
      'data-variant',
      'hero',
    )
    expect(screen.getByTestId('open-full-version')).toHaveTextContent(
      'menu.openFullVersion',
    )
  })

  it('still shows the two actions when the persisted queue is missing', () => {
    renderLayout({})
    expect(screen.getByTestId('simple-mobile-layout')).toBeInTheDocument()
    expect(screen.getByTestId('shuffle-all-hero')).toBeInTheDocument()
    expect(screen.getByTestId('open-full-version')).toBeInTheDocument()
  })

  it('persists full-UI preference when opening the full version', () => {
    renderLayout()
    fireEvent.click(screen.getByTestId('open-full-version'))
    expect(mockSetPref).toHaveBeenCalledWith(false)
  })
})
