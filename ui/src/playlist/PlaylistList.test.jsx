import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PlaylistLove, PlaylistNameField } from './PlaylistList'

vi.mock('../config', () => ({
  default: { enableFavourites: true },
}))

vi.mock('../common', () => ({
  LoveButton: ({ record, resource }) => (
    <button data-testid="love" data-resource={resource}>
      {record?.starred ? 'starred' : 'not-starred'}
    </button>
  ),
  SmartPlaylistIcon: () => <span data-testid="smart-icon" />,
  isSmartPlaylist: (pls) => !!pls.rules,
  isWritable: () => true,
}))

describe('<PlaylistLove />', () => {
  it('renders a LoveButton bound to the playlist resource', () => {
    render(<PlaylistLove record={{ id: 'pl-1', starred: true }} />)
    const btn = screen.getByTestId('love')
    expect(btn.getAttribute('data-resource')).toBe('playlist')
    expect(btn.textContent).toBe('starred')
  })

  it('exposes datagrid header props so the column renders unsorted', () => {
    // The Datagrid reads these off the element; the wrapper body must not
    // forward them to the button (which would leak onto the DOM).
    expect(PlaylistLove.defaultProps).toEqual({
      source: 'starred',
      sortable: false,
    })
  })
})

describe('<PlaylistNameField />', () => {
  it('flags a smart playlist next to its name', () => {
    render(
      <PlaylistNameField
        record={{ id: 'pl-1', name: 'Top Rock', rules: { all: [] } }}
      />,
    )
    expect(screen.getByText('Top Rock')).not.toBeNull()
    expect(screen.getByTestId('smart-icon')).not.toBeNull()
  })

  it('shows no flag for a hand-picked playlist', () => {
    render(<PlaylistNameField record={{ id: 'pl-2', name: 'Road Trip' }} />)
    expect(screen.getByText('Road Trip')).not.toBeNull()
    expect(screen.queryByTestId('smart-icon')).toBeNull()
  })

  it('renders nothing without a record', () => {
    const { container } = render(<PlaylistNameField />)
    expect(container.innerHTML).toBe('')
  })

  it('exposes the source so the datagrid keeps a sortable Name column', () => {
    expect(PlaylistNameField.defaultProps).toEqual({ source: 'name' })
  })
})
