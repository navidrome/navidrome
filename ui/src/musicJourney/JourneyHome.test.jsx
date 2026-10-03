import * as React from 'react'
import {
  render,
  screen,
  waitFor,
  fireEvent,
  cleanup,
} from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { journeyReducer } from './state/journeyReducer'
import { playerReducer } from '../reducers'
import { clearSongPoolCache } from './hooks/fetchSongPool'
import JourneyHome from './JourneyHome'

// ---- mocks ---------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  dataProvider: { getList: vi.fn() },
  notify: vi.fn(),
}))

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useTranslate: () => (key, opts) => opts?._ || key,
    useNotify: () => mocks.notify,
    useDataProvider: () => mocks.dataProvider,
  }
})

const fakeSongs = (count = 40) =>
  Array.from({ length: count }, (_, i) => ({
    id: `sg-${i}`,
    title: `Track ${i}`,
    artist: i % 2 ? 'Aurora Signal' : 'Still Water',
    artistId: i % 2 ? 'ar-1' : 'ar-2',
    album: `Album ${i % 4}`,
    albumId: `al-${i % 4}`,
    genre: ['Ambient', 'Electronic', 'Jazz', 'Pop'][i % 4],
    year: 1990 + (i % 30),
    duration: 150 + (i % 5) * 30,
    bpm: 70 + (i % 8) * 12,
    playCount: i % 7,
    playDate: i % 3 ? '2025-12-01T00:00:00Z' : null,
    rating: i % 5,
    starred: i % 9 === 0,
    starredAt: null,
    lyrics: '',
    missing: false,
    updatedAt: '2026-01-01T00:00:00Z',
  }))

const makeStore = () =>
  createStore(
    combineReducers({
      journey: journeyReducer,
      player: playerReducer,
    }),
  )

describe('JourneyHome', () => {
  beforeEach(() => {
    localStorage.clear()
    clearSongPoolCache()
    mocks.notify.mockClear()
    mocks.dataProvider.getList.mockReset()
    mocks.dataProvider.getList.mockImplementation((resource) => {
      if (resource === 'genre') return Promise.resolve({ data: [], total: 0 })
      return Promise.resolve({ data: fakeSongs(), total: 40 })
    })
  })

  afterEach(cleanup)

  it('renders the creator and starts a journey from the user library', async () => {
    const store = makeStore()
    render(
      <Provider store={store}>
        <JourneyHome />
      </Provider>,
    )

    expect(screen.getByText('Music Journey')).toBeInTheDocument()
    expect(screen.getByText(/Mood/)).toBeInTheDocument()
    const startButton = screen.getByRole('button', {
      name: /Start the journey/i,
    })
    fireEvent.click(startButton)

    await waitFor(() => {
      const state = store.getState()
      expect(state.journey.active?.journey?.tracks.length).toBeGreaterThan(0)
    })

    const state = store.getState()
    const journey = state.journey.active.journey
    // The existing player queue was filled with the journey tracks.
    expect(state.player.queue.length).toBe(journey.tracks.length)
    expect(state.player.queue.map((q) => q.trackId)).toEqual(
      journey.tracks.map((t) => t.track.id),
    )
    // Every queued track comes from the library the dataProvider returned.
    state.player.queue.forEach((q) => expect(q.trackId).toMatch(/^sg-/))

    // The live view took over (title appears in the player and in history).
    expect(
      await screen.findByText(/Now playing · Music Journey/i),
    ).toBeInTheDocument()
    expect(screen.getAllByText(journey.title).length).toBeGreaterThan(0)
    expect(mocks.notify).toHaveBeenCalled()
  })

  it('shows actionable guidance when the library has no matching tracks', async () => {
    mocks.dataProvider.getList.mockImplementation((resource) => {
      if (resource === 'genre') return Promise.resolve({ data: [], total: 0 })
      return Promise.resolve({ data: [], total: 0 })
    })
    const store = makeStore()
    render(
      <Provider store={store}>
        <JourneyHome />
      </Provider>,
    )

    fireEvent.click(screen.getByRole('button', { name: /Start the journey/i }))

    expect(
      await screen.findByText(/Unable to generate journey/i),
    ).toBeInTheDocument()
    // Empty pool surfaces the EMPTY_LIBRARY guidance.
    expect(screen.getByText(/looks empty right now/i)).toBeInTheDocument()
    expect(store.getState().journey.active).toBeNull()
  })

  it('offers Continue for the most recent journey from history', async () => {
    mocks.dataProvider.getList.mockImplementation((resource) => {
      if (resource === 'genre') return Promise.resolve({ data: [], total: 0 })
      return Promise.resolve({ data: fakeSongs(), total: 40 })
    })
    const store = makeStore()
    render(
      <Provider store={store}>
        <JourneyHome />
      </Provider>,
    )

    fireEvent.click(screen.getByRole('button', { name: /Start the journey/i }))
    await waitFor(() => expect(store.getState().journey.history.length).toBe(1))
    cleanup()

    // Re-open the page with the journey no longer active.
    store.dispatch({ type: 'JOURNEY_ENDED', data: { reason: 'stopped' } })
    render(
      <Provider store={store}>
        <JourneyHome />
      </Provider>,
    )
    const title = store.getState().journey.history[0].title
    expect(
      screen.getByRole('button', { name: /Continue the last journey/i }),
    ).toBeInTheDocument()
    expect(screen.getAllByText(title).length).toBeGreaterThan(0)
  })
})
