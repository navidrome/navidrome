import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import {
  PlaylistLove,
  TogglePublicInput,
  ToggleAutoImport,
} from './PlaylistList'

vi.mock('../config', () => ({
  default: { enableFavourites: true },
}))

vi.mock('react-admin', async (importOriginal) => ({
  ...(await importOriginal()),
  useRecordContext: () => undefined,
  useUpdate: () => [vi.fn()],
  useNotify: () => vi.fn(),
}))

vi.mock('../common', () => ({
  isWritable: () => true,
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

// react-admin evicts records older than 10 minutes while the list still holds
// their ids, so rows can render with no record.
describe('playlist toggles without a record', () => {
  it('<TogglePublicInput /> renders nothing', () => {
    const { container } = render(
      <TogglePublicInput resource="playlist" source="public" />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('<ToggleAutoImport /> renders nothing', () => {
    const { container } = render(
      <ToggleAutoImport resource="playlist" source="sync" />,
    )
    expect(container.innerHTML).toBe('')
  })
})
