import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PlaylistTypeIcon } from './PlaylistTypeIcon'

vi.mock('react-admin', () => ({
  useTranslate: () => (key) => key,
}))

describe('<PlaylistTypeIcon />', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('marks a playlist with rules as smart', () => {
    render(<PlaylistTypeIcon record={{ id: 'pl-1', rules: { all: [] } }} />)
    expect(
      screen.getByTitle('resources.playlist.message.smartPlaylist'),
    ).not.toBeNull()
  })

  it('uses the plain playlist icon when there are no rules', () => {
    render(<PlaylistTypeIcon record={{ id: 'pl-2', rules: null }} />)
    expect(screen.getByTitle('resources.playlist.name')).not.toBeNull()
    expect(
      screen.queryByTitle('resources.playlist.message.smartPlaylist'),
    ).toBeNull()
  })

  it('names the smart icon for screen readers', () => {
    // react-icons discards the <title> SvgIcon builds from titleAccess, so the
    // icon has to pass react-icons' own `title` prop as well
    const { container } = render(
      <PlaylistTypeIcon record={{ id: 'pl-3', rules: { all: [] } }} />,
    )
    const svg = container.querySelector('svg')
    expect(svg.getAttribute('role')).toBe('img')
    expect(svg.querySelector('title').textContent).toBe(
      'resources.playlist.message.smartPlaylist',
    )
  })
})
