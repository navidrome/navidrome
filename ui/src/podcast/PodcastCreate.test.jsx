import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../subsonic', () => ({
  default: {
    createPodcastChannel: vi.fn().mockResolvedValue({}),
    previewPodcastFeed: vi.fn().mockResolvedValue({
      json: {
        title: 'Example Podcast',
        description: 'An example feed',
        episodeCount: 3,
        alreadyExists: false,
      },
    }),
  },
}))

const mockRedirect = vi.fn()
const mockRefresh = vi.fn()
const mockNotify = vi.fn()

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useTranslate: () => (key) => key,
    useNotify: () => mockNotify,
    useRedirect: () => mockRedirect,
    useRefresh: () => mockRefresh,
    Title: () => null,
    Button: ({ children, onClick, label, disabled }) => (
      <button aria-label={label} onClick={onClick} disabled={disabled}>
        {children}
      </button>
    ),
  }
})

import subsonic from '../subsonic'
import PodcastCreate from './PodcastCreate'

const fetchPreview = async (url) => {
  fireEvent.change(screen.getByRole('textbox'), { target: { value: url } })
  fireEvent.click(
    screen.getByLabelText('resources.podcast.actions.fetchFeed'),
  )
  await waitFor(() => {
    expect(subsonic.previewPodcastFeed).toHaveBeenCalledWith(url)
  })
}

describe('PodcastCreate', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renders a URL input field', () => {
    render(<PodcastCreate />)
    expect(screen.getByRole('textbox')).toBeTruthy()
  })

  it('fetches a preview of the feed for the entered URL', async () => {
    render(<PodcastCreate />)
    await fetchPreview('https://example.com/feed.xml')
    expect(await screen.findByText('Example Podcast')).toBeTruthy()
  })

  it('calls createPodcastChannel with the entered URL when adding the previewed channel', async () => {
    render(<PodcastCreate />)
    await fetchPreview('https://example.com/feed.xml')
    fireEvent.click(
      await screen.findByLabelText('resources.podcast.actions.addChannel'),
    )
    await waitFor(() => {
      expect(subsonic.createPodcastChannel).toHaveBeenCalledWith(
        'https://example.com/feed.xml',
      )
    })
  })

  it('redirects to /podcast after successfully adding the channel', async () => {
    render(<PodcastCreate />)
    await fetchPreview('https://example.com/feed.xml')
    fireEvent.click(
      await screen.findByLabelText('resources.podcast.actions.addChannel'),
    )
    await waitFor(() => {
      expect(mockRedirect).toHaveBeenCalledWith('/podcast')
    })
  })

  it('notifies on success', async () => {
    render(<PodcastCreate />)
    await fetchPreview('https://example.com/feed.xml')
    fireEvent.click(
      await screen.findByLabelText('resources.podcast.actions.addChannel'),
    )
    await waitFor(() => {
      expect(mockNotify).toHaveBeenCalledWith(
        'resources.podcast.notifications.channelAdded',
      )
    })
  })
})
