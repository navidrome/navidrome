import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { RecordContextProvider } from 'react-admin'
import ArtistInfo from './ArtistInfo'

vi.mock('../common', async (importOriginal) => ({
  ...(await importOriginal()),
  ArtworkInfo: () => (
    <tr>
      <td>artwork-section</td>
    </tr>
  ),
}))

const renderWith = (record) =>
  render(
    <RecordContextProvider value={record}>
      <ArtistInfo />
    </RecordContextProvider>,
  )

describe('<ArtistInfo />', () => {
  it('renders the artist fields', () => {
    renderWith({ id: 'ar-1', name: 'Radiohead', albumCount: 9, songCount: 120 })
    expect(screen.getByText('Radiohead')).toBeInTheDocument()
    expect(screen.getByText('9')).toBeInTheDocument()
  })

  it('drops optional fields that are empty', () => {
    renderWith({ id: 'ar-1', name: 'Radiohead' })
    expect(screen.queryByText(/MusicBrainz/i)).toBeNull()
  })

  it('renders the artwork section', () => {
    renderWith({ id: 'ar-1', name: 'Radiohead' })
    expect(screen.getByText('artwork-section')).toBeInTheDocument()
  })
})
