import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../subsonic', () => ({
  default: { createPodcastChannel: vi.fn().mockResolvedValue({}) },
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
  }
})

import subsonic from '../subsonic'
import PodcastCreate from './PodcastCreate'

describe('PodcastCreate', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renders a URL input field', () => {
    render(<PodcastCreate />)
    expect(screen.getByRole('textbox')).toBeTruthy()
  })

  it('calls createPodcastChannel with the entered URL on submit', async () => {
    render(<PodcastCreate />)
    fireEvent.change(screen.getByRole('textbox'), {
      target: { value: 'https://example.com/feed.xml' },
    })
    fireEvent.submit(screen.getByRole('form'))
    await waitFor(() => {
      expect(subsonic.createPodcastChannel).toHaveBeenCalledWith(
        'https://example.com/feed.xml',
      )
    })
  })

  it('redirects to /podcast after successful submit', async () => {
    render(<PodcastCreate />)
    fireEvent.change(screen.getByRole('textbox'), {
      target: { value: 'https://example.com/feed.xml' },
    })
    fireEvent.submit(screen.getByRole('form'))
    await waitFor(() => {
      expect(mockRedirect).toHaveBeenCalledWith('/podcast')
    })
  })

  it('notifies on success', async () => {
    render(<PodcastCreate />)
    fireEvent.change(screen.getByRole('textbox'), {
      target: { value: 'https://example.com/feed.xml' },
    })
    fireEvent.submit(screen.getByRole('form'))
    await waitFor(() => {
      expect(mockNotify).toHaveBeenCalledWith(
        'resources.podcast.notifications.channelAdded',
      )
    })
  })
})
