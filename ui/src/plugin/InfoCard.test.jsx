import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'

vi.mock('../common', () => ({
  DateField: ({ source }) => <span data-testid={`date-${source}`} />,
}))

const { InfoCard } = await import('./InfoCard')

const record = {
  id: 'apple-music',
  path: '/data/plugins/apple-music.ndp',
  updatedAt: '2026-01-01T00:00:00Z',
  createdAt: '2026-01-01T00:00:00Z',
}

const renderCard = () =>
  render(
    <InfoCard
      record={record}
      manifest={{ name: 'Apple Music Metadata Agent' }}
      classes={{}}
      translate={(key) => key}
      isSmall={false}
    />,
  )

describe('InfoCard', () => {
  it('shows the plugin ID', () => {
    renderCard()
    expect(screen.getByText('apple-music')).toBeInTheDocument()
  })

  it('explains that the ID is the name used in config options', () => {
    renderCard()
    expect(
      screen.getByText('resources.plugin.messages.idHelp'),
    ).toBeInTheDocument()
  })
})
