import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Provider } from 'react-redux'
import { createStore } from 'redux'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { ShuffleAllButton } from './ShuffleAllButton'

const mockGetList = vi.fn()
const mockNotify = vi.fn()
const mockDispatch = vi.fn()

vi.mock('react-admin', () => ({
  Button: ({ onClick, label, children, ...rest }) => (
    <button onClick={onClick} {...rest}>
      {label}
      {children}
    </button>
  ),
  useTranslate: () => (x) => x,
  useDataProvider: () => ({ getList: mockGetList }),
  useNotify: () => mockNotify,
}))

vi.mock('react-redux', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useDispatch: () => mockDispatch,
  }
})

const renderButton = (props) =>
  render(
    <Provider store={createStore(() => ({}))}>
      <ThemeProvider theme={createTheme()}>
        <ShuffleAllButton {...props} />
      </ThemeProvider>
    </Provider>,
  )

describe('<ShuffleAllButton />', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    delete window.__ndPlayRandom
    mockGetList.mockResolvedValue({
      data: [
        { id: 's1', artist: 'A', album: 'X' },
        { id: 's2', artist: 'B', album: 'Y' },
      ],
    })
  })

  afterEach(() => {
    delete window.__ndPlayRandom
  })

  it('fetches random songs and plays a shuffled queue', async () => {
    renderButton()
    fireEvent.click(screen.getByTestId('shuffle-all-button'))

    await waitFor(() =>
      expect(mockGetList).toHaveBeenCalledWith('song', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'random', order: 'ASC' },
        filter: { missing: false },
      }),
    )
    expect(mockDispatch).toHaveBeenCalled()
    const action = mockDispatch.mock.calls[0][0]
    expect(action.type).toBe('PLAYER_PLAY_TRACKS')
    expect(Object.keys(action.data)).toHaveLength(2)
  })

  it('renders an app-bar icon button with an accessible label', () => {
    renderButton({ variant: 'icon' })
    const button = screen.getByTestId('title-shuffle-button')
    expect(button).toHaveAttribute('aria-label', 'menu.playRandom')
  })

  it('renders a large hero button labeled playRandom', () => {
    renderButton({ variant: 'hero' })
    const button = screen.getByTestId('shuffle-all-hero')
    expect(button).toHaveTextContent('menu.playRandom')
  })

  it('exposes window.__ndPlayRandom for the static simple-mode shell', async () => {
    renderButton()
    expect(typeof window.__ndPlayRandom).toBe('function')
    window.__ndPlayRandom()
    await waitFor(() => expect(mockGetList).toHaveBeenCalled())
    expect(mockDispatch).toHaveBeenCalled()
  })
})
