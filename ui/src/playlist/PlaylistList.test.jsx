import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TestContext } from 'ra-test'
import { RecordContextProvider } from 'react-admin'
import { PlaylistLove, ToggleField, ToggleAutoImport } from './PlaylistList'

vi.mock('../config', () => ({
  default: { enableFavourites: true },
}))

vi.mock('../common', () => ({
  LoveButton: ({ record, resource }) => (
    <button data-testid="love" data-resource={resource}>
      {record?.starred ? 'starred' : 'not-starred'}
    </button>
  ),
  isWritable: (ownerId) => ownerId === 'me',
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
  it('<ToggleField /> renders nothing', () => {
    const { container } = render(
      <TestContext>
        <ToggleField resource="playlist" source="public" />
      </TestContext>,
    )
    expect(container.innerHTML).toBe('')
  })

  it('<ToggleAutoImport /> renders nothing', () => {
    const { container } = render(
      <TestContext>
        <ToggleAutoImport resource="playlist" source="sync" />
      </TestContext>,
    )
    expect(container.innerHTML).toBe('')
  })
})

// Secondary is a surface color in many themes, so these toggles must use primary
describe('<ToggleField />', () => {
  const renderToggle = (record) =>
    render(
      <TestContext>
        <RecordContextProvider value={record}>
          <ToggleField resource="playlist" source="public" />
        </RecordContextProvider>
      </TestContext>,
    )

  it.each([
    ['owner', 'me', false],
    ['non-owner', 'someone-else', true],
  ])('renders a primary-colored switch for the %s', (_, ownerId, disabled) => {
    renderToggle({ id: 'pl-1', public: true, ownerId })
    const input = screen.getByRole('checkbox')
    const switchBase = input.closest('.MuiSwitch-switchBase')
    expect(input.checked).toBe(true)
    expect(input.disabled).toBe(disabled)
    expect(switchBase.classList).toContain('MuiSwitch-colorPrimary')
    expect(switchBase.classList).not.toContain('MuiSwitch-colorSecondary')
  })
})
