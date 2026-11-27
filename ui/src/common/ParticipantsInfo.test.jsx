import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ParticipantsInfo } from './ParticipantsInfo'

vi.mock('react-admin', () => ({
  useTranslate: () => (key, opts) =>
    opts?.smart_count !== undefined ? `${key}:${opts.smart_count}` : key,
}))

vi.mock('./index', () => ({
  ArtistLinkField: ({ record, source }) => (
    <span>
      {record.participants[source]
        .map((a) => a.name + (a.subRole ? ` (${a.subRole})` : ''))
        .join(', ')}
    </span>
  ),
}))

const renderInfo = (participants) =>
  render(
    <table>
      <tbody>
        <ParticipantsInfo record={{ id: 'r1', participants }} />
      </tbody>
    </table>,
  )

const rowTexts = () =>
  screen
    .getAllByRole('row')
    .slice(1)
    .map((row) => row.textContent)

describe('ParticipantsInfo', () => {
  it('renders one row per performer sub-role', () => {
    renderInfo({
      performer: [
        { id: '1', name: 'John Ryan', subRole: 'Guitar' },
        { id: '1', name: 'John Ryan', subRole: 'Lead Vocals' },
        { id: '2', name: 'Pitbull', subRole: 'Lead Vocals' },
        { id: '3', name: 'Dan Boissy', subRole: 'Saxophone' },
      ],
    })

    expect(rowTexts()).toEqual([
      'resources.artist.roles.performer:1 (Guitar):John Ryan',
      'resources.artist.roles.performer:2 (Lead Vocals):John Ryan, Pitbull',
      'resources.artist.roles.performer:1 (Saxophone):Dan Boissy',
    ])
  })

  it('keeps artists without a sub-role in a plain row', () => {
    renderInfo({
      performer: [
        { id: '1', name: 'Plain Performer' },
        { id: '2', name: 'Guitarist', subRole: 'Guitar' },
      ],
      composer: [
        { id: '3', name: 'Composer A' },
        { id: '4', name: 'Composer B' },
      ],
    })

    expect(rowTexts()).toEqual([
      'resources.artist.roles.performer:1:Plain Performer',
      'resources.artist.roles.performer:1 (Guitar):Guitarist',
      'resources.artist.roles.composer:2:Composer A, Composer B',
    ])
  })

  it('skips artist and albumartist roles', () => {
    const { container } = renderInfo({
      artist: [{ id: '1', name: 'Main' }],
      albumartist: [{ id: '1', name: 'Main' }],
    })

    expect(container.querySelector('tr')).toBeNull()
  })
})
