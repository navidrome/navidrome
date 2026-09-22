import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { DateField } from './DateField'

vi.mock('react-admin', async (importOriginal) => ({
  ...(await importOriginal()),
  useLocale: vi.fn(),
}))

describe('<DateField>', () => {
  const record = { id: '1', updatedAt: '2026-09-17T14:30:00Z' }

  beforeEach(async () => {
    vi.clearAllMocks()
    vi.spyOn(navigator, 'languages', 'get').mockReturnValue([])
    const { useLocale } = await import('react-admin')
    vi.mocked(useLocale).mockReturnValue('de')
  })

  it('formats the date using the selected language', () => {
    render(<DateField record={record} source="updatedAt" />)
    expect(screen.getByText('17.9.2026')).toBeInTheDocument()
  })

  it('renders nothing when the date is not set', () => {
    const { container } = render(
      <DateField record={{ id: '1' }} source="updatedAt" />,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
