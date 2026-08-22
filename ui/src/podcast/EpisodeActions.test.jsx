import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../subsonic', () => ({
  default: {
    downloadPodcastEpisode: vi.fn().mockResolvedValue({}),
    deletePodcastEpisode: vi.fn().mockResolvedValue({}),
  },
}))

vi.mock('react-redux', () => ({
  useDispatch: () => vi.fn(),
}))

import subsonic from '../subsonic'
import EpisodeActions from './EpisodeActions'

describe('EpisodeActions', () => {
  const onRefresh = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows play and delete buttons for completed episode', () => {
    render(
      <EpisodeActions episode={{ id: 'ep-1', status: 'completed' }} onRefresh={onRefresh} />,
    )
    expect(screen.getByLabelText('play')).toBeTruthy()
    expect(screen.getByLabelText('delete')).toBeTruthy()
    expect(screen.queryByLabelText('download')).toBeNull()
  })

  it('shows download and delete buttons for new episode', () => {
    render(
      <EpisodeActions episode={{ id: 'ep-1', status: 'new' }} onRefresh={onRefresh} />,
    )
    expect(screen.getByLabelText('download')).toBeTruthy()
    expect(screen.getByLabelText('delete')).toBeTruthy()
    expect(screen.queryByLabelText('play')).toBeNull()
  })

  it('shows download and delete buttons for error episode', () => {
    render(
      <EpisodeActions episode={{ id: 'ep-1', status: 'error' }} onRefresh={onRefresh} />,
    )
    expect(screen.getByLabelText('download')).toBeTruthy()
    expect(screen.getByLabelText('delete')).toBeTruthy()
  })

  it('shows spinner only for downloading episode', () => {
    render(
      <EpisodeActions episode={{ id: 'ep-1', status: 'downloading' }} onRefresh={onRefresh} />,
    )
    expect(screen.getByRole('progressbar')).toBeTruthy()
    expect(screen.queryByLabelText('play')).toBeNull()
    expect(screen.queryByLabelText('download')).toBeNull()
  })

  it('renders nothing for deleted episode', () => {
    const { container } = render(
      <EpisodeActions episode={{ id: 'ep-1', status: 'deleted' }} onRefresh={onRefresh} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('calls downloadPodcastEpisode when download clicked', () => {
    render(
      <EpisodeActions episode={{ id: 'ep-1', status: 'new' }} onRefresh={onRefresh} />,
    )
    fireEvent.click(screen.getByLabelText('download'))
    expect(subsonic.downloadPodcastEpisode).toHaveBeenCalledWith('ep-1')
  })

  it('calls deletePodcastEpisode when delete clicked on completed episode', () => {
    render(
      <EpisodeActions episode={{ id: 'ep-1', status: 'completed' }} onRefresh={onRefresh} />,
    )
    fireEvent.click(screen.getByLabelText('delete'))
    expect(subsonic.deletePodcastEpisode).toHaveBeenCalledWith('ep-1')
  })
})
