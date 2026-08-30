import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { TestContext } from 'ra-test'
import ArtistExternalLinks from './ArtistExternalLink'

const { mockConfig } = vi.hoisted(() => ({
  mockConfig: { lastFMEnabled: true },
}))
vi.mock('../config', () => ({ default: mockConfig }))

describe('ArtistExternalLinks', () => {
  beforeEach(() => {
    mockConfig.lastFMEnabled = true
  })

  const renderLinks = (artistInfo, record = { id: 'ar-1', name: 'Björk' }) =>
    render(
      <TestContext>
        <ArtistExternalLinks artistInfo={artistInfo} record={record} />
      </TestContext>,
    )

  const lastFmHref = () =>
    screen.getByLabelText('message.openIn.lastfm').closest('a').href

  it('uses the URL returned by the server', () => {
    renderLinks({ lastFmUrl: 'https://www.last.fm/music/Bjork' })
    expect(lastFmHref()).toBe('https://www.last.fm/music/Bjork')
  })

  it('uses the URL found in the biography', () => {
    renderLinks({
      biography: 'Read more on <a href="https://www.last.fm/music/Bjork">',
      lastFmUrl: 'https://bjork.com',
    })
    expect(lastFmHref()).toBe('https://www.last.fm/music/Bjork')
  })

  it('builds the URL from the artist name when the server has none', () => {
    renderLinks({ lastFmUrl: 'https://bjork.com' })
    expect(lastFmHref()).toBe('https://last.fm/music/Bj%C3%B6rk')
  })

  it('builds the URL when there is no artist info at all', () => {
    renderLinks(undefined)
    expect(lastFmHref()).toBe('https://last.fm/music/Bj%C3%B6rk')
  })

  it('shows no Last.fm link when Last.fm is disabled', () => {
    mockConfig.lastFMEnabled = false
    renderLinks({ lastFmUrl: 'https://www.last.fm/music/Bjork' })
    expect(screen.queryByLabelText('message.openIn.lastfm')).toBeNull()
  })

  it('shows no Last.fm link when the artist has no name', () => {
    renderLinks({}, { id: 'ar-1', name: '' })
    expect(screen.queryByLabelText('message.openIn.lastfm')).toBeNull()
  })
})
