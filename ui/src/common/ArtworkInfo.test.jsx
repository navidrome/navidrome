import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ArtworkInfo } from './ArtworkInfo'

const explainArtwork = vi.fn()
const { mockPermissions } = vi.hoisted(() => ({
  mockPermissions: { value: 'admin' },
}))

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useDataProvider: () => ({ explainArtwork }),
    usePermissions: () => ({ permissions: mockPermissions.value }),
    useTranslate: () => (key) => key,
  }
})

const report = {
  result: 'resolved from external:deezer',
  chainOrigin: 'recorded 2026-09-01T10:00:00Z',
  stored: { source: 'external:deezer', attemptedAt: '2026-09-01T10:00:00Z' },
  steps: [
    { candidate: 'cover.*', outcome: 'miss' },
    {
      candidate: 'external:deezer',
      outcome: 'hit',
      detail: 'https://cdn/x.jpg',
    },
  ],
  config: { setting: 'ArtistArtPriority', value: 'external' },
}

// ArtworkInfo renders bare TableRows to slot into the caller's TableBody, so it needs a
// table/tbody ancestor or React logs DOM-nesting warnings.
const renderInTable = (ui) =>
  render(
    <table>
      <tbody>{ui}</tbody>
    </table>,
  )

describe('<ArtworkInfo />', () => {
  beforeEach(() => {
    mockPermissions.value = 'admin'
    explainArtwork.mockReset()
    explainArtwork.mockResolvedValue({ data: report })
  })

  it('renders nothing for a non-admin', () => {
    mockPermissions.value = 'regular'
    const { container } = renderInTable(
      <ArtworkInfo resource="artist" id="ar-1" />,
    )
    expect(container.querySelector('tbody')).toBeEmptyDOMElement()
    expect(explainArtwork).not.toHaveBeenCalled()
  })

  it('shows the summary for an admin', async () => {
    renderInTable(<ArtworkInfo resource="artist" id="ar-1" />)
    expect(
      await screen.findByText('resolved from external:deezer'),
    ).toBeInTheDocument()
    expect(screen.queryByText('external:deezer')).not.toBeNull()
    expect(screen.queryByText('cover.*')).toBeNull()
  })

  it('reveals the step table when details are expanded', async () => {
    renderInTable(<ArtworkInfo resource="artist" id="ar-1" />)
    await screen.findByText('resolved from external:deezer')
    await userEvent.click(screen.getByText('artwork.showDetails'))
    expect(screen.getByText('cover.*')).toBeInTheDocument()
    expect(screen.getByText('ArtistArtPriority:')).toBeInTheDocument()
  })

  it('shows the not-recorded state', async () => {
    explainArtwork.mockResolvedValue({
      data: { result: 'not resolved', chainOrigin: 'not recorded', steps: [] },
    })
    renderInTable(<ArtworkInfo resource="artist" id="ar-1" />)
    expect(await screen.findByText('artwork.notRecorded')).toBeInTheDocument()

    // stored/queued/config/agents are all omitted by the endpoint here, so the optional
    // blocks' guards must not throw when expanded.
    await userEvent.click(screen.getByText('artwork.showDetails'))
    expect(screen.queryByText('artwork.priority')).toBeNull()
    expect(screen.queryByText('artwork.agents')).toBeNull()
  })

  it('renders nothing when the fetch fails', async () => {
    explainArtwork.mockRejectedValue(new Error('boom'))
    const { container } = renderInTable(
      <ArtworkInfo resource="artist" id="ar-1" />,
    )
    await waitFor(() => expect(explainArtwork).toHaveBeenCalled())
    expect(container.querySelector('tbody')).toBeEmptyDOMElement()
  })
})
