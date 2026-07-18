import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { PlaylistLove } from './PlaylistList'

vi.mock('../config', () => ({
  default: { enableFavourites: true },
}))

vi.mock('../common', () => ({
  LoveButton: ({ record, resource }) => (
    <button data-testid="love" data-resource={resource}>
      {record?.starred ? 'starred' : 'not-starred'}
    </button>
  ),
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
