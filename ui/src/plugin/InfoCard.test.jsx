import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'

vi.mock('react-admin', async () => {
  const actual = await vi.importActual('react-admin')
  return {
    ...actual,
    useTranslate: vi.fn(() => (key) => key),
  }
})

vi.mock('../common', () => ({
  DateField: ({ source }) => <span data-testid={`date-${source}`} />,
}))

const { InfoCard } = await import('./InfoCard')

const classes = {
  section: '',
  sectionTitle: '',
  infoGrid: '',
  infoLabel: '',
  pathField: '',
  permissionChip: '',
  permissionsContainer: '',
  tooltipContent: '',
}

const record = {
  id: 'apple-music',
  path: '/data/plugins/apple-music.ndp',
  updatedAt: '2026-01-01T00:00:00Z',
  createdAt: '2026-01-01T00:00:00Z',
}

const renderCard = (manifest = { name: 'Apple Music Metadata Agent' }) =>
  render(
    <InfoCard
      record={record}
      manifest={manifest}
      classes={classes}
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
